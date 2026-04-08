---
title: "RFC-001 — Composable Tooling for Go Multi-Module Monorepos"
description: "Architectural RFC: unix-way ecosystem of CLI tools for Go multi-module monorepo lifecycle. Born from a three-round adversarial design review, 2026-04-07."
---

# RFC-001 — Composable Tooling for Go Multi-Module Monorepos

| | |
|---|---|
| **Status** | Draft |
| **Date** | 2026-04-07 |
| **Origin** | Three-round adversarial architecture review (Skeptic, Implementor, Arbiter) |

> **Focus on Capabilities, Not Structure Compliance.**
> This RFC describes desired behaviors and constraints. Implementation details — function names, package layout, file structure — are deliberately omitted. Code that satisfies the capabilities is correct, regardless of how it's organized.

---

## 1. Problem Statement

Go has no standard tooling for multi-module monorepos.

A Go multi-module monorepo is a single git repository containing multiple `go.mod` files — a root module and one or more sub-modules. The pattern is growing: projects with optional integrations (OTEL, gRPC, Redis) isolate dependencies into separate modules so users pull only what they need.

The pattern is well-understood. The tooling is not.

**What every multi-module Go project must solve manually:**

1. **Workspace sync** — `go.work` must list all modules. Add a module, forget to update `go.work` — IDE breaks, `go mod tidy` fetches from registry instead of local.
2. **Replace directives** — sub-modules that depend on root (or each other) need `replace` directives pointing to local paths. Without them, `go mod tidy` fetches from registry — gets wrong version or 404.
3. **Go version alignment** — every `go.mod` should declare the same Go version. Drift causes subtle build differences.
4. **Release transforms** — dev-state `go.mod` has `replace ../` directives. Users must never see these. Before tagging a release, replaces must be stripped and internal requires pinned to the release version.
5. **Multi-module tagging** — Go proxy resolves sub-modules by prefix tag: `otel/v1.2.3` for `example.com/root/otel`. Each sub-module needs its own tag. Manual tagging is error-prone.
6. **Iterative commands** — `go test ./...` in root does not test sub-modules. Each module must be tested independently.

**How the ecosystem solves this today:**

- **OTEL Go** — ~3000 lines of shell scripts + `versions.yaml` config. Not reusable.
- **Kubernetes** — custom `staging/` scripts. Not reusable.
- **Everyone else** — Taskfile/Makefile with `cd sub && go test ./...` loops. Fragile, duplicated across projects.

Rust has `cargo-release`. Node has `changesets`. Java has `mvn release`. Elixir has umbrella projects. Go has nothing.

**Real projects struggling with this today:** Uber's zap extracted benchmarks into a separate module to avoid dependency pollution, but this "complicates the build script" **[E5]**. Pulumi must "publish a tag for each go.mod path" manually **[E3]**. HashiCorp's Azure SDK split into 3 modules means "each release will become 3 separate Git Tags" **[E6]**. Grafana's replace directives from local debugging leak into shared code **[E7]**. Even goreleaser — the most popular Go release tool — "is unable to detect" sub-module tags **[E4]**. AWS acknowledges "the lack of official Golang support for this task" **[E8]**. The pattern is wanted; the tooling is missing.

This RFC proposes a composable ecosystem of CLI tools that covers the full lifecycle: clone → develop → test → release → publish.

---

## 2. Prior Art & Analysis

### 2.1 OTEL Go multimod (opentelemetry-go-build-tools)

The largest public Go multi-module project (~40 modules) built their own tool also called `multimod` (`go.opentelemetry.io/build-tools/multimod`). The name collision is coincidental — the tools share a problem domain but differ fundamentally in approach **[E8]**.

**OTEL multimod:** config-driven. Requires `versions.yaml` that groups modules into named sets (stable-v1, experimental-metrics, bridge), each with a version number. Three CLI commands: `verify` (validate YAML), `prerelease` (update go.mod files, create branch + commit), `tag` (create git tags). Written in Go with Cobra. Tied to OTEL conventions.

**Our multimod:** convention-driven. Zero config files. Auto-discovers modules from filesystem. Manages go.work, replace directives, go version sync. Emits JSON module map for pipe. Release via separate `multirelease` binary with detached commit model.

