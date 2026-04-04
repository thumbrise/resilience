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

multimod is an infrastructure autopilot for Go multi-module monorepos. It solves the problems that Go's toolchain doesn't: workspace sync, replace management, go version alignment, publish preparation, and template generation — all with zero configuration.

**Always Apply.** Every invocation guarantees the filesystem matches the desired state. You cannot forget to sync.

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
multimod                        — apply + status (syncs FS + generates templates)
multimod go <args>              — transparent proxy with multi-module iteration
multimod release <version>      — publish preparation (detached commit + tags)
```

## Architecture

```
Boot → Discovery → desired State → Applier
```

See [Specification](./spec.md) for the full technical design.

## Pages

- [Specification](./spec.md) — architecture, commands, release flow, generate
- [Research](./research.md) — ecosystem analysis, comparison with existing tools
- [Vision](./vision.md) — roadmap from pkg/ to standalone repository
