---
title: "multimod Specification — Architecture for Go Multi-Module Management"
description: "Full technical specification: Boot, Kernel, Discovery, Fixer pipeline. State contract, verify algorithm, release workflow for Go multi-module monorepos."
head:
  - - meta
    - name: keywords
      content: go multi-module architecture, go monorepo tool design, go.work automation, go mod replace strip, go sub-module versioning, golang workspace management
---

# Specification

## Architecture

```
Boot → Kernel → Discovery → Executor → Runner → Command
```

### Boot

Finds the project root (walks up directories looking for `go.mod`). Determines if this is a multi-module project (sub-directories with their own `go.mod`). If not — transparent proxy to `go`, multimod is invisible.

### Kernel

Receives root path from Boot. Calls Discovery to build State. Passes State to Executor. Two responsibilities, nothing more.

### Discovery

Reads the filesystem. Finds all `go.mod` files (excluding `_tools/`, `vendor/`, `testdata/`). Parses each via `golang.org/x/mod/modfile`. Builds the dependency graph from `require` directives. Returns raw State — no judgments about correctness.

### Executor

Resolves the command category and delegates to the appropriate Runner:

- `go`, `verify`, `generate` → **FixableRunner**
- `release` → **UnfixableRunner**

### FixableRunner

```
Pre-invariant:  Fixer checks and fixes State (atomic: disk + model)
Execute:        Command runs with valid State
Post-invariant: Fixer checks State again (Command may have changed files)
```

### UnfixableRunner

```
Validate: State must already be consistent. If not — die with error.
Execute:  ReleaseCommand transforms dev → publish state.
```

Used for `release`. The assumption: you're in CI, dev-state was verified earlier.

### Fixer

Compares "what is" with "what should be". Fixes what it can, atomically. Uses Writer internally. Returns fixed State or error.

**Atomicity contract:** either everything is fixed (disk + model) or nothing is touched.

### Command

Receives valid State. Trusts it completely. Does its work.

## State contract

Valid State means:

1. Root module exists and is parsed
2. All sub-modules discovered and parsed
3. Dependency graph is acyclic (DAG)
4. Every sub-module has `replace` for all internal deps
5. `go.work` contains all modules (no missing, no extra)
6. `go` directive is the same everywhere (root = source of truth)

## Verify + Fix algorithm

```
1. FIND      — all go.mod files
2. PARSE     — modfile.Parse() each
3. CLASSIFY  — root, subs, internal deps graph
4. CHECK     — go.work, replaces, go directive, graph
5. FIX       — atomically (memory + disk) or error
6. VERIFY    — re-run checks, issues must be empty
```

### Issues

| Issue | Fixable | Action |
|---|---|---|
| MISSING_GOWORK | ✅ | Create go.work with all modules |
| GOWORK_MISSING_MODULE | ✅ | Add to go.work |
| GOWORK_EXTRA_MODULE | ✅ | Remove from go.work |
| MISSING_REPLACE | ✅ | Add replace with computed relative path |
| GO_VERSION_MISMATCH | ✅ | Sync to root's go directive |
| CYCLIC_DEPENDENCY | ❌ | Error: architecture issue, suggest extraction |
| CORRUPTED_GOMOD | ❌ | Discovery fails, Kernel stops |

## Commands

### `multimod go <args>`

Transparent proxy with multi-module awareness.

Multi-module commands (`test`, `vet`, `build`, `mod tidy` with `./...`): iterate all modules, aggregate results.

Everything else: `syscall.Exec("go", args)` — direct passthrough.

Post-invariant runs after execution (e.g. `go mod tidy` may change go directive — Fixer reverts).

### `multimod release <version>`

CI command. Transforms dev-state → publish-state.

1. Validate: State must be consistent (UnfixableRunner — no auto-fix)
2. Transform in memory: strip `replace`, pin `require` to version, skip self
3. Self-check: no local replace remaining, all require pinned, no self-dependency
4. Create tags: core `v1.2.3` + sub-module `<dir>/v1.2.3`

Default: dry-run (print what would change). `--write`: apply changes.

Idempotent: safe to re-run if interrupted.

### `multimod generate`

Project model → templates → files.

Templates live in `.multimod/templates/`. Convention:
- `dependabot.yml.tmpl` → one file `dependabot.yml`
- `module-ci.yml.tmpl` → N files `module-ci_<dir>.yml` (one per sub-module)

Engine: Go `text/template`. Model: root, modules, go_version, dependency graph.

Default: dry-run (show diff). `--write`: write files.

### `multimod verify`

Runs the full pipeline (Boot → Discovery → Fixer). Prints State and what was fixed.

CI pattern:
```bash
multimod verify --write
git diff --quiet || (echo "Inconsistent state. Run multimod locally." && exit 1)
```

## IO

- **stdout** — belongs to Go. Never touched by multimod.
- **stderr** — multimod's channel. `[multimod]` prefix. Silent when everything is ok.

## Delegation to Go

| Task | Who |
|---|---|
| Parse go.mod | `golang.org/x/mod/modfile` (library) |
| Discover modules | multimod (`find` + `modfile.Parse`) |
| Auto-add replace | multimod (knows internal deps from graph) |
| Sync go.work | multimod |
| Sync go directive | multimod |
| Validate replace paths | Go (`go mod tidy` at exec time) |
| Detect import cycles | Go (`go build` at exec time) |
| Strip replace (publish) | multimod (`modfile`) |
| Pin require (publish) | multimod (`modfile`) |
| Git tags | multimod (`git tag`) |
| Code errors | Go (passthrough, not parsed) |

## Requirements

- Linux / macOS (Windows via WSL)
- Git
- Core module in root, sub-modules in subdirectories
- Sub-modules depend on core, not reverse (DAG)
- One version for all modules at release time