| | OTEL multimod | Our multimod |
|---|---|---|
| Discovery | Manual (`versions.yaml`) | Auto (filesystem scan) |
| Config | Required (`versions.yaml`) | None (convention-over-config) |
| Module groups | Yes (named sets with versions) | No (uniform lifecycle, YAGNI) |
| Release model | Prerelease branch | Detached commit |
| Replace management | No | Yes (sync + strip for publish) |
| go.work management | No | Yes (generate + sync) |
| Go version sync | No | Yes |
| JSON pipe output | No | Yes |
| Standalone | No (tied to OTEL) | Yes |

**What they got right:** module sets with different lifecycle (stable v1.x, experimental v0.x). Explicit grouping for large projects (40+ modules).

**What they got wrong:** config-driven discovery. YAML duplicates what `go.mod` already declares. No auto-discovery. No dev-state management (replace, go.work, go version).

### 2.2 semantic-release

The dominant release automation tool (Node ecosystem, used in Go via npx). Analyzes conventional commits, determines semver bump, creates tags and GitHub Releases.

**Fundamental incompatibility with multi-module Go:** semantic-release tags the current branch (main). In a multi-module project, main is in dev-state — `go.mod` files contain `replace ../` directives. Users who `go get @v1.2.3` receive broken go.mod. Additionally, semantic-release uses `git tag --merged` to find previous versions. Detached commits (our release model) are not reachable from main — the version chain breaks.

**Confirmed through adversarial review:** this is not a plugin/configuration issue. It is a fundamental architectural mismatch. The Node ecosystem reached the same conclusion — `changesets` replaced semantic-release for monorepo use cases (Vercel, Chakra UI, Radix).

### 2.3 goreleaser

Builds and publishes Go binaries. Does not understand Go modules, replace directives, or multi-module workspaces. Different tool for a different problem (binaries vs libraries).

### 2.4 svu, cocogitto, git-cliff

Unix-way CLI tools for version management and changelog generation. `svu` — semver from git tags. `cocogitto` — conventional commits analysis. `git-cliff` — changelog generation. Each does one thing. Composable through stdout. These are potential components of the ecosystem, not competitors.

### 2.5 kimono (bonzai)

Part of the bonzai CLI framework by rwxrob. Provides `work` (toggle go.work), `tidy` (go mod tidy across modules), `tag` (prefix-based tagging), `deps`/`dependents` (dependency analysis). Auto-discovers modules via filesystem walk.

**What it does well:** dev-time convenience — toggling workspace, running tidy across modules.

**What it doesn't do:** no replace management, no go version sync, no release transforms (strip replaces, pin requires), no detached commit, no JSON output, no publish-state validation. Tags current HEAD directly.

**Classification:** dev convenience tool, not a release tool.

### 2.6 monorel (The Root Company)

Automates releases for individual modules in a monorepo. Generates `.goreleaser.yaml`, computes next version from git log, creates prefix tags (`cmd/tool/v1.0.0`), publishes via goreleaser + gh.

**What it does well:** binary release automation with per-module version tracking.

**What it doesn't do:** no replace management, no go.work sync, no publish-state transforms. Tightly coupled to goreleaser — designed for binaries, not libraries. No JSON output.

**Classification:** binary release tool, not a library release tool.

### 2.7 Crosslink (OTEL build-tools)

Part of OTEL's build toolchain (`go.opentelemetry.io/build-tools/crosslink`). Scans modules and inserts `replace` directives for intra-repository dependencies. Can generate `go.work`. Supports `prune` for stale replaces.

**Limitations:** requires `--root` flag or git-based root detection (not fully auto-discovery). Works only within one module namespace. Does not sync `go version`. No JSON output, not pipe-friendly. Tied to OTEL conventions.

**Classification:** partial dev-state tool — covers replace sync but not the full lifecycle.

### 2.8 Gorepomod (Kustomize/SIG)

Tool for multi-module repos in Kubernetes ecosystem (`sigs.k8s.io/kustomize/cmd/gorepomod`). Commands: `pin` (remove replaces, fix versions for publish), `unpin` (add replaces for dev), `release` (compute version, create release branch, tag, push).

