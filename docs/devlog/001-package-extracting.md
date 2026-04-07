---
title: "resilience Devlog #1 — Package Extracting"
description: "How a sub-package of a GitHub automation daemon became a standalone resilience library — and why Go needs one."
---

# #1 — Package Extracting

> "Every service writes its own retry loop. Nobody asks why."

## The Origin

resilience was born inside [autosolve](https://github.com/thumbrise/autosolve) — a self-hosted Go daemon that polls GitHub repos and dispatches AI agents to solve issues. The daemon needed retry, backoff, and error classification for its polling loop. We built `pkg/longrun` — a 1500-line framework with Task, Runner, Baseline, Policy, ErrorCategory, TransientRule, failureHandler, AttemptStore. Twelve terms for one idea: "if a function fails, maybe try again."

Then we [killed it](https://thumbrise.github.io/autosolve/devlog/013-killing-longrun.html).

The full story lives in the autosolve devlog:
- [#11 — Killing the God Object](https://thumbrise.github.io/autosolve/devlog/011-longrun-god-object.html) — structural review caught a 486-LOC God Object
- [#12 — The Do Primitive](https://thumbrise.github.io/autosolve/devlog/012-resilience-do-vision.html) — the insight that every resilience pattern is `func(ctx, call) error`
- [#13 — Killing longrun](https://thumbrise.github.io/autosolve/devlog/013-killing-longrun.html) — 20 files deleted, 6 files created, the full rewrite

What emerged was `pkg/resilience` — a composable toolkit with one primitive (`Option`), one lifecycle contract (`Plugin`), and zero external dependencies. It worked inside autosolve. But it had nothing to do with autosolve.

## The NIH Check

Before extracting, we checked what already exists in Go. The result was depressing.

**cenkalti/backoff** — stateful `BackOff` interface with `Reset()`. Retry function takes `func() error` — no `context.Context` in the original design, bolted on later. No error matching: retries everything except `Permanent()`. Want to retry `ErrTimeout` with exponential backoff and `ErrRateLimit` with constant backoff? Write a wrapper yourself. No composition with other patterns.

**sony/gobreaker** — circuit breaker state machine. `Execute(func() (interface{}, error))` — no `context.Context` in the signature. Standalone: want retry + circuit breaker? Nest one inside the other manually. No observability hooks. Pre-generics `interface{}` return type.

**eapache/go-resiliency** — retrier + breaker + batcher. Three separate worlds. `func() error` — no context, anywhere. Retrier has no error matching. Breaker doesn't compose with retrier. Each pattern is an island.

**failsafe-go** — the closest competitor. Has composition, context, error matching. But: generic-heavy `Builder[T]()` API, Java-style `.Build()` pattern, policies are structs not functions. Can't write a custom pattern as a simple function — must implement their internal interface. Observability via per-policy callbacks, not a unified plugin system.

The pattern was clear: every library solves one pattern in isolation. None of them compose naturally. None of them separate per-call behavior from shared lifecycle. None of them treat resilience patterns as what they are — **pure algorithms that don't depend on your ecosystem**.

| | cenkalti/backoff | sony/gobreaker | failsafe-go | resilience |
|---|---|---|---|---|
| Custom pattern in 10 lines | no | no | no | `func(ctx, call) error` |
| Compose retry + timeout + circuit | manually | no | builder chain | `With(a, b, c)` |
| Observability without code changes | no | no | per-policy | Plugin interface |
| context.Context by design | no | no | yes | yes |
| Zero deps in core | no | yes | no | yes (target — today mono-module) |
| Error matching | `Permanent()` only | no | yes | `errors.Is` + `errors.As` + `func(error) bool` |
| WaitHint (Retry-After) | no | N/A | no | yes |
| Per-call state (no data races) | N/A | shared mutable | shared mutable | by construction |

## The resilience4j Discovery

After designing our Option/Plugin architecture, we discovered [resilience4j](https://resilience4j.readme.io/) — the Java library that catalogs the same patterns we envisioned. Circuit breaker, retry, rate limiter, bulkhead, timeout — all as composable decorators.

Our design converged independently to the same conclusions:
- Every pattern wraps a function call
- Patterns compose as a pipeline
- Observability is separate from control flow
- Algorithms are pure — storage is pluggable

But resilience4j is Java. Spring Cloud. Annotation processors. Config objects. Registry pattern. `Supplier<T>` decorators with non-obvious nesting order. The Go ecosystem has nothing equivalent — not because the problem doesn't exist, but because nobody built it with Go idioms.

We didn't port resilience4j. We arrived at the same destination from a different direction. Our primitive — `func(ctx context.Context, call func(context.Context) error) error` — is simpler and more expressive than Java's decorator pattern. Go has closures as first-class citizens, `context.Context` by convention, and a culture of small composable interfaces. The same algorithms, different instrument.

## What resilience Actually Is

Not a retry library. Not a circuit breaker. Not an OTEL wrapper.

resilience is a **composable toolkit of objective resilience algorithms** with a Go-native DSL.

The core insight: resilience patterns are not framework features. They are **algorithms**. Like sorting. Like hashing. Exponential backoff is math — `initial * multiplier^attempt`. Circuit breaker is a state machine — three states, two thresholds, one timeout. Token bucket rate limiter is a counter with a refill rate. These have known inputs, known outputs, known edge cases, and known optimal implementations. They don't depend on whether you use gRPC or HTTP, Redis or Postgres, Kubernetes or bare metal.

Nobody reinvents bubble sort in every service. But every service reinvents its retry loop. Then writes tests. Then finds bugs. Then fixes them. In four different places, because four services have four implementations of the same algorithm.

What depends on your ecosystem is the **wiring** — how you compose patterns, where you store shared state, how you observe them. That's what the DSL handles:

```
Algorithms (pure, testable, eternal):
  backoff.Exponential, backoff.Constant
  retry loop, circuit breaker state machine
  bulkhead semaphore, token bucket rate limiter

DSL (Go-native, composable):
  Option — per-call behavior, wraps execution
  Plugin — shared lifecycle, observes execution
  Client — holds plugins, creates call builders
  Do     — stateless shortcut for one-off calls

Integrations (opt-in, target: separate modules):
  otel/  — OTEL metrics plugin
  grpc/  — gRPC middleware (future)
  http/  — HTTP client wrapper (future)
```

The architecture is designed so that any pattern — existing or not yet invented — is a `func(ctx, call) error`. Circuit breaker in 15 lines:

```go
func Breaker(cb *CircuitBreaker) resilience.Option {
    return func(ctx context.Context, call func(context.Context) error) error {
        if !cb.Allow() {
            return ErrCircuitOpen
        }
        err := call(ctx)
        cb.Record(err)
        return err
    }
}
```

The algorithm (`CircuitBreaker` state machine) is pure — write once, test a billion times, plug any storage backend. The Option is the adapter that makes it composable with retry, timeout, bulkhead, and anything else. One line in `With()`, pattern appears. Remove the line, pattern disappears.

## Why Extract

`pkg/resilience` inside autosolve was already independent — zero imports from `internal/`, zero dependency on the daemon's domain. But keeping it inside autosolve meant:

- Users must `go get` the entire autosolve module to use resilience
- The package version is tied to autosolve releases
- Contributors must understand autosolve to contribute to resilience
- The `go.sum` pulls autosolve's dependencies (SQLite, Cobra, Wire, etc.)

Extraction into `github.com/thumbrise/resilience` gives:
- Independent versioning, independent CI, independent contributors
- Architecture ready for future sub-packages (`circuit/`, `bulkhead/`, `grpc/`)
- Target: zero-dependency core via separate Go modules (blocked on [multimod](/devlog/003-multimod-gap) — today everything lives in one `go.mod`)

## What's Here Now

640 lines of working code. Not a vision document — working, tested code:

- **Option** as `func(ctx, call) error` — the universal primitive
- **Plugin + Events** — shared lifecycle, observe-only (YAGNI: no control flow yet)
- **Client + CallBuilder** — stateful API with per-call fresh state (no data races by construction)
- **Do()** — stateless shortcut for one-off calls
- **retry.On** with `errors.Is` + `errors.As` + `OnFunc` — error matching that actually works
- **backoff.Func** as `func(attempt int) time.Duration` — pure math, open/closed forever
- **WithWaitHint** — application-level override for server-suggested delays (Retry-After)
- **rsotel.Plugin()** — OTEL metrics plugin (sub-package today, separate module planned)

This is Phase 1. A house is not weak because it doesn't have a second floor yet.

## What's Next

Phase 2: `GuardPlugin` interface for control flow (circuit breaker, rate limiter). Pure algorithms in `algorithm/`, pluggable storage backends (memory, Redis, Postgres). Zero breaking changes — `GuardPlugin` extends `Plugin`, existing code untouched.

Phase 3: Presets, integrations, ecosystem. `preset.HTTP()`, `preset.GRPC()`. Community backends. The algorithms will be written once. Tested once. Used everywhere.

Because `2 + 2 = 4` doesn't depend on your infrastructure.

---

*The package that outgrew its parent.*
