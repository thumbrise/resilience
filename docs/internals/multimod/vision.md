---
title: "multimod Roadmap — From Internal Tool to Go Ecosystem Standard"
description: "Roadmap for multimod: battle-test inside resilience, stabilize API, extract to standalone repo. The missing cargo-release for Go."
head:
  - - meta
    - name: keywords
      content: go multi-module roadmap, golang monorepo tool, cargo-release for go, changesets for go, go module release automation
---

# Vision

## The Pattern

Tools emerge from tools:

1. **longrun** (1500-line framework) → killed → **resilience** (`func(ctx, call) error`)
2. **Task runner lifecycle gap** → documented → waiting for someone to build `BeforeAll`
3. **Release shell scripts** (grep over JSON in YAML) → killed → **multimod**

Each time: concrete pain → honest research → minimal primitive → extract when ready.

## Phases

### Phase 1: Internal (current)

Lives in `pkg/multimod/` inside resilience. Battle-tested on resilience itself.

- ✅ Boot — cwd is root, no traversal, .git warning
- ✅ Discovery pipeline — Parse, ValidateAcyclic, Enrich*
- ✅ Applier — sync go.work, go.mod, replaces, go version
- ✅ CLI — `multimod go <args>` proxy with multi-module iteration
- Implement `release` command — detached commit + dev/prod tags
- Implement `generate` — templates in `.multimod/templates/`
- Enable full multi-module in resilience (separate go.mod per sub-package)

**Success criteria:** resilience uses multimod for all multi-module operations. Zero shell scripts for release.

### Phase 2: Stabilize

Still inside resilience. API hardening.

- CI examples (GitHub Actions workflows)
- Template convention finalized
- Error messages polished
- Edge cases covered (empty projects, single module, nested modules)
- `multimod modules` — JSON output for external tools
- Tests: real temp directories, real git repos

**Success criteria:** another project (internal) uses multimod successfully.

### Phase 3: Extract

Standalone repository `thumbrise/multimod`. Own release cycle. Semver.

- `go install github.com/thumbrise/multimod@latest`
- README, docs, examples
- Community feedback
- First stable release `v1.0.0`

**Success criteria:** strangers use it and don't file bugs about basic operations.

## Non-goals

- **Not a build system** — we don't replace `go build`
- **Not a package manager** — we don't replace `go get`
- **Not a CI tool** — we give you the model, you write the pipeline
- **Not a monorepo framework** — we solve Go multi-module, nothing else
- **Not cross-language** — Go only

## The Bet

The Go ecosystem will grow more multi-module projects as the language matures. Libraries with optional integrations (OTEL, gRPC, Redis) need isolated dependencies. The tooling gap will become more painful, not less.

multimod bets that a zero-config, convention-based tool can fill this gap — the same way `cargo-release` filled it for Rust and `changesets` filled it for Node.
