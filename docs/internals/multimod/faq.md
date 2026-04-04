---
title: "multimod FAQ — The Angry User Edition"
description: "Every hard question about multimod, answered honestly. Why not just shell scripts? Why detached commits? Why unconditional replaces? What if Go steals the idea?"
head:
  - - meta
    - name: keywords
      content: go multi-module faq, go monorepo questions, go.work committed, go mod replace why, detached commit release, multimod vs goreleaser
---

# FAQ — The Angry User Edition

> Every tool gets angry users. Here are their questions — and our honest answers.

## General

### "What does multimod even do? I don't get it."

One command syncs your entire multi-module Go monorepo: go.work, replace directives, go version alignment, template generation. One command releases all modules with correct tags. Zero config.

Without multimod, you maintain shell scripts, Taskfile targets with `deps: [check-invariant]`, and hope nobody forgets to run them.

### "Why not just write shell scripts like everyone else?"

You can. OTEL wrote 3000 lines of shell scripts. AWS wrote their own. Every Go monorepo reinvents this wheel. We got tired of reinventing.

### "This is massive NIH. Nobody needs this."

Rust has `cargo-release`. Node has `changesets`. Java has `mvn release`. Elixir has umbrella projects. Go has... nothing. We checked. The gap is real, not imagined.

### "What's your use case? This seems very niche."

Any Go project with 2+ modules in one repo. That's every project with optional integrations (OTEL, gRPC, Redis) that want isolated dependencies. The pattern is growing, not shrinking.

## Architecture

### "Why cwd-is-root? What if I want to run from a subdirectory?"

`cd` to your project root. Like goreleaser. Like terraform. `go.mod` files are not unique markers — there could be 10 in a directory tree. Traversing upward without a boundary is a footgun. We chose safety over convenience.

### "Why not use go.work as the root marker and search upward?"

Because go.work might not exist yet. Multimod creates it. Chicken-and-egg. Cwd-is-root has zero edge cases.

### "Why does every sub-module get replace for ALL other modules? That's noisy!"

Chicken-and-egg. Developer writes `import "example.com/root"` → runs `go mod tidy` → tidy adds `require` → but no replace exists yet → tidy fetches from registry → gets wrong version or 404. The replace must exist **before** the require. Unconditional replaces guarantee this. Unused replaces are harmless — Go ignores them.

### "But go.work already solves the replace problem!"

Only if go.work exists. After `git clone` before running any tool — go.work might not be there. Committed replace directives work immediately. Belt and suspenders.

### "Why commit go.work? The Go team says not to!"

The Go team's advice is for single-module projects. For multi-module monorepos, committed go.work means: after `git clone`, IDE works, `go mod tidy` works, `go test` works. Zero setup. Multimod maintains it, but it's useful without multimod too.

## Replace & Publishing

### "You commit replace directives?! Users will get broken go.mod!"

No. Go **ignores** replace directives in dependencies. When a user does `go get example.com/your/module@v1.2.3`, Go reads the go.mod from the tagged commit but skips all replace directives. Users never see your dev-state. This is how Go works by design.

### "What if someone imports my -dev tag?"

They'd have to explicitly type `go get example.com/root@v1.2.3-dev`. The `-dev` suffix is a semver pre-release — `go get @latest` ignores it. And if they do import it, `require example.com/root v0.0.0` will fail loudly. No silent bugs.

## Release

### "How do you tag a release if multimod always keeps dev-state?"

Detached commit. Multimod creates a commit that's not on any branch — just floating in git space with a tag. Main stays in dev-state. The publish-state commit is accessible only via tag. `go get @v1.2.3` finds the tag, gets clean go.mod.

### "Detached commits are bad practice! Nobody does this!"

Prometheus accidentally got a detached release tag — that caused confusion. But ours is **deliberate**, with a clear `-dev` tag for traceability. The alternative — two commits on main (release + restore) — means main temporarily has publish-state. CI runs between commits, someone pulls at the wrong moment — broken dev environment. We chose invisible over broken.

### "What does the release commit look like?"

```
chore(release): v1.2.3 [multimod]

- strip internal replace directives
- pin internal require to v1.2.3
```

Only go.mod files change. No code changes. `[multimod]` fingerprint so you know it's automated. `chore` type — invisible in changelogs (conventional commits convention).

### "How does this work with semantic-release / changelog generators?"

The detached commit is a child of the main commit. `git log v1.2.2..v1.2.3` sees all main commits between releases through the parent chain. The `chore(release)` commit is filtered by conventional commits. Changelog is clean automatically.

### "I don't trust your tool with write access to my repo!"

```bash
multimod release v1.2.3            # dry-run: shows plan, touches nothing
multimod release v1.2.3 --write    # local only: commit + tags on your machine
multimod release v1.2.3 --write --push  # CI: commit + tags + push
```

Three levels of trust. Without `--write` — read-only. With `--write` — local only, inspect before pushing. Don't like it? `git tag -d v1.2.3` and start over. Main is untouched.

### "My root module depends on sub-modules. Will this work?"

No. Multimod will reject this with a clear error: "root module must not require internal sub-modules." Root is the zero-deps core. Subs depend on root, not reverse. This is the standard Go monorepo convention (OTEL, Kubernetes, every major project).

## Competition

### "Why not use OTEL's multimod?"

OTEL's multimod is ~3000 lines of internal tooling hardcoded for OTEL module paths. It requires manual `versions.yaml` config. It's not published as a reusable tool. We built a generic, zero-config alternative.

### "Why not use goreleaser?"

Goreleaser is for binaries, not libraries. It doesn't understand Go modules, replace directives, or multi-module workspaces. Different tool for a different problem.

### "What if Go adds built-in multi-module support?"

Then we win. It means the problem was real, the design was right, and the ecosystem adopted the pattern. Archived repositories that inspired stdlib features are not failures — they're victories. Docker → OCI. Left-pad → npm ecosystem. That said — Go hasn't shown interest in this for 5+ years. We're not holding our breath.

## Daily Use

### "What's the CI setup?"

```yaml
# Gate: ensure state is synced
- run: multimod && git diff --quiet || exit 1

# Tests: all modules
- run: multimod go test ./...

# Release: one command
- run: multimod release $VERSION --write --push
```

Three lines. Zero shell scripts. Zero Taskfile dependencies.

### "Do I need a verify command?"

No. `multimod && git diff --quiet` is your verify. Multimod syncs state, git shows if anything changed. Separation of concerns — each tool does what it's best at.

### "What about template generation? Dependabot configs?"

Templates in `.multimod/templates/` run automatically as part of every `multimod` invocation. Not a command — a pipeline step. Add a template, run multimod, get generated files. Remove a template, multimod stops owning those files.

---

*Still angry? [Open an issue.](https://github.com/thumbrise/resilience/issues) We like hard questions.*
