---
title: "resilience Devlog #4 — Building multimod"
description: "Three rewrites in one session. From Issue/Fixer/Runner to Discovery → desired State → Applier. How Terraform thinking killed our diff-based architecture."
head:
  - - meta
    - name: keywords
      content: go multi-module tool, golang workspace automation, terraform pattern, desired state, functional pipeline, go.work management
---

# #4 — Building multimod

> "Why compute current state when you can always upsert the desired one?"

## Three architectures in one day

### Architecture 1: Issue/Fixer/Runner

Started classically. `Invariant` finds problems, creates `Issue`. Issue knows how to fix itself via `Fix(state)`. Fixer orchestrates the cycle: analyze → fix → verify.

Problems:
- Issue reaches into State, mutates modfile directly, writes to disk
- Partial writes: if the third Issue is unfixable, the first two already wrote to disk
- `modfile.File` leaks through the entire domain
- Fixer — God Object with two responsibilities (analyze + fix)

We tried to fix it: encapsulate State, add dirty tracking, defer Flush. But dirty tracking mixed two entity types (modules vs go.work) in one map. Magic string keys. The deeper we dug, the worse it got.

### Architecture 2: Rule/Op/Applier

Inspired by `golang.org/x/tools/go/analysis`. Rule — bounded context: detection + fix in one package. Op — operation with priority. Applier — DSL with two methods.

Progress: Op doesn't know about State, only Applier DSL. Unfixable ops (priority 0) float to the top — atomicity through sorting.

But then we asked: "What is an invariant? What is an enricher? How are they different?"

Answer: both are `func(State) (State, error)`. A step in a pipeline.

### Architecture 3: Discovery → desired State → Applier

The breakthrough: **why compute a diff when you can declare desired state?**

Terraform doesn't compare "what is" with "what should be" at the user level. The user declares desired. `terraform apply` makes reality match.

Same for us:
- Discovery reads FS, builds the model, validates (acyclic graph), enriches (go versions, replaces, workspace)
- Output: desired State. Complete. Valid. Or error.
- Applier receives desired State and makes FS match it. Idempotent.

Issue, Fixer, Runner, Op, Rule — all gone. Two steps: Discovery → Applier.

## Key decisions

### State as value type, FP pipeline

```go
type Step func(State) (State, error)
```

Each step receives State by value, returns a new one. Immutability. With 300 sub-modules — ~400 KB for the entire pipeline. Free for a CLI tool.

Pipeline: Parse → ValidateAcyclic → EnrichGoVersion → EnrichReplaces → EnrichWorkspace.

New step = new file in `steps/` + one line in `NewDefaultDiscovery()`. Core doesn't change.

### modfile stays inside boundaries

Discovery: `modfile.Parse → Module{Path, Dir, GoVersion, Requires, Replaces}`. modfile dies inside the Parse step.

Applier: `State → modfile.Parse → mutate → write`. modfile is born inside Applier.

Between them — pure domain model. Zero `*modfile.File` in domain types.

### Unconditional replaces

Chicken-and-egg problem: `go mod tidy` downloads internal module from registry because replace doesn't exist yet (no require → no replace).

Solution: **every sub-module gets replace for all other project modules. Unconditionally.** Unused replaces — Go ignores them. But when `go mod tidy` adds require — replace is already in place. No ambiguous import. No registry fetch.

### Always Apply

Any use of multimod = guaranteed synced FS. Apply runs in bootstrap, before any command. You cannot forget `apply`. You cannot end up in an inconsistent state.

```
multimod go test ./...
  1. Boot → cwd is root (go.mod must exist)
  2. Discovery → desired State (or die)
  3. Apply(State) → FS synced (idempotent)
  4. go test in each module
```

### graph/ — pure algorithm

DFS cycle detection — math. Knows nothing about multimod, State, Module. Tested in isolation. Like `backoff.Exponential` — an algorithm, not an integration.

### Domain model without infrastructure leakage

```go
type Module struct {
    Path      string
    Dir       AbsDir
    GoVersion string
    Requires  []string
    Replaces  []string
}
```

No `*modfile.File`. No `*modfile.WorkFile`. No FS handles. Pure data. Applier figures out how to write it. Discovery figures out how to read it. The model doesn't care.

## What we built

```
pkg/multimod/
├── model.go                    ← Module, State, AbsDir (pure domain)
├── graph/cycle.go              ← DetectCycle (pure algorithm)
├── discovery/                  ← FS → desired State (pipeline)
│   ├── steps/parse.go          ← modfile dies here
│   ├── steps/validate_acyclic.go
│   ├── steps/enrich_*.go
│   └── discovery.go            ← pipeline runner
├── applier/applier.go          ← State → FS (modfile born here)
├── testutil/scaffold.go
└── testdata/ (11 scenarios)

cmd/multimod/
├── multimod.go                 ← Boot → Discovery → Apply → CLI
├── boot.go, root.go, cmds.go
└── cmds/go.go                  ← transparent proxy
```

22 tests. Lint green. Build green. Zero-config.

## The pattern continues

| Problem | Status | Primitive |
|---|---|---|
| Resilience patterns | ✅ Extracted | `func(ctx, call) error` |
| Task runner lifecycle | 📝 Documented | `BeforeAll` / phases |
| Multi-module management | ✅ Built | `Discovery → State → Applier` |

---

*The tool that died three times and came back cleaner each time.*