**Key insight:** `pin`/`unpin` is the same two-state model as our dev-state/publish-state — different names, same concept. Confirms the pattern is real and independently discovered.

**Limitations:** uses release branches, not detached commits — mixes dev-state and publish-state on the same branch during hotfix. Tied to Kustomize structure. Last release 6+ years ago — effectively unmaintained. No JSON output.

**Classification:** partial release tool with correct model but abandoned implementation.

### 2.9 Go toolchain (`go work`, `go mod`)

`go work` manages workspace. `go mod tidy` syncs dependencies. But:

- `go work` requires manual `go work use ./otel` — no auto-discovery
- `go work` does not manage replace directives in `go.mod`
- `go mod tidy` does not sync Go version across modules
- Neither knows about releases

The ecosystem complements Go toolchain, not competes with it. If Go adds built-in multi-module release support — the ecosystem has served its purpose.

**Go proposals for optional dependencies:** issue [#44550](https://github.com/golang/go/issues/44550) (2019) proposed optional dependencies in `go.mod` — not implemented. Issue [#47034](https://github.com/golang/go/issues/47034) (2021) proposed optional mode for semantic import versioning — not implemented. The Go team is aware of the problem but has not prioritized it. Until they do, the gap remains.

---

## 3. Design Principles

### 3.1 Unix Philosophy

Each tool does one thing. Tools communicate through stdin/stdout/JSON. Any tool is replaceable with an alternative that speaks the same contract. No tool knows about the internals of another.

**Litmus test:** can a user replace any single tool with a shell script or third-party alternative? If not — the boundary is wrong.

### 3.2 Target Niche: Core + Optional Extensions

The ecosystem targets a specific Go monorepo pattern: **root module is the core library (zero or minimal deps), sub-modules are optional extensions (own deps)**.

Users `go get` only what they need:
- `go get example.com/root` — core, zero transitive deps
- `go get example.com/root/otel` — OTEL extension, pulls only `go.opentelemetry.io/otel`

In this model: sub-modules always depend on root, never reverse. Root is the foundation. Extensions build on top. An extension cannot require a higher Go version than its core — that would mean the extension is incompatible with its own foundation.

**Examples:** OTEL Go (core + bridge/sdk), go-kit (core + transports), resilience (core + otel extension).

**Not targeted:** monorepos with independent modules that happen to share a repository (e.g. microservices). For those, each module has its own lifecycle and version — our "one version for all" model does not apply.

**Monorepo ≠ multi-module project.** A monorepo is a storage strategy (one git repo, many projects). A multi-module project is an architecture strategy (one product, many Go modules). These are orthogonal:

- A monorepo can contain multiple multi-module projects, each with its own `multimod`
- A multi-module project can live in a standalone repo (not a monorepo)
- Monorepo tools (bazel, nx, turborepo) manage **which projects to build**. multimod manages **how a single Go product organizes its modules**. They operate at different levels and do not conflict.

multimod activates only when it finds a root `go.mod` with sub-module `go.mod` files in subdirectories. No root `go.mod` → "not a multi-module project" → transparent proxy to `go`. This is architectural enforcement, not documentation.

### 3.3 Zero Configuration

Directory structure is the config. A `go.mod` file in a subdirectory = a sub-module. No YAML, no TOML, no `.multimod.json`. Discovery is automatic, deterministic, and auditable (run the tool, see what it found).

**Known limitation:** zero-config works for projects with uniform lifecycle (all modules release together). Projects with 10+ modules and mixed stability levels (stable v1.x + experimental v0.x) may need a grouping mechanism. This is acknowledged as future work — the architecture does not close this path, but does not solve it today.

### 3.4 Terraform Thinking

Discovery reads the filesystem and builds the desired State. Applier makes the filesystem match it. No diff-based patching — declare desired, apply unconditionally. Idempotent: running twice produces the same result.

### 3.5 Two States of go.mod

Every sub-module's `go.mod` exists in exactly two states:

| | Dev-state | Publish-state |
|---|---|---|
| Replace | `replace example.com/root => ../` | Removed |
| Require | `require example.com/root v0.0.0` | `require example.com/root v1.2.3` |
| Where | Main branch, always | Detached commit behind tag |
| Who sees | Developers | Users (`go get`) |

Main **never leaves dev-state**. This is the core invariant. Dev-state is committed to git — Go ignores replace directives in dependencies, so users never see them.

### 3.6 Detached Commit Release Model

Publish-state lives on a detached git commit, accessible only via tag. `go get @v1.2.3` resolves the tag, downloads the commit, reads clean `go.mod`. The commit is not on any branch.

**Why not two commits on main (release + restore)?** Main temporarily has publish-state. CI runs between commits, someone pulls at the wrong moment — broken dev environment. Detached commit is invisible to branch-based workflows.

**Verified:** `proxy.golang.org` caches modules permanently after first fetch, even if the tag is deleted from the repository. Detached commits behind tags are fully supported by Go's module infrastructure.

### 3.7 Composable, Not Framework

The ecosystem is a set of tools, not a framework. Each tool has a clear input/output contract. Users can adopt one tool without adopting all. `multimod` (dev-state sync) is useful without `multirelease` (publish-state). `multirelease` is useful without `multimod` — pipe any JSON module map into it.

**Anti-goal:** becoming semantic-release for Go. One monolithic tool that does everything and can't be decomposed.

---

## 4. Ecosystem Overview

The full lifecycle is covered by four tools. Each is an independent binary with its own domain.

```
clone → multimod → develop → multimod go → test → release pipeline
                                                         │
                                          version-bumper → multirelease → ghreleaser
```

| Tool | Domain | Input | Output | Status |
|------|--------|-------|--------|--------|
| **multimod** | Dev-state sync + module iteration | Filesystem | Synced FS, JSON module map | Implemented |
| **multirelease** | Publish-state creation | JSON module map (stdin) + version (arg) | Detached commit + tags | PoC |
| **version-bumper** | Version determination | Git history | Version string (stdout) | Planned |
| **ghreleaser** | GitHub Release publication | Version + git history | GitHub Release | Planned (may use `gh` CLI directly) |

**Adoption is incremental.** A project can use only `multimod` for dev-state sync and never touch the release tools. Or use `multirelease` with a manually specified version and skip version-bumper entirely. Each tool is useful in isolation.

**Third-party alternatives are welcome.** version-bumper can be replaced by `svu`, `cocogitto`, or a shell script. ghreleaser can be replaced by `gh release create --generate-notes`. The ecosystem does not require all four tools — it requires the contracts between them.

---

## 5. Tool Capabilities

### 5.1 multimod — Dev-State Guardian

**Purpose:** guarantee that after any invocation, the filesystem matches the desired dev-state. Zero-config. Idempotent.

**Capabilities:**

- **Discovery** — scan filesystem, find all `go.mod` files, classify root vs sub-modules. Exclude `vendor/`, `testdata/`, `.`-prefixed directories. Include `_`-prefixed directories as workspace-only modules.
- **Workspace sync** — generate `go.work` with all discovered modules. Write only if content differs.
- **Replace sync** — ensure every sub-module has `replace` directives for all internal modules. Add missing, remove stale, fix incorrect paths.
- **Go version sync** — propagate root module's `go` directive to all sub-modules.
- **Module iteration** — execute `go` commands across all modules. Classify which commands need iteration (test, vet, build with `./...`, mod tidy, tool with `./...`) vs passthrough.
- **Module map output** — emit JSON module map to stdout for consumption by other tools.

**Conventions:**

- `_`-prefixed directories contain workspace-only modules — included in workspace and dev-state sync, but marked as non-releasable.
- `.`-prefixed directories are excluded entirely (hidden directories).
- `vendor/` and `testdata/` are excluded (Go convention).

### 5.2 multirelease — Publish-State Creator

**Purpose:** transform dev-state go.mod files into publish-state, create detached commit with tags. Reads module map from stdin — zero knowledge of how modules were discovered.

**Capabilities:**

- **Read module map** — parse JSON from stdin. Expect `root` (path, dir) and `subs` (path, dir, requires). Contract versioned (`version` field).
- **Plan** — compute release plan: which files to transform, which tags to create, which modules are workspace-only (not tagged).
- **Dry-run** — output plan to stdout without touching filesystem or git. Default mode.
- **Transform** — for each sub-module go.mod: strip internal replace directives, pin internal require versions to release version.
- **Validate publish-state** — after transform, before commit: run `GOWORK=off go build ./...` in each transformed module. If any module fails to build in isolation — abort, rollback, clear error. Publish-state must be proven buildable before it becomes a tag.
- **Detached commit** — checkout detach, stage transforms, commit with release message, return to original branch.
- **Tagging** — tag detached commit: root tag (`v1.2.3`) + per-sub-module tags (`otel/v1.2.3`). Dev traceability tag (`v1.2.3-dev`) on original HEAD.
- **Push** — push specific tags to origin (not `--tags` which pushes all local tags).
- **Stdin detection** — fail fast with clear message if stdin is a terminal, not a pipe.

**Three levels of trust:**

| Mode | What happens | Who uses |
|------|-------------|----------|
| (default) | Dry-run: show plan, touch nothing | Developer verification |
| `--write` | Local: detached commit + tags | Developer local testing |
| `--write --push` | Commit + tags + push to origin | CI pipeline |

### 5.3 version-bumper — Version Oracle

**Purpose:** determine the next semantic version from git history. Output version string to stdout. Nothing else.

**Capabilities:**

- **Analyze commits** — parse conventional commits between last tag and HEAD. Determine bump: `feat` → minor, `fix`/`perf` → patch, `BREAKING CHANGE` or `!` → major.
- **Output** — print next version to stdout (e.g. `v1.3.0`). If no release-worthy commits — exit 0 with empty stdout (no release needed).
- **Configurable** — which commit types trigger which bump. Default follows conventional commits spec.

**May be replaced by:** `svu next`, `cog bump --auto --dry-run`, or any tool that outputs a version string.

### 5.4 ghreleaser — GitHub Release Publisher

**Purpose:** create a GitHub Release with release notes on an existing tag. Does not create tags — that's multirelease's job.

**Capabilities:**

- **Release notes** — generate from conventional commits between previous and current tag. Group by type (Features, Bug Fixes, etc.).
- **GitHub Release** — create release on existing tag via GitHub API.
- **Configurable** — note format, grouping, header template.

**May be replaced by:** `gh release create v1.2.3 --generate-notes`, `git-cliff | gh release create v1.2.3 --notes-file -`, or any tool that creates GitHub Releases.

---

## 6. Contracts & Interfaces

### 6.1 Module Map JSON (multimod → multirelease)

The module map is the primary contract between tools. Emitted by `multimod modules` on stdout, consumed by `multirelease` on stdin.

```json
{
  "version": 1,
  "root": {
    "path": "github.com/example/project",
    "dir": "/absolute/path/to/project",
    "go_version": "1.25.0"
  },
  "subs": [
    {
      "path": "github.com/example/project/otel",
      "dir": "/absolute/path/to/project/otel",
      "requires": ["github.com/example/project"],
      "workspace_only": false
    },
    {
      "path": "github.com/example/project/tools",
      "dir": "/absolute/path/to/project/_tools",
      "requires": ["github.com/example/project"],
      "workspace_only": true
    }
  ]
}
```

**Guarantees at version 1:**

- `root` is always present, has `path` and `dir`
- `subs` is an array (may be empty)
- `dir` is absolute path
- `requires` contains only internal module paths (external deps excluded)
- `workspace_only` indicates modules that participate in dev-state but are not tagged for release
- Fields may be added in future versions — consumers must ignore unknown fields

### 6.2 Full Pipeline Example

```bash
# CI release pipeline — one line
multimod modules | multirelease $(version-bumper) --write --push

# Expanded form:
VERSION=$(version-bumper)                          # → "v1.2.3" or empty
[ -z "$VERSION" ] && echo "No release needed" && exit 0
multimod modules | multirelease "$VERSION" --write --push  # → detached commit + tags
gh release create "$VERSION" --generate-notes              # → GitHub Release
```

Note: `version-bumper` is command substitution (argument), not stdin pipe. `multirelease` reads JSON module map from stdin and version from argument. If `version-bumper` returns empty — no release, pipeline exits cleanly.

### 6.3 Version String (version-bumper → multirelease)

Stdout, one line, semver with `v` prefix: `v1.2.3`. Empty stdout = no release needed.

### 6.4 Exit Codes

All tools follow Unix convention:

- `0` — success (or "no action needed" for version-bumper)
- `1` — error (message on stderr)

### 6.5 IO Convention

- **stdout** — structured output (JSON, version string). Reserved for pipe.
- **stderr** — human-readable logs, progress, errors. Tools use `slog` with component tag.

---

## 7. Disputed Points

This section documents arguments raised during the adversarial architecture review and their resolutions. Each point was debated across three rounds by Skeptic (challenger), Implementor (defender), and Arbiter (fact-checker with search access).

### 7.1 "Pipe-ecosystem from Go binaries is hypocrisy"

**Challenge:** Unix utilities weigh kilobytes. Each Go binary is 10-15MB. Four tools = 50MB. This is not Unix-way.

**Resolution:** argument about binary size was withdrawn by Skeptic — compile-time disk cost in 2026 is negligible. The real question was whether tools share domain knowledge (which would argue for one binary). Analysis showed that `internalPaths()` in multirelease is derivation from input data, not duplicated domain knowledge — like `wc` counting lines from stdin. The `_` prefix convention was identified as the one piece of shared knowledge — resolved by adding `workspace_only` to the JSON contract so multirelease does not need to interpret directory names.

**Precedent:** Terraform (state management) and Terragrunt (orchestration) — different binaries, different domains, communicate through files. Not plugins of each other.

### 7.2 "This is just semantic-release decomposed into boxes"

**Challenge:** version-bumper + multirelease + ghreleaser = same three steps as semantic-release.

**Resolution:** rejected. The detached commit model is fundamentally different from semantic-release's branch-tagging model. semantic-release uses `git tag --merged` to find previous versions — detached commits are unreachable from main, breaking the version chain. This is not a configuration issue but an architectural incompatibility. The Node ecosystem reached the same conclusion — changesets replaced semantic-release for monorepo use cases.

### 7.3 "JSON contract is your vendor lock-in"

**Challenge:** multirelease reads JSON from multimod modules. Anyone wanting to use multirelease without multimod must generate this JSON format.

**Resolution:** partially accepted. JSON is an open format, and the contract is simple enough to generate with `jq` or any language. However, the contract needs explicit versioning and stability guarantees. Resolution: add `"version": 1` field, document guarantees, require consumers to ignore unknown fields (forward compatibility).

**Note:** compatibility with `go list -json -m all` was investigated. The formats are structurally different (stream of objects vs hierarchical document) and serve different purposes (`go list` doesn't distinguish root/sub or track internal requires). Superset is not feasible, but the module map adds genuine value over `go list`.

### 7.4 "Zero-config doesn't scale to 40+ modules"

**Challenge:** OTEL has module sets with different lifecycle. "Release all together" doesn't work for mixed stable/experimental projects.

**Resolution:** accepted as future limitation, rejected as current blocker. The architecture supports extension — `Module` struct can gain fields, pipeline can gain steps. Convention-based grouping (`_` prefix for workspace-only) is enforced by tooling, not just documented. For 3-10 modules this is sufficient. For 40+ modules with mixed stability — a grouping mechanism will be needed. YAGNI today, path not closed.

**Skeptic's valid point:** convention without enforcement is documentation, not architecture. `_` prefix is enforced by multimod (workspace-only classification) — this is tool enforcement, analogous to how Go compiler enforces `internal/`.

### 7.5 "Detached commit is a hack"

**Challenge:** Prometheus accidentally got a detached release tag — chaos ensued. What about Go proxy, GitHub Archive, `git gc`?

**Resolution:** rejected. Detached commits behind tags are fully supported by Go module infrastructure. Verified through external research: `proxy.golang.org` caches modules permanently after first fetch, even if the source tag is deleted. Tags protect commits from `git gc`. The `-dev` tag on main provides traceability (which main commit produced the release).

**"Why not release branch?"** — debated across multiple rounds. Skeptic argued release branch allows amend and has branch protection. Implementor showed the fundamental problem: release branch contains publish-state (no replaces, pinned requires). Hotfix requires restoring dev-state on the branch, patching, then re-publishing. This mixes two states on one branch. With detached commit: hotfix on any dev-state branch → new detached commit. No state mixing.

**Arbiter resolved the LTS question:** any branch (main, release/v1.2, feature/x) can be source for detached commit. LTS branch contains dev-state, releases are detached from its HEAD. Same flow everywhere:

```
main (dev-state)           → detached v1.3.0
release/v1.2 (dev-state)   → detached v1.2.4
```

**Detached commit is a design trade-off, not an innovation:**

| | Detached commit | Release branch |
|---|---|---|
| Cleanup after release | Not needed | Must delete or maintain |
| Amend before push | New commit required | Amend possible |
| Tag protection | Limited (GitHub tag rules) | Branch protection (mature) |
| Hotfix workflow | Patch dev-state → new detached | Must restore dev-state on branch |
| LTS support | Separate dev-state branch + detached | Release branch per version |
| State mixing | Never — branches always dev-state | Branch alternates between states |

**Bugs found during review:** (1) `git push origin --tags` pushes all local tags, not just release tags — must use explicit tag list. (2) No cleanup of tags if push fails — retry gets "tag already exists". Both accepted as implementation bugs, not architectural flaws.

### 7.6 "You compete with Go toolchain"

**Challenge:** `go work` and `go mod tidy` already solve parts of this. Go team may add more.

**Resolution:** accepted as conscious risk. The ecosystem uses `golang.org/x/mod/modfile` (official Go library) — correct side of the API boundary. If Go adds built-in multi-module release support, the ecosystem has served its purpose. Recommendation: CI job on Go release candidates to catch breaking changes early.

---

## 8. Known Limitations & Future Work

### 8.1 Module Groups

Current design: all modules release with the same version. Works for uniform-lifecycle projects. Does not work for projects with stable (v1.x) and experimental (v0.x) modules. Future work: grouping mechanism, likely convention-based (directory prefix or go.mod annotation), not config-file-based.

### 8.2 Rollback

If multirelease creates tags but push fails, tags remain locally. No automatic rollback. Future work: cleanup tags in error path, or make the tool idempotent (skip existing tags instead of failing).

### 8.3 Selective Release

Current: release all modules or nothing. `--module` flag mentioned in FAQ but not implemented. Future work: selective release of individual modules or groups.

### 8.4 Template Generation

Mentioned in spec but not covered by this RFC. Templates (dependabot.yml, CI configs) generated from module map. Orthogonal to release — part of multimod's dev-state responsibilities.

### 8.5 Integration Testing on Go RC

No CI job on Go release candidates. Risk: Go toolchain changes could break multimod. Mitigation: `golang.org/x/mod/modfile` is stable API, but semantic changes in `go mod tidy` or `go work` could affect behavior.

### 8.6 CI Isolation Check

`go.work` in repository root changes behavior of `go build` and `go test` — they use local modules instead of published ones. CI must include a `GOWORK=off go test ./...` step to verify that published modules work in isolation. Without this, a release may break users who consume modules individually.

### 8.7 Toolchain Directive Sync

Go 1.21+ introduced `toolchain` directive in `go.mod`. Currently multimod syncs `go` version but not `toolchain`. Future work: sync both, or make `toolchain` sync optional.

### 8.8 Retract Automation

`go mod retract` is the only way to mark a broken published version. Currently requires manual intervention. Future work: `multirelease retract v1.2.3` command that creates a new detached commit with retract directive and tags it as the next patch version.

### 8.9 Release Validation

Before creating detached commit, multirelease should validate: all internal replaces are stripped, all internal requires are pinned, no local-path replaces remain. Strict validation prevents publishing broken go.mod.

### 8.10 Atomic Multi-Module Release

When modules have cross-dependencies (A depends on B), release order matters: B must be tagged before A's go.mod can reference B's version. Future work: dependency-aware release ordering, or a separate tool that computes which modules changed and should release together.

---

## 9. Decision Log

| # | Decision | Rationale | Date |
|---|----------|-----------|------|
| D1 | Detached commit for publish-state | Main never leaves dev-state. Go proxy works with tags, not branches. | 2026-04-07 |
| D2 | Separate binaries per domain | Different domains (dev-state vs release vs versioning). JSON contract between them. Enforced by `cmd/` not `pkg/`. | 2026-04-07 |
| D3 | Replace semantic-release with pipe ecosystem | Fundamental incompatibility: semantic-release tags main (dev-state), can't find detached tags via `git tag --merged`. | 2026-04-07 |
| D4 | Zero-config with known limitation | Convention over configuration for uniform-lifecycle projects. Module groups deferred (YAGNI). | 2026-04-07 |
| D5 | `_` prefix = workspace-only | Modules in `_`-prefixed dirs participate in dev-state but are not tagged for release. Enforced by tooling. | 2026-04-07 |
| D6 | JSON module map as primary contract | Versioned (`"version": 1`), absolute paths, internal requires only. Forward-compatible (ignore unknown fields). | 2026-04-07 |
| D7 | Explicit tag push, not `--tags` | `git push origin --tags` pushes all local tags. Explicit list prevents leaking experimental tags. | 2026-04-07 |
| D8 | `pkg/` → `cmd/` for loose coupling | Types in `cmd/` are not importable by external tools. Forces JSON as the only interface. Architectural enforcement, not documentation. | 2026-04-07 |

---

*This RFC is a living document. It will be updated as the ecosystem evolves. Decisions are final unless revisited through a new adversarial review.*

---

## Appendix A: Evidence Base

Public sources confirming the problem space and validating design decisions. Each link verified for existence, quote accuracy, and contextual relevance (2026-04-08).

### A.1 Go Issues — Official Recognition

- **[E1]** [golang/go#75900](https://github.com/golang/go/issues/75900) — *"replace directives don't really work because it breaks go install for that command package."* Multi-module updates in same repo require multiple commits.
- **[E2]** [golang/go#51967](https://github.com/golang/go/issues/51967) — *"it is more likely that people will take code intended for local development and deploy it into a production environment."* go.work as production risk.

### A.2 Real Projects — Pain Documentation

- **[E3]** [pulumi/pulumi#4213](https://github.com/pulumi/pulumi/issues/4213) — *"we need to publish a tag for each go.mod path."* Manual multi-module tagging burden.
- **[E4]** [goreleaser/goreleaser#1848](https://github.com/goreleaser/goreleaser/issues/1848) — *"Goreleaser is unable to detect those tags at the moment."* Popular release tool lacks multi-module support.
- **[E5]** [uber-go/zap@fd38d2a](https://github.com/uber-go/zap/commit/fd38d2a0cbc80152758800ff5d7e676e748d33af) — *"To keep the list of direct dependencies of Zap minimal, benchmarks are placed in their own Go module."* Dependency isolation via multi-module.
- **[E6]** [hashicorp/pandora#3601](https://github.com/hashicorp/pandora/issues/3601) — *"each release will become 3 separate Git Tags — one per submodule."* HashiCorp facing same tagging problem.
- **[E7]** [grafana/grafana#72346](https://github.com/grafana/grafana/pull/72346) — *"this line is giving me some errors due to inconsistency in the versions."* Replace directives from local debugging leak into shared code.

### A.3 Industry Analysis

- **[E8]** [AWS Blog: Simplifying OpenTelemetry releases with MultiMod](https://aws.amazon.com/cn/blogs/opensource/simplifying-opentelemetry-collector-and-go-library-releases-with-the-go-multimod-tool/) — *"Managing versions and releases of multiple Go Modules can be a struggle... especially due to the lack of official Golang support for this task."*
- **[E9]** [Hacker News discussion](https://news.ycombinator.com/item?id=27028202) — *"This is trivial to do with any other module system I've used (Maven, Nuget, Konan, pip, cargo), but it is extraordinarily brittle with Go."*
