---
title: "Go Multi-Module Monorepo — Ecosystem Research and Tool Comparison"
description: "How Rust, Node, Java, Elixir solved multi-module releases. Why Go hasn't. Comparison of OTEL multimod, kimono, release-please, goreleaser for Go monorepos."
head:
  - - meta
    - name: keywords
      content: go monorepo, golang multi-module, go.work, go mod replace publish, cargo workspace vs go, npm workspaces vs go, otel multimod, kimono go, goreleaser multi-module, go sub-module release
---

# Research

## The Problem

Go has no built-in tooling for multi-module monorepos. `go.work` is dev-only (can't be committed). `replace` directives break on publish. Sub-modules need prefixed tags. Every project solves this with shell scripts and tribal knowledge.

## How Other Ecosystems Solved It

### Rust

`Cargo.toml` with `[workspace]`. All crates in one workspace. `cargo test` tests everything. `cargo-release` handles version bumps, stripping dev-deps, publishing in dependency order, and tagging. Dev overrides (`[patch]`) don't leak into published crates — Cargo handles this automatically.

### Node / TypeScript

`npm workspaces` or `yarn workspaces`. `package.json` in root with `"workspaces": ["packages/*"]`. Local packages auto-linked. `changesets` (community standard, 10k+ stars) handles version bumps, changelogs, npm publish, and tagging.

### Java / Kotlin

Maven multi-module since 2005. Root `pom.xml` with `<modules>`. `mvn release:prepare release:perform` — bumps versions, tags, deploys to Maven Central. 20 years of polish.

### Python

`extras` / optional dependencies. `pip install mylib[otel]` — user gets only what they need. No separate module required for optional deps.

### Elixir

Umbrella projects. `mix new --umbrella`. `{:dep, in_umbrella: true}` automatically replaced with version on publish to Hex. The exact transformation we do manually with shell scripts.

## Summary

| Problem | Rust | Node | Java | Python | Elixir | **Go** |
|---|---|---|---|---|---|---|
| Workspace init | `Cargo.toml` | `package.json` | `pom.xml` | `pyproject.toml` | `mix.exs` | **manual go.work** |
| Local deps in dev | `[patch]` auto | workspaces auto | parent POM | path deps | `in_umbrella` | **manual replace** |
| Dev deps don't leak to publish | ✅ auto | ✅ auto | ✅ auto | ✅ auto | ✅ auto | **❌ manual strip** |
| Release tool | cargo-release | changesets | mvn release | hatch | mix hex.publish | **❌ none** |
| Sub-module tagging | cargo-release | changesets | mvn release | N/A | mix | **❌ manual script** |
| Optional deps without split | `[features]` | `peerDeps` | `<optional>` | extras | optional | **❌ impossible** |

**Go is the only major language where multi-module monorepo has no tooling support.**

## What Exists in Go

### OTEL `multimod`

Internal tool (~3000 lines). Has `verify`, `prerelease`, `tag`. Requires manual `versions.yaml` config. Hardcoded for OTEL module paths. Not published as reusable tool. Supports "module sets" (groups with different versions) — overengineered for most projects.

### `kimono`

Young project for Go monorepo management. Has module discovery and tag coordination. Unclear how well it handles strip replace / pin require. Worth watching.

### AWS `go-multi-module-repository-tools`

Internal scripts including `updaterequires` for managing inter-module versions. Not designed for external use.

### `release-please` (Google)

Generic release tool. Creates tags and changelogs. Doesn't understand Go `replace` directives. Requires manual per-module config.

### `goreleaser`

For binaries, not libraries. Doesn't know about Go modules or multi-module.

### `gorelease` (golang.org/x/exp)

Checks API compatibility between versions. Useful but orthogonal — checks "what to release", not "how".

## Comparison

| Tool | Discovery | Strip replace | Pin require | Tags | Verify | Zero-config | Reusable |
|---|---|---|---|---|---|---|---|
| OTEL multimod | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| kimono | ✅? | ❓ | ❓ | ✅ | ❌ | ❌ | ✅? |
| AWS tools | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❓ |
| release-please | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ✅ |
| gorelease | N/A | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ |
| **multimod** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

No existing tool fills all columns. The gap is real.

## Industry Strategy

Large players avoid the problem entirely:

- **Google** — Bazel, not Go modules
- **Uber, HashiCorp** — single module, `internal/` packages
- **Kubernetes** — staging repos + publishing bot (10k lines of infra)

The absence of tooling causes the absence of multi-module projects, not the other way around.
