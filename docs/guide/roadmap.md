---
title: "resilience: Roadmap"
description: "What's ready, what's next, what's on the horizon — and what we won't do."
head:
  - - meta
    - name: keywords
      content: go resilience roadmap, golang circuit breaker, go rate limiter, go bulkhead, resilience patterns go, go timeout middleware
---

# Roadmap

What's ready, what's next, what's on the horizon. No deadlines. When a real use case demands it — we build it.

## ✅ Ready now

Battle-tested, shipped, documented:

- **retry.On / retry.OnFunc** — error matching via `errors.Is`, `errors.As`, custom `func(error) bool`
- **backoff.Func** — Exponential, ExponentialWith, Constant, Default, or write your own
- **retry.WithWaitHint** — server-suggested delays (Retry-After) override backoff
- **Option / Plugin / Client / CallBuilder** — the composable DSL
- **rsotel.Plugin()** — OTEL metrics for calls, errors, retries, waits
- **resilience.Do()** — stateless shortcut for one-off calls

This is enough to cover the most common resilience need: retry with backoff and observability.

## 🔜 Next up

Small, realistic, unblocked:

- **timeout** — `timeout.After(5*time.Second)` — per-call deadline. The simplest possible Option. ~10 lines.
- **Presets** — tested combinations of Options with metadata. `preset.HTTP("github")` — a recipe, not a bag of Options. Core can introspect, validate conflicts, log what was applied.
- **Core validation** — conflict detection at `Do()` time. Two retry Options matching the same error? Timeout shorter than backoff max? Fail fast, not at 3am.

## 🔮 Horizon

When a real use case demands it:

- **circuit** — `circuit.Breaker("github")` — state machine via Plugin, per-call check via Option
- **bulkhead** — `bulkhead.Max(20)` — concurrency semaphore
- **hedge** — `hedge.After(100*ms)` — speculative parallel call
- **ratelimit** — `ratelimit.Wait(limiter)` — token bucket before call
- **fallback** — `fallback.To(backupFn)` — alternative on failure
- **Introspection** — `resilience.Dump(client, opts...)` — describe the pipeline before running
- **Plugin toolkit** — helpers for common plugin patterns (`plugin.BeforeCall`, `plugin.AfterCall`)

Each pattern is `func(ctx, call) error`. Each is an independent sub-package. Zero coupling between patterns. Community can publish their own.

## 🧱 Infrastructure: multimod

The [multi-module gap](/devlog/003-multimod-gap) blocks the target architecture: zero-dependency core, each plugin in its own Go module. Today everything lives in one `go.mod`. When multimod ships its `release` command, we split modules. No breaking API changes — just `go.mod` boundaries.

## What we won't do

- **Config file for resilience** — resilience is code, not YAML. Patterns are engineering decisions.
- **Global state** — no package-level defaults, no `init()`. Client is explicit.
- **Magic** — no auto-detection of error types, no implicit retry. Every behavior is opt-in.
