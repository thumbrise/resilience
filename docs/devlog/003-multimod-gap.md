---
title: "Why Go Has No Multi-Module Release Tool — And What We're Building"
description: "Every Go project with sub-modules writes its own release script. We analyzed Rust, Node, Java, Elixir, Python — and built multimod. The full story."
head:
  - - meta
    - name: keywords
      content: go multi-module release, golang monorepo problem, go.work replace publish, go mod replace strip, cargo-release vs go, npm changesets vs go, otel multimod alternative, go sub-module tagging
---

# #3 — The Multi-Module Gap

> "Every Go project with sub-modules writes its own release script. Nobody asks why."

## The Pain

resilience needed multi-module. Core with zero dependencies, OTEL plugin in a separate module so users don't pull the OTEL SDK unless they want it. Simple requirement. Every mature library in every ecosystem does this.

We created separate `go.mod` files. Then everything broke.

`go test ./...` stopped testing sub-modules. `go.work` needed to exist but couldn't be committed. Every sub-module needed `replace` directives for local development that had to be stripped before publishing. Sub-modules needed prefixed git tags (`otel/v1.2.3`). Dependabot needed a separate entry per module. CI needed to iterate modules.

We wrote shell scripts. grep over JSON inside YAML. 20 lines of fragile string manipulation to do what Cargo does in zero lines. It worked. Then it broke — added a self-dependency to every module because grep matched `Module.Path` in `go mod edit -json` output.

## The Research

Before building anything, we checked what exists.

### Other Ecosystems

We asked friends from other language communities. The responses were... educational.

**Rust friend:** "Cargo workspace. `cargo-release`. Done." Dev overrides don't leak into published crates — Cargo strips them automatically.

**Node friend:** "`npm workspaces` + `changesets`. One changeset file, merge, CI publishes everything." 10k+ stars on GitHub. Community standard.

**Java friend:** "`mvn release:prepare release:perform`. Works since 2005." Twenty years of polish.

**Elixir friend:** "`in_umbrella: true` becomes a version number on publish. Automatically." The exact transformation we do with grep.

**Python friend:** "`pip install mylib[otel]`. Optional dependencies. No separate module needed."

Every ecosystem solved this. Most solved it at the package manager level. Go is the only major language where multi-module monorepo has no tooling support.

### Go Ecosystem

| Tool | What it does | What it doesn't |
|---|---|---|
| OTEL `multimod` | verify, prerelease, tag | Requires manual config, hardcoded for OTEL, not reusable |
| `kimono` | Module discovery, tag coordination | Young, unclear on replace/require |
| AWS tools | `updaterequires` for inter-module versions | Internal scripts, not designed for reuse |
| `release-please` | Tags and changelogs | Doesn't understand Go `replace` |
| `goreleaser` | Binary packaging | For CLIs, not libraries |
| `gorelease` | API compatibility checks | Orthogonal — "what" not "how" |

No tool fills all the columns. The gap is real and confirmed from multiple angles.

### Industry Strategy

The largest Go projects avoid multi-module entirely:

- **Google** uses Bazel instead of Go modules
- **Uber and HashiCorp** use single-module repos with `internal/`
- **Kubernetes** has a publishing bot — 10,000 lines of infrastructure

The absence of tooling causes the absence of multi-module projects, not the other way around.

## The Design

Same method as resilience itself: separate objective from subjective, find the minimal primitive, design for extraction.

### Objective problems (every multi-module project has these)

1. **Discovery** — find all `go.mod`, build dependency graph
2. **Integrity** — replace directives valid, go.work synced, go directive consistent
3. **Publish transform** — dev state → publish state (strip replace, pin require, tag)
4. **Template generation** — project model → any file (CI, dependabot, whatever)

### Subjective choices (each project decides differently)

- Which CI system
- Which release tool (semantic-release, release-please, manual)
- Which files to generate
- When to release

multimod handles the objective. Users handle the subjective.

### Architecture

```
Boot → Kernel → Discovery → Executor → Runner → Command
```

Four public commands:
```
multimod go <args>          — daily work, transparent proxy
multimod release <version>  — CI, publish preparation
multimod generate           — templates → files
multimod verify             — check + auto-fix state
```

Key insight: multimod is not a shim for `go`. It's a **workspace-aware orchestrator** that knows it's a multi-module project. When you run `multimod go test ./...`, it guarantees workspace is ready, iterates all modules, aggregates results. When you run a non-multi-module command, it passes through to `go` transparently.

stdout belongs to Go (gopls parses it). stderr belongs to multimod (Unix convention — programs don't parse others' stderr).

Full specification: [docs/internals/multimod/spec](/internals/multimod/spec).

## The Decision

multimod lives inside resilience first. `pkg/multimod/` — battle-tested on our own pain. When stable — extracted into `thumbrise/multimod` with its own lifecycle.

Same pattern as resilience itself:
- `pkg/longrun` inside autosolve → killed → `resilience` extracted
- `release:pin` inside resilience → killed → `multimod` building

## The Pattern

Tools emerge from tools. ClickHouse from Yandex internals. MapReduce from Google's data processing. resilience from autosolve's retry needs. multimod from resilience's release needs.

The deathbook grows:

| Problem | Status | Primitive |
|---|---|---|
| Resilience patterns in Go | ✅ Extracted | `func(ctx, call) error` |
| Task runner lifecycle | 📝 Documented | `BeforeAll` / lifecycle phases |
| Multi-module release | 🔨 Building | multimod |

Each entry: concrete pain → honest research → minimal primitive → extract when ready.

Because the only honest response to "there's no tool for this" is to build one.

---

*The tool that will outgrow its parent. Again.*
