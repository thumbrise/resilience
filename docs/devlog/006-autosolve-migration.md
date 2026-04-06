---
title: "resilience Devlog #6 — autosolve migrated to standalone"
description: "The package that outgrew its parent is now the dependency. autosolve switched from internal pkg/resilience to the standalone library. The extraction thesis proved."
head:
  - - meta
    - name: keywords
      content: go package extraction, golang monorepo to standalone, open source library extraction, autosolve resilience migration
---

# #6 — autosolve migrated to standalone

> "The child became the dependency."

## The Trigger

[autosolve](https://github.com/thumbrise/autosolve) — the project where resilience was born — has switched from its internal `pkg/resilience` to `github.com/thumbrise/resilience`. The internal package is gone, replaced with meta-refresh redirects pointing here.

The code is identical. The import path changed. That's it.

## Why This Matters

In [devlog #1](/devlog/001-package-extracting) we wrote:

> resilience was born inside autosolve. But it had nothing to do with autosolve.

That was the thesis. This migration is the proof.

autosolve is now a **consumer** of resilience, not the owner. The interface boundary we designed — `Option`, `Plugin`, `Client`, `Do` — survived extraction without a single API change. No adapters, no shims, no "compatibility layer." The abstraction was correct.

What changed:

- Import paths: `autosolve/pkg/resilience` → `github.com/thumbrise/resilience`
- Zero changes to autosolve's business logic
- Zero API changes in resilience
- Independent CI, independent versioning, independent contributors

What didn't change: everything that matters.

## The NIH Vindication

In devlog #1, we spent a section on why existing Go libraries (cenkalti/backoff, sony/gobreaker, failsafe-go) didn't fit. Some readers might have thought: "just use failsafe-go and move on."

Five devlogs later, the library that started as "maybe we should build our own" has:

- A composable DSL with one primitive (`func(ctx, call) error`)
- Per-error retry budgets with `errors.Is` + `errors.As` + custom matchers
- Pure backoff functions (open/closed forever — `func(int) time.Duration`)
- WaitHint for server-suggested delays (Retry-After)
- An OTEL plugin that's one line to enable
- A real consumer (autosolve) that validates the API daily

640 lines. The NIH research wasn't paranoia — it was due diligence.

## The Honest Status

The documentation you're reading is the result of this migration. Adapted from autosolve's internal docs, cleaned up, and made honest:

- **Ready:** retry, backoff, OTEL plugin, Option/Plugin architecture
- **Blocked:** zero-dependency core — today everything lives in one `go.mod`, OTEL SDK included. Waiting on [multimod](/devlog/003-multimod-gap) to split modules
- **Planned:** timeout, circuit breaker, presets — see the [roadmap](/guide/roadmap)

The monomodule problem remains. `go get github.com/thumbrise/resilience` pulls the OTEL SDK transitively. That gets solved when multimod ships its `release` command. One problem at a time.

## The Pattern Continues

| Problem | Status | Primitive |
|---|---|---|
| Resilience patterns in Go | ✅ Extracted, standalone | `func(ctx, call) error` |
| Task runner lifecycle | 📝 Documented | `BeforeAll` / phases |
| Multi-module release | 🔨 Building | `Discovery → State → Applier` |
| Package extraction | ✅ Proved | autosolve → resilience |

Each entry: concrete pain → honest research → minimal primitive → extract when ready.

---

*The package that outgrew its parent is now its dependency. The circle is complete.*
