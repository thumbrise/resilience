---
title: "multimod Specification"
description: "Full technical specification for multimod: zero-config multi-module management tool for Go monorepos. Architecture, commands, release flow, generate, conventions."
head:
  - - meta
    - name: keywords
      content: go multi-module architecture, go monorepo tool design, go.work automation, go mod replace strip, go sub-module versioning, golang workspace management
---

# Specification

## Philosophy

Multimod is an **infrastructure autopilot** for Go multi-module monorepos. The developer does not think about go.work, replace directives, go version sync, or release transforms. Multimod thinks for them.

**Convention over configuration.** Zero config files. Directory structure is the config. `.multimod/` directory is the only convention — and it's optional until you need templates.

**Always Apply.** Every invocation of multimod guarantees the filesystem matches the desired state. You cannot forget to sync. You cannot end up in an inconsistent state.

**Terraform thinking.** Discovery reads the filesystem and builds the desired State. Applier makes the filesystem match it. No diff-based patching — declare desired, apply unconditionally.

## Architecture

```
Boot → Discovery → desired State → Applier
```

### Boot

Checks that cwd has a `go.mod` — cwd is the project root. No upward directory traversal (same convention as goreleaser, terraform). Determines if this is a multi-module project (sub-directories with their own `go.mod`). If not — transparent proxy to `go`, multimod is invisible. Excludes `vendor/`, `testdata/`, `_`-prefixed, `.`-prefixed directories. Warns if no `.git` directory (CI misconfiguration, not a git repo).

### Discovery

Pipeline of steps: Parse → ValidateAcyclic → EnrichGoVersion → EnrichReplaces → EnrichWorkspace.

- **Parse** — walks filesystem, finds all `go.mod`, parses via `golang.org/x/mod/modfile`, extracts into domain `Module`. modfile types do not escape.
- **ValidateAcyclic** — builds dependency graph from sub-module requires, delegates to `graph.DetectCycle`. Root excluded by design (core has zero internal deps).
- **EnrichGoVersion** — sets all sub GoVersions to root's GoVersion.
- **EnrichReplaces** — unconditional: every sub gets replace for root + every other sub. Prevents `go mod tidy` from fetching internal modules from registry (chicken-and-egg solved).
- **EnrichWorkspace** — sets workspace to root + all subs.

New step = new file in `steps/` + one line in `NewDefaultDiscovery()`.

### State

Pure domain model. Architectural boundary between Discovery and Applier. No modfile types, no FS handles.

```go
type State struct {
    Root      Module
    Subs      []Module
    Workspace []Module
}
```

### Applier

Receives desired State, makes filesystem match it. Idempotent — compares before writing.

- **syncGoWork** — generates go.work, writes only if content differs.
- **syncGoMod** per sub — syncs go version + replaces. Three-phase replace sync: build desired → drop unwanted/stale → add missing.

## State contract

Valid State means:

1. Root module exists and is parsed
2. All sub-modules discovered and parsed
3. Dependency graph is acyclic (DAG)
4. Every sub-module has `replace` for all internal deps
5. `go.work` contains all modules
6. `go` directive is the same everywhere (root = source of truth)

## Two states of go.mod

Every sub-module's go.mod exists in two states:

| | Dev-state | Publish-state |
|---|---|---|
| Replace | `replace example.com/root => ../` | Removed |
| Require | `require example.com/root v0.0.0` | `require example.com/root v1.2.3` |
| Where | Main branch, always | Detached commit behind tag |
| Who sees | Developers | Users (`go get`) |
| Who creates | Multimod apply | Multimod release |

Dev-state is committed to git. This is normal — Go ignores replace directives in dependencies. Users never see dev-state.

## go.work — committed

`go.work` is part of managed state, committed to git. After `git clone` everything works: IDE resolves imports, `go mod tidy` sees workspace, `go test` runs. Multimod creates and maintains go.work, but it's useful without multimod too.

## Commands

### `multimod`

Apply + status banner. Every invocation syncs filesystem to desired state. Also runs generate if `.multimod/templates/` exists.

CI pattern:
```bash
multimod && git diff --quiet || exit 1
```

