---
title: Getting Started
description: Install the resilience library and make your first resilient call in under a minute.
---

# Getting Started

## Install

```bash
go get github.com/thumbrise/resilience
```

Core module has **zero external dependencies**. Your `go.sum` stays clean.

Want OTEL metrics? Separate module, separate `go get`:

```bash
go get github.com/thumbrise/resilience/otel
```

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

## What's next

- [Retry docs](https://pkg.go.dev/github.com/thumbrise/resilience/retry) — error matching, budgets, WaitHint
- [Backoff docs](https://pkg.go.dev/github.com/thumbrise/resilience/backoff) — Exponential, Constant, custom
- [OTEL docs](https://pkg.go.dev/github.com/thumbrise/resilience/otel) — metrics plugin
- [Origin story](https://thumbrise.github.io/autosolve/devlog/013-killing-longrun.html) — how this package was born
