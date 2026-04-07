---
title: "resilience: Observability (OTEL)"
description: "rsotel.Plugin() — OpenTelemetry metrics for resilience calls, errors, retries, and backoff waits."
head:
  - - meta
    - name: keywords
      content: go opentelemetry plugin, golang otel metrics, resilience observability, go retry metrics, backoff metrics otel
---

# Observability (OTEL)

`resilience/otel` provides an OpenTelemetry Plugin for the resilience package.

::: info Same module for now
OTEL is a sub-package inside the same Go module as core. Separate `go.mod` for otel is the [target architecture](/devlog/003-multimod-gap), blocked on multimod.
:::

## Usage

```go
import rsotel "github.com/thumbrise/resilience/otel"

client := resilience.NewClient(rsotel.Plugin())
```

That's it. All calls through this Client emit OTEL metrics automatically.

## Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `resilience.call.total` | Counter | Every fn call (including retries) |
| `resilience.call.duration_seconds` | Histogram | Duration of each fn call |
| `resilience.call.errors` | Counter | fn calls that returned an error |
| `resilience.retry.total` | Counter | Retry decisions (label: `option`) |
| `resilience.retry.wait_seconds` | Histogram | Backoff wait duration (label: `option`) |

The `option` label carries the retry option name (e.g. `"retry"`, `"service"`, `"custom"`). This enables per-category dashboards in Grafana.

## How it works

The Plugin implements `resilience.Plugin` — it returns `Events` hooks:

- **OnAfterCall** — increments `call.total`, records `call.duration_seconds`, increments `call.errors` on failure
- **OnBeforeWait** — increments `retry.total`, records `retry.wait_seconds` with the option name label

No `OnBeforeCall` — the Plugin doesn't need it. Events are additive — multiple Plugins can coexist.

## No OTEL SDK? No overhead.

If no OTEL SDK is configured, the default no-op meter produces no-op instruments. Zero allocations, zero overhead. The Plugin is always safe to register.

## Design

OTEL is a Plugin, not an Option. It observes — doesn't control execution. Shared across all calls via Client. This is the canonical example of when to use Plugin vs Option.

Want to go deeper? [Options & Plugins](/guide/options-plugins) explains the two extension points and how to write your own.
