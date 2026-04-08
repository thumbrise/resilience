---
title: "resilience Devlog #7 — The architecture fight"
description: "How we stress-tested a release design through adversarial debate. Two AI personas, one arbiter, six rounds, and an RFC that survived."
head:
  - - meta
    - name: keywords
      content: adversarial architecture review, go multi-module design, detached commit debate, semantic-release incompatibility, RFC design process
---

# #7 — The architecture fight

> "The best way to test an architecture is to let someone try to destroy it."

## The setup

April 7, 2026. We had a working multimod — discovery, applier, go proxy, release command. Devlog #5 documented the detached commit design. Everything looked clean on paper.

Too clean. That's suspicious.

We had an architecture that nobody had seriously challenged. Internal consistency is easy — you agree with yourself by default. What we needed was someone who genuinely wanted to tear it apart.

So we built a fight club.

## The format

Three roles. Two AI personas with opposing prompts. One human arbiter.

**The Skeptic** got a prompt to be a ruthless pragmatist — Torvalds-style. "Find every weakness. Assume NIH. Assume overengineering. Don't be polite." The goal: destroy the architecture or prove it can't be destroyed.

**The Implementor** defended the design. Not blindly — if the Skeptic found a real flaw, the Implementor had to acknowledge it. But the default stance was: "this works, here's why."

**The Arbiter** (me, the human) watched the fight, intervened when it went circular, and sent disputed facts to DeepSeek for independent verification. The arbiter's word was final. No appeals.

The rules were simple:

1. Each round: Skeptic attacks, Implementor defends, Arbiter judges
2. Factual claims get verified externally — no "I think" allowed
3. If neither side convinces the Arbiter, the point is logged as "disputed" and moves to the RFC as a known limitation
4. The fight continues until the Skeptic runs out of ammunition or the architecture collapses

It ran for six rounds. The architecture survived. But not unchanged.

## Round 1: "Just use semantic-release"

The Skeptic opened with the obvious move: why build release tooling when semantic-release exists?

We expected this. We'd already tried semantic-release. But the Skeptic pushed harder than we had — and found the real reason it doesn't work. Not a configuration issue. A fundamental architectural incompatibility.

semantic-release tags the current branch. In our model, main is always in dev-state — `go.mod` files have `replace ../` directives. Tag main → users `go get` broken go.mod. That's the surface problem.

The deeper problem: semantic-release uses `git tag --merged` to find previous versions and compute the next one. Our detached commits are not reachable from main. The version chain breaks. semantic-release literally cannot see its own previous releases.

We sent this to DeepSeek for verification. Confirmed: `git tag --merged main` does not include tags on detached commits. This isn't a plugin gap — it's how git works.

The Node ecosystem already knew this. Vercel, Chakra UI, Radix — they all moved from semantic-release to changesets for monorepo use cases. Same architectural mismatch, different ecosystem.

**Arbiter ruling:** semantic-release is incompatible. Not "hard to configure" — incompatible. Decision D3 in the RFC.

## Round 2: "Detached commits are a hack"

This was the longest fight. Three sub-rounds.

**Skeptic's opening:** Prometheus accidentally got a detached release tag — chaos ensued. What about `git gc`? What about GitHub Archive? What about developers who can't find the commit in `git log`?

**Implementor's defense:** tags are first-class refs. `git gc` never collects tagged commits. Go proxy caches permanently after first fetch — even if the tag is deleted from the repo. The `-dev` tag on main provides traceability.

The Skeptic pivoted: "Fine, it works technically. But release branches are better. You get branch protection, you can amend, you have a clear audit trail."

This is where it got interesting. The Implementor asked: "What happens when you need a hotfix on a release branch?"

Silence. Then the Skeptic worked it out: the release branch contains publish-state (no replaces, pinned requires). To hotfix, you need to restore dev-state on the branch, make the fix, then re-publish. The branch alternates between two states. State mixing on a single branch.

With detached commits: any branch (main, release/v1.2, feature/x) is always in dev-state. Hotfix on the branch → new detached commit. No state mixing. Ever.

**The LTS moment.** The Skeptic's last card: "What about LTS? You need a release/v1.2 branch for long-term support. Detached commits can't do that."

