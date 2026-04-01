---
title: "multimod — Go Multi-Module Monorepo Tool"
description: "Zero-config multi-module management for Go monorepos. Auto-discovery, workspace sync, publish preparation, sub-module tagging. The tool Go ecosystem is missing."
head:
  - - meta
    - name: keywords
      content: go, golang, multi-module, monorepo, go.work, go mod replace, workspace, release tool, sub-module tagging, go mod tidy
---

# multimod

> "Every Go project with sub-modules writes its own release script. Nobody asks why."

## What

multimod is a multi-module management tool for Go monorepos. It solves the problems that Go's toolchain doesn't: workspace management, consistency checks, publish preparation, and template generation — all with zero configuration.

## Status

**Work in progress.** Lives in `pkg/multimod/` inside the resilience repository. Battle-testing on resilience itself. Will be extracted into a standalone repository when stable.

## Why

Go is the only major language ecosystem where multi-module monorepos have no tooling support. Every other ecosystem solved this years ago:

- **Rust** — `cargo workspace` + `cargo-release`
- **Node** — `npm workspaces` + `changesets`
- **Java** — Maven multi-module + `mvn release`
- **Elixir** — umbrella projects with automatic `in_umbrella` → version transform

In Go, every project writes its own shell scripts. OTEL wrote 3000 lines of internal tooling. Google Cloud has unpublished internal tools. AWS has scattered scripts. Nobody published a reusable solution.

## Commands

```
multimod go <args>              — daily work
multimod release <version>      — CI, publish preparation
multimod generate               — templates → files
multimod verify                 — check + auto-fix state
```

## Architecture

See [Specification](./spec.md) for the full technical design.

## Pages

- [Specification](./spec.md) — architecture, components, algorithms
- [Research](./research.md) — ecosystem analysis, comparison with existing tools
- [Vision](./vision.md) — roadmap from pkg/ to standalone repository