### `multimod go <args>`

Transparent proxy with multi-module awareness.

Multi-module commands (registry: `test`, `vet`, `build` with `./...`, `mod tidy`, `tool <name>` with `./...`): iterate all modules via `exec.CommandContext`, aggregate results.

Everything else: `syscall.Exec("go", args)` — direct passthrough, replaces process.

### `multimod release <version>`

Transforms dev-state → publish-state. Three levels of trust:

| Flag | What it does | Who uses |
|---|---|---|
| (no flags) | Dry-run: show plan | Developer, verification |
| `--write` | Locally: detached commit + tags | Developer, local testing |
| `--write --push` | Commit + tags + push | CI pipeline |

Release flow with `--write`:

1. Tag current HEAD as `v1.2.3-dev` (traceability)
2. `git checkout --detach` (main is never touched)
3. Strip internal replace directives from all sub go.mod
4. Pin internal require to version
5. `git commit "chore(release): v1.2.3 [multimod]"`
6. Tag detached commit: `v1.2.3` (root) + `<dir>/v1.2.3` (each sub)
7. `git checkout main` (return to dev-state)

With `--push`: also `git push origin --tags`.

Main **never leaves dev-state**. The publish-state commit is detached — accessible only via tag. `go get @v1.2.3` resolves the tag, gets publish-state. `go get @latest` picks the highest stable tag.

The `-dev` tag is for human traceability — quickly find which main commit produced the release. Tools (semantic-release, changelog generators) work through git parent chain from the stable tag.

### `multimod modules`

JSON output with project module map. Designed for piping into external tools.

```bash
multimod modules | multirelease v1.2.3 --write
multimod modules | jq '.subs[].dir'
```

Output contract:

```json
{
  "root": {"path": "...", "dir": "/absolute/path", "go_version": "1.25.0"},
  "subs": [
    {"path": "...", "dir": "/absolute/path/otel", "requires": ["..."]}
  ]
}
```

Dirs are absolute — pipe consumers don't know the caller's cwd. Only internal requires are included — external deps are not multimod's concern.

## Template generation (part of apply)

Not a command — a pipeline step. Runs automatically if `.multimod/templates/` exists.

Two template types:

- **Single file** — `dependabot.yml.tmpl` → one `dependabot.yml` from all modules
- **Per-module** — <code>module-ci-&#123;&#123;.Name&#125;&#125;.yml.tmpl</code> → one file per module

Path convention: template path minus `.tmpl` = output path. Directory structure in `templates/` mirrors project structure.

Engine: `text/template` from stdlib. Model = State (root, subs, paths, versions, dependencies).

Cleanup: per-module templates → glob pattern from template name (<code>&#123;&#123;.Name&#125;&#125;</code> → `*`). Files matching glob but not in desired set → deleted. Template deleted → multimod no longer owns those files, user cleans up manually.

Generated files get header where syntax allows: `# Code generated by multimod. DO NOT EDIT.`

## `.multimod/` directory

Created on first run if templates are needed:

```
.multimod/
├── .gitignore          # ignores cache/
├── cache/              # internal (SHA, timestamps)
│   └── .gitignore      # *
└── templates/          # user puts templates here
    └── .gitkeep
```

Committed: `templates/`, `.gitignore`. Ignored: `cache/`.

## IO

- **stdout** — belongs to Go. Never touched by multimod.
- **stderr** — multimod's channel. `[multimod]` prefix. Silent when everything is ok.

## Extension points

| Extension | How | Files touched |
|---|---|---|
| New discovery step | New file in `steps/` + one line in `NewDefaultDiscovery()` | 1 new + 1 line |
| New command | New struct + add to `NewCommands` registry | 1 new + 1 line |
| New multi-module go subcommand | Add matcher to `NewDefault()` in `goinvoker/internal/classifier/classifier.go` | 1 line |
| New template | Add `.tmpl` file to `.multimod/templates/` | 1 file |

## Requirements

- Linux / macOS (Windows via WSL)
- Git repository (warns if `.git` missing)
- Core module in root, sub-modules in subdirectories
- Sub-modules depend on core, not reverse (DAG)
- One version for all modules at release time