I (the Arbiter) broke this one with a single sentence: "The LTS branch contains dev-state. Releases are detached from its HEAD. Same flow everywhere."

```
main (dev-state)           → detached v1.3.0
release/v1.2 (dev-state)   → detached v1.2.4
```

The Skeptic conceded. Not because detached commits are perfect — they're a trade-off. No amend, limited tag protection. But the alternative (state mixing) is worse.

**Arbiter ruling:** detached commit is a design trade-off, not a hack. The trade-off favors us. Decision D1.

## Round 3: "Four Go binaries is not Unix philosophy"

The Skeptic got creative: "Unix utilities weigh kilobytes. Each Go binary is 10-15MB. Four tools = 50MB. You're calling this Unix-way? That's hypocrisy."

The Implementor started defending binary size. Wrong move. The Arbiter intervened: "Binary size is irrelevant in 2026. What's the real question?"

The real question: **do the tools share domain knowledge?** If yes — they should be one binary. If no — separate binaries are correct.

We audited. multirelease has `internalPaths()` — is that duplicated domain knowledge from multimod? No. It's derivation from input data. Like `wc` counting lines from stdin — `wc` doesn't share domain knowledge with `cat`.

But then the Skeptic found something real: the `_` prefix convention. multirelease checks `strings.HasPrefix(filepath.Base(rel), "_")` to skip workspace-only modules from tagging. That's multimod's convention leaking into multirelease. Shared knowledge.

**Resolution:** add `workspace_only` to the JSON contract. multimod marks modules, multirelease reads the flag. The convention lives in one place. This was a genuine find — the Skeptic earned it.

**Arbiter ruling:** separate binaries are correct. `workspace_only` field added to the contract. Decision D2 + D6.

## Round 4: "Zero-config doesn't scale"

The Skeptic brought OTEL: 40+ modules, module sets with different lifecycles (stable-v1, experimental-metrics, bridge). "Your 'release all together' doesn't work for them. Where's your grouping mechanism?"

The Implementor tried to dismiss it. The Arbiter stopped that: "The Skeptic is right that it doesn't scale to 40 modules. The question is whether that matters today."

We traced our niche: core library + optional extensions. 2-10 modules, uniform lifecycle. resilience has root + otel. That's two. Even ambitious projects in this niche rarely exceed 10.

But the Skeptic landed a philosophical hit: "Convention without enforcement is documentation, not architecture." Fair point. The `_` prefix for workspace-only modules — is it enforced or just documented?

It's enforced. multimod classifies `_`-prefixed dirs as workspace-only. The release tool skips them from tagging. This is tool enforcement, analogous to how Go's compiler enforces `internal/`. Not a README rule — a code rule.

**Arbiter ruling:** accepted as future limitation, rejected as current blocker. YAGNI today, path not closed. Decision D4.

## Round 5: "You compete with Go toolchain"

Short round. The Skeptic argued `go work` and `go mod tidy` already solve parts of this.

True. But `go work` requires manual `go work use ./otel`. No auto-discovery. `go mod tidy` doesn't sync Go version across modules. Neither knows about releases. We complement the toolchain, not compete with it.

We use `golang.org/x/mod/modfile` — the official Go library for parsing go.mod. Correct side of the API boundary. If Go adds built-in multi-module release support, the ecosystem has served its purpose.

**Arbiter ruling:** conscious risk accepted. Decision D8 (`pkg/` → `cmd/` for loose coupling) makes the tools non-importable — if Go absorbs the functionality, users switch without breaking anything.

## The NIH audit

After six rounds, the Skeptic was out of architectural ammunition. But we weren't done. The nagging question remained: "Are we sure nobody else built this?"

We sent DeepSeek on a deep search. Five tools came back:

- **Crosslink** (OTEL build-tools) — partial replace sync, tied to OTEL namespace
- **Gorepomod** (Kustomize/SIG) — pin/unpin model (same two-state concept!), dead project (6+ years without release)
- **kimono** (bonzai) — dev convenience tool, no release capability
- **monorel** (The Root Company) — binary release tool (goreleaser wrapper), not for libraries
- **OTEL multimod** — our namesake, config-driven vs our convention-driven

