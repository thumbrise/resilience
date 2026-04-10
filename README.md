# resilience

[![CI](https://github.com/thumbrise/resilience/actions/workflows/ci.yml/badge.svg)](https://github.com/thumbrise/resilience/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/thumbrise/resilience.svg)](https://pkg.go.dev/github.com/thumbrise/resilience)
[![Latest Release](https://img.shields.io/github/v/release/thumbrise/resilience?label=latest&color=blue)](https://github.com/thumbrise/resilience/releases)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](/LICENSE)
[![Coverage Status](https://coveralls.io/repos/github/thumbrise/resilience/badge.svg?branch=main)](https://coveralls.io/github/thumbrise/resilience?branch=main)

Composable resilience for Go function calls.

It all started with [killing a 1500-line framework](https://thumbrise.github.io/resilience/devlog/001-package-extracting). We had 20 files, 12 terms, and a God Object — all for one idea: "if a function fails, maybe try again." We deleted everything and replaced it with one type: `func(ctx, call) error`. That type covers retry, backoff, and any future resilience pattern — composed like Lego.

> **Mono-module today.** The project is currently a single Go module. `go get` pulls all transitive dependencies including the OTEL SDK. Zero-dependency core via separate Go modules is the [target architecture](https://thumbrise.github.io/resilience/devlog/003-multimod-gap), blocked on [multimod](https://github.com/thumbrise/multimod) — a standalone tool extracted from this repository.

## Install

```bash
go get github.com/thumbrise/resilience
```

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

## The Primitive

Every resilience pattern has the same shape:

```go
type Option func(ctx context.Context, call func(context.Context) error) error
```

Retry, timeout, circuit breaker, bulkhead — any pattern is a function that wraps a call. The architecture is designed so that future patterns (timeout, circuit breaker, rate limiter) are the same `Option` type. Write your own in 5–15 lines:

```go
// Timeout — not shipped yet, but this is all it takes:
func Timeout(d time.Duration) resilience.Option {
    return func(ctx context.Context, call func(context.Context) error) error {
        ctx, cancel := context.WithTimeout(ctx, d)
        defer cancel()
        return call(ctx)
    }
}
```

See the [roadmap](https://thumbrise.github.io/resilience/guide/roadmap) for what's ready and what's planned.

## Two Extension Points

| | Option | Plugin |
|---|---|---|
| **What** | `func(ctx, call) error` | Interface: `Name()` + `Events()` |
| **Controls execution?** | Yes — wraps the call | No — observes only |
| **State** | Per-call, fresh each `Do()` | Shared across calls |
| **Use for** | retry (ready), timeout/circuit/bulkhead (planned) | metrics (OTEL — ready), logging (planned) |

The compiler enforces the boundary: `With()` accepts Option, `NewClient()` accepts Plugin. Can't mix them up.

## Packages

| Package | What's inside |
|---|---|
| `resilience` | Core: `Do`, `Client`, `Option`, `Plugin`, `Events` |
| `resilience/backoff` | `Exponential`, `ExponentialWith`, `Constant`, `Default` |
| `resilience/retry` | `On`, `OnFunc`, `WithWaitHint` |
| `resilience/otel` | `Plugin()` — OTEL metrics |

All packages live in a single Go module today. See [devlog #3](https://thumbrise.github.io/resilience/devlog/003-multimod-gap) for the multi-module plan.

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

## Documentation

**[thumbrise.github.io/resilience](https://thumbrise.github.io/resilience/)** — full docs, guide, devlog.

- [Getting Started](https://thumbrise.github.io/resilience/guide/getting-started) — install, first call, core concepts
- [Retry](https://thumbrise.github.io/resilience/guide/retry) — error matching, budgets, WaitHint
- [Backoff](https://thumbrise.github.io/resilience/guide/backoff) — Exponential, Constant, custom
- [OTEL](https://thumbrise.github.io/resilience/guide/otel) — metrics plugin, one line setup
- [Roadmap](https://thumbrise.github.io/resilience/guide/roadmap) — what's ready, what's next, what's on the horizon
- [Devlog](https://thumbrise.github.io/resilience/devlog/) — design decisions, dead ends, lessons learned

## License

Apache 2.0
