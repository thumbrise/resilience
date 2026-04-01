# resilience

[![CI](https://github.com/thumbrise/resilience/actions/workflows/ci.yml/badge.svg)](https://github.com/thumbrise/resilience/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/thumbrise/resilience.svg)](https://pkg.go.dev/github.com/thumbrise/resilience)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](/LICENSE)

Composable resilience for Go function calls. Zero-dependency core.

It all started with [killing a 1500-line framework](https://thumbrise.github.io/autosolve/devlog/013-killing-longrun.html). We had 20 files, 12 terms, and a God Object — all for one idea: "if a function fails, maybe try again." We deleted everything and replaced it with one type: `func(ctx, call) error`. That type covers retry, timeout, circuit breaker, bulkhead, hedge, rate limiter — any resilience pattern, composed like Lego.

## Install

```go
import "github.com/thumbrise/resilience"
import "github.com/thumbrise/resilience/retry"
import "github.com/thumbrise/resilience/backoff"

// OTEL metrics — separate module, opt-in:
import rsotel "github.com/thumbrise/resilience/otel"
```

Core has **zero external dependencies**. Your `go.sum` stays clean.

## Quick Start

```go
// One-off call — no setup needed
err := resilience.Do(ctx, callAPI,
    retry.On(ErrTimeout, 3, backoff.Exponential(1*time.Second, 30*time.Second)),
)

// Multiple retry rules — different errors, different strategies
err := resilience.Do(ctx, callAPI,
    retry.On(ErrTimeout, 3, backoff.Exponential(1*time.Second, 30*time.Second)),
    retry.On(ErrRateLimit, 5, backoff.Constant(10*time.Second),
        retry.WithWaitHint(extractRetryAfter),  // Retry-After header overrides backoff
    ),
)

// Client with OTEL observability — create once, pass everywhere
client := resilience.NewClient(rsotel.Plugin())

err := client.Call(callAPI).
    With(retry.On(ErrTimeout, 3, backoff.Exponential(1*time.Second, 30*time.Second))).
    Do(ctx)
```

## Any Pattern Is One Function

Every resilience pattern has the same shape:

```go
type Option func(ctx context.Context, call func(context.Context) error) error
```

This isn't just retry. Circuit breaker? 15 lines:

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

Timeout? 5 lines. Bulkhead? A semaphore. Rate limiter? A token bucket. Each is a function. Each composes with everything else. Add a line — pattern appears. Remove a line — pattern disappears.

```go
client.Call(fn).
    With(timeout.After(5*time.Second)).   // deadline
    With(retry.On(ErrTimeout, 3, bo)).    // retry on timeout
    With(Breaker(cb)).                    // circuit breaker
    Do(ctx)
```

Reads top-to-bottom as an execution pipeline.

## Two Extension Points

| | Option | Plugin |
|---|---|---|
| **What** | `func(ctx, call) error` | Interface: `Name()` + `Events()` |
| **Controls execution?** | Yes — wraps the call | No — observes only |
| **State** | Per-call, fresh each `Do()` | Shared across calls |
| **Use for** | retry, timeout, circuit, bulkhead | metrics, logging, tracing |

The compiler enforces the boundary: `With()` accepts Option, `NewClient()` accepts Plugin. Can't mix them up.

## Packages

| Package | Description | Dependencies |
|---|---|---|
| `resilience` | Core: `Do`, `Client`, `Option`, `Plugin`, `Events` | **none** |
| `resilience/backoff` | `Exponential`, `Constant`, `Default` | **none** |
| `resilience/retry` | `On`, `OnFunc`, `WithWaitHint` | **none** |
| `resilience/otel` | `Plugin()` — OTEL metrics (separate module) | `go.opentelemetry.io/otel` |

> **Note:** The project is currently a single module with `otel/` as the only separate sub-module (to keep core zero-deps). Full multi-module isolation is planned — see [devlog](https://thumbrise.github.io/resilience/devlog/) for the roadmap.

## How It Compares

| | cenkalti/backoff | sony/gobreaker | failsafe-go | **resilience** |
|---|---|---|---|---|
| Custom pattern in 10 lines | no | no | no | `func(ctx, call) error` |
| Compose retry + timeout + circuit | manually | no | builder chain | `With(a, b, c)` |
| Observability without code changes | no | no | per-policy | Plugin interface |
| `context.Context` by design | bolted on | no | yes | **yes** |
| Error matching | `Permanent()` only | no | yes | `errors.Is` + `errors.As` + custom |
| WaitHint (Retry-After) | no | N/A | no | **yes** |
| Per-call state (no data races) | N/A | shared mutable | shared mutable | **by construction** |
| Zero deps in core | no | yes | no | **yes** |

## Documentation

**[thumbrise.github.io/resilience](https://thumbrise.github.io/resilience/)** — full docs, architecture, devlog.

The README is a quick-start. The docs are the real thing:
- [Getting Started](https://thumbrise.github.io/resilience/guide/getting-started) — install, first call, core concepts
- [Devlog](https://thumbrise.github.io/resilience/devlog/) — design decisions, dead ends, lessons learned

## Independent Convergence

After designing the Option/Plugin architecture, we discovered [resilience4j](https://resilience4j.readme.io/) (Java) — a library that catalogs the same patterns. Same algorithms, same composability conclusions, arrived at independently. We didn't port it and didn't know about it during design. The convergence confirms the patterns are objective, not invented.

## License

Apache 2.0
