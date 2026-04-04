---
title: "resilience Devlog #5 — The release problem nobody talks about"
description: "How we designed the release flow for a Go multi-module monorepo tool. Dev-state vs publish-state, detached commits, and why main never leaves dev-state."
head:
  - - meta
    - name: keywords
      content: go multi-module release, go monorepo publish, go mod replace strip, detached commit release, semver pre-release, go.work committed
---

# #5 — The release problem nobody talks about

> "Main is the kitchen. Tags are the restaurant menu. The customer never sees the kitchen."

## The contradiction at the heart of Go monorepos

Every Go multi-module monorepo lives a double life.

During development, sub-modules need `replace` directives — without them, `go mod tidy` tries to download your own internal code from the registry. Your code that isn't published yet. Or worse — downloads an ancient version from three months ago.

During publishing, users doing `go get` need clean go.mod files. No `replace => ../`. No `v0.0.0` fake versions. Real dependency versions that resolve from the registry.

Every team that hits this writes shell scripts. OTEL wrote 3000 lines. We wanted one command.

## The dual-state revelation

We were going back and forth — "when do we strip replace? how do we restore it? what if someone forgets?" — until we stopped and looked at it differently.

Every sub-module's go.mod isn't one file with one state. It's **two states of the same file**:

- **Dev-state** — `replace example.com/root => ../` and `require example.com/root v0.0.0`. Committed to git. Always on main.
- **Publish-state** — no replace, `require example.com/root v1.2.3`. Exists only behind a tag. Users see this.

The question isn't "how to switch between states." The question is "where does each state live?"

## The moment everything clicked

We were debating: should the release commit go on main? OTel does two commits — release + restore. But that means main temporarily has publish-state. CI runs between commits. Someone pulls at the wrong moment. Broken dev environment.

Then we asked: **does the publish-state commit need to be on main at all?**

`go get example.com/root@v1.2.3` resolves a **tag**. Not a branch. Not HEAD. Not main. Go finds the tag, downloads the commit behind it, reads go.mod. It has zero knowledge of what branch that commit belongs to.

Branches are for developers. Tags are for users. Two parallel worlds.

This changes everything. The publish-state commit can be **detached** — not on any branch. Just a commit with a tag. Main never knows it exists.

## How it works

```
main:  A → B → C (dev-state, replace present)
                ↑ tag v1.2.3-dev
                \
                 D (publish-state, replace stripped)
                 ↑ tag v1.2.3
```

`multimod release v1.2.3 --write`:

1. Tag current HEAD as `v1.2.3-dev` (traceability — "this main commit became v1.2.3")
2. `git checkout --detach` — physically detach from main
3. Strip replace, pin require to `v1.2.3`
4. Commit: `chore(release): v1.2.3 [multimod]`
5. Tag: `v1.2.3` (root) + `otel/v1.2.3` (each sub)
6. `git checkout main` — back to dev-state, as if nothing happened

Main didn't move. Not one byte changed. The publish-state commit floats in git space, anchored only by its tag. `git gc` won't touch it — tags are first-class refs, same as branches.

## "But detached commits are bad practice!"

We heard this objection. So we researched it. Prometheus accidentally got a detached release tag — it confused contributors who couldn't find the commit in branch history. But that was an **accident**. Ours is a **deliberate choice** with a clear reason.

The real question is: what's worse?

- A detached commit that's invisible in `git log main` but always findable via tag?
- Two commits on main where the first one (publish-state) breaks development for anyone who pulls at the wrong moment?

We chose invisible over broken. The `-dev` tag provides traceability for anyone who needs to find the source commit.

## The `-dev` tag — paired traceability

Every release produces two tags:

```
git tag --list 'v1.2.3*'
v1.2.3-dev    → commit C (main, dev-state)
v1.2.3        → commit D (detached, publish-state)
```

Why `-dev` and not `-rc`? Because this isn't a release candidate in the traditional sense. There are no iterations. No rc.1, rc.2, rc.3. Dev-state is the permanent state of main. `-dev` is honest about what it is.

And it's safe: `-dev` is a semver pre-release. `go get @latest` ignores it. A user would have to explicitly type `@v1.2.3-dev` to get it — and if they do, the broken `v0.0.0` require will fail loudly. No silent bugs.

## Three levels of trust

We borrowed from Terraform's `plan` → `apply` pattern, but added a middle step:

```bash
multimod release v1.2.3                     # dry-run: show the plan
multimod release v1.2.3 --write             # local: create commit + tags
multimod release v1.2.3 --write --push      # CI: create + push
```

The middle step exists because we asked: "what if someone wants to inspect the release locally before pushing?" With `--write`, you get the detached commit and tags on your machine. You can `go get` against them locally. You can `git show v1.2.3` and verify the go.mod looks right. And if something's wrong — `git tag -d v1.2.3` and start over. Main is untouched.

## The chicken-and-egg that spawned unconditional replaces

Early in development, we debated: should we only add replace directives for modules that actually have `require`? Seems cleaner. Less noise in go.mod.

Then we traced the developer workflow:

1. Developer writes `import "example.com/root"` in a sub-module
2. Runs `go mod tidy`
3. Tidy sees the import, adds `require example.com/root`
4. But there's no replace yet — tidy fetches from registry
5. Registry has an old version (or nothing) → **broken**

The replace needs to exist **before** the require. But the require is added automatically by tidy. Chicken, meet egg.

Solution: every sub gets replace for **all** other project modules. Unconditionally. Unused replaces? Go ignores them. But when tidy adds a require — the replace is already waiting. No registry fetch. No stale version. Zero footgun.

## go.work — committed, not generated

"Why not just commit go.work? Then after `git clone` everything works without multimod."

We heard this question and realized: **yes, exactly**. go.work should be committed. It's not a local dev artifact — it's part of the managed state. After clone: IDE works, `go mod tidy` works, `go test` works. No setup step.

Multimod creates and maintains go.work, but go.work is useful even without multimod. It's the first thing that works after clone, before anyone runs any tool.

## Generate dissolved into apply

We initially designed `multimod generate` as a separate command — templates in `.multimod/templates/` producing files like `dependabot.yml` and per-module CI workflows.

Then we asked: "when would someone run generate but not apply?" Never. Generate needs the same State that apply needs. And the output should always be in sync — just like go.work and replace directives.

So generate became a **pipeline step**, not a command. Like EnrichReplaces or EnrichWorkspace. The user doesn't call `multimod enrich-replaces` — they call `multimod` and everything happens. Same with generate.

Templates exist in `.multimod/templates/`? Generate runs. No templates? Nothing happens. Zero config.

## What we learned

1. **Go doesn't care about branches** — only tags matter for publishing. This single insight unlocked the entire release design.
2. **Main should never leave dev-state** — the vulnerability window between "release commit" and "restore commit" is a real footgun. Detached commits eliminate it.
3. **Replace is a dev tool, not a publishing artifact** — committed to git, invisible to users. Unconditional for safety.
4. **Always Apply eliminates verify** — `multimod && git diff --quiet` is the only CI gate needed. Separation of concerns: multimod syncs, git shows diff.
5. **Commands dissolve into pipeline steps** — if something should always happen, it's not a command, it's an invariant.

---

*The release command that never touches main.*