The Gorepomod discovery was the most interesting. Their `pin`/`unpin` commands are exactly our dev-state/publish-state — different names, same concept. Independently discovered by the Kubernetes ecosystem. Confirms the pattern is real.

But Gorepomod uses release branches (state mixing), requires manual config, and hasn't been maintained since ~2020. The right model, abandoned implementation.

**Conclusion:** no existing tool covers our full lifecycle. The gap is real, verified by independent search.

## The cold review

The fight was over. We had an RFC draft — 500+ lines. But drafts written in the heat of debate have blind spots. We needed cold eyes.

DeepSeek got the RFC with one instruction: "Review this as if you're deciding whether to adopt this tool for your Go monorepo. Find every gap, every unstated assumption, every missing edge case."

It found one critical gap we'd completely missed: **publish-state was never tested.**

The entire release flow — strip replaces, pin requires, commit, tag — never verified that the resulting go.mod files actually work. What if stripping a replace breaks a build? What if a pinned version doesn't exist on the proxy yet? The detached commit would be tagged, pushed, cached by the Go proxy permanently — and broken.

Solution: `GOWORK=off go build ./...` in each transformed module, after transform, before commit. If any module fails to build in isolation — abort, rollback, clear error. Publish-state must be proven buildable before it becomes a tag.

This wasn't a Skeptic attack or an Implementor defense. This was a quiet, methodical read by a third party who wasn't invested in either side. Sometimes the best review comes from someone who wasn't in the room.

## The process insight

We discovered something about how adversarial review works:

**The Skeptic finds structural flaws.** "Your model doesn't handle X" — these are the architectural challenges. Detached commits, semantic-release incompatibility, Unix philosophy. Big moves.

**The cold reviewer finds operational gaps.** "You never test the output" — these are the things everyone assumes work because the architecture is sound. The architecture can be perfect and the implementation still broken.

**The Arbiter prevents circular arguments.** Without a human with final authority, AI debates go in circles. The Skeptic repeats the same point with different words. The Implementor defends with increasing verbosity. The Arbiter says "this is decided, move on" — and the debate advances.

The most powerful tool in the Arbiter's arsenal wasn't expertise — it was the ability to say "send this to DeepSeek for fact-checking." Disputed claims get resolved by evidence, not rhetoric.

## What changed

The architecture survived. But the RFC that came out is different from what went in:

1. **semantic-release dropped** — not "maybe later," but "fundamentally incompatible." We would have kept it as a TODO otherwise.
2. **`workspace_only` in JSON contract** — the `_` prefix convention was leaking across tool boundaries. Now it's explicit data.
3. **Publish-state validation** — `GOWORK=off go build` before commit. The most important addition, found by the cold review.
4. **`pkg/` → `cmd/`** — types are not importable. JSON is the only interface. Architectural enforcement, not documentation.
5. **Explicit tag push** — `git push origin --tags` pushes ALL local tags. Must push specific tags. Found as a bug during the detached commit debate.
6. **Target niche documented** — "core + optional extensions," not "any monorepo." The Skeptic forced us to be honest about scope.
7. **Evidence base** — 9 verified public links. Every claim about "real projects struggling" is backed by a GitHub issue or blog post.

Eight decisions in the Decision Log. Ten known limitations. Zero hand-waving.

## What we learned about the process

1. **Adversarial review works** — but only with an arbiter. Two AIs debating without a human judge produces heat, not light.
2. **Fact-checking kills rhetoric** — "I think semantic-release can't do X" vs "DeepSeek confirmed: `git tag --merged` does not include detached commits." Night and day.
3. **The cold reviewer finds what the fighters miss** — the publish-state validation gap was invisible to both Skeptic and Implementor. Fresh eyes see different things.
4. **Concessions are features, not bugs** — the Skeptic's valid points (workspace_only, zero-config scaling, convention enforcement) made the RFC stronger. Acknowledging weakness is not losing — it's building trust.
5. **An RFC is a living document** — ours has a "Disputed Points" section that documents the arguments honestly. Future contributors see not just the decision, but the reasoning and the counterarguments.

---

*The architecture that survived the fight is stronger than the one that went in.*
