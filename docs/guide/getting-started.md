---
title: Getting Started
description: Install the open source Go (golang) fault tolerance library. Retry, backoff, error handling middleware — first resilient call in under a minute.
---

# Getting Started

## Install

```bash
go get github.com/thumbrise/resilience
```

::: warning Mono-module today
Currently, the entire project (core + OTEL + multimod CLI) lives in one Go module. `go get` pulls all transitive dependencies, including the OTEL SDK. Zero-dependency core via separate Go modules is the [target architecture](/devlog/003-multimod-gap), blocked on multimod tooling.
:::

## Your first resilient call

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/thumbrise/resilience"
    "github.com/thumbrise/resilience/backoff"
    "github.com/thumbrise/resilience/retry"
)

var ErrTimeout = errors.New("timeout")

func main() {
    ctx := context.Background()

    err := resilience.Do(ctx, func(ctx context.Context) error {
        // Your unreliable call here.
        return ErrTimeout
    },
        retry.On(ErrTimeout, 3, backoff.Exponential(1*time.Second, 30*time.Second)),
    )

    fmt.Println(err) // timeout (after 3 retries)
}
```

That's it. One function, one Option, done.

## With a Client (OTEL observability)

```go
import rsotel "github.com/thumbrise/resilience/otel"

// Create once, pass everywhere — like http.Client.
client := resilience.NewClient(rsotel.Plugin())

err := client.Call(fn).
    With(retry.On(ErrTimeout, 3, backoff.Exponential(1*time.Second, 30*time.Second))).
    With(retry.On(ErrRateLimit, 5, backoff.Constant(10*time.Second))).
    Do(ctx)
```

Every call through this Client emits OTEL metrics automatically:
`resilience.call.total`, `resilience.call.duration_seconds`, `resilience.call.errors`,
`resilience.retry.total`, `resilience.retry.wait_seconds`.

## Core concepts

| Concept | What | Lifecycle |
|---|---|---|
| **Option** | `func(ctx, call) error` — wraps a call with behavior | Per-call, fresh each `Do()` |
| **Plugin** | Interface with `Name()` + `Events()` — observes calls | Client-level, shared state |
| **Client** | Holds Plugins, creates CallBuilders | Application-wide, immutable |
| **Do** | Stateless shortcut — no Client needed | One-off calls |

## Architecture

Two levels of configuration, two extension points:

```
Client (application-wide)          CallBuilder (per-call)
├── Plugin: OTEL metrics           ├── Option: retry
├── Plugin: circuit breaker (*)    ├── Option: timeout (*)
└── Plugin: logging (*)            └── Option: rate limit (*)

(*) planned — not yet implemented
```

**Client** — immutable, thread-safe, one per application. Holds Plugins with shared state.

**CallBuilder** — per-call, fresh on every `Call()`. Holds Options with per-call state.

## Design principles

- **Option is the universal primitive** — `func(ctx, call) error`. Full control. Any pattern.
- **Plugin observes, Option controls** — two contracts, two lifecycles, no confusion.
- **Per-call state** — Options are fresh on every `Do()`. No shared mutable state. No data races.
- **Events via context** — Plugins attach Events to context. Options extract if needed. No coupling.
- **Backoff is pure math** — `func(attempt int) time.Duration`. Open/closed forever.

## What's next

- [Retry](/guide/retry) — error matching, budgets, WaitHint
- [Backoff](/guide/backoff) — Exponential, Constant, custom
- [Observability (OTEL)](/guide/otel) — metrics plugin, one line setup
- [Options & Plugins](/guide/options-plugins) — want to write your own patterns? Start here
- [Roadmap](/guide/roadmap) — what's ready, what's next, what's on the horizon
- [Origin story](/devlog/001-package-extracting) — how this package was born
