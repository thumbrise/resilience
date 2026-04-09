---
title: "RFC: Core Redesign — Generic Events, Fallback, Composable Extensibility"
description: "Full architectural RFC for resilience v2 core: generic event model, fallback sub-package, listener-based observability. Includes adversarial review transcript, NIH research, and decision log."
head:
  - - meta
    - name: keywords
      content: resilience rfc, go composable middleware, generic events go, fallback chain go, adversarial architecture review, resilience4j go alternative
---

# RFC: Core Redesign — Generic Events, Fallback, Composable Extensibility

> "The best way to test an architecture is to let someone try to destroy it."
> — [Devlog #7](../devlog/007-adversarial-architecture-review.md)

## Meta

| Field | Value |
|-------|-------|
| Authors | Ruslan Kokoev (arbiter), Devin (analyst), DeepSeek (adversarial reviewer) |
| Date | 2026-04-09 |
| Status | Draft |
| Scope | `resilience` core, `fallback` sub-package |
| Preceded by | [Devlog #7 — The architecture fight](../devlog/007-adversarial-architecture-review.md) |
| Repositories | [thumbrise/resilience](https://github.com/thumbrise/resilience) |

## Abstract

This RFC proposes a redesign of the `resilience` core to replace the fixed `Events` struct and `Plugin` interface with a generic event model based on `Emit(ctx, any)` and `Listener func(ctx, any)`. The redesign enables infinite extensibility: community sub-packages define their own typed events, listeners consume them via type switch, and the core never changes when new patterns are added.

The RFC also specifies the `fallback` sub-package — the first new pattern to use the redesigned core.

Every decision in this document was stress-tested through a multi-hour adversarial review session. The process, the debates, and the rejected alternatives are documented in full.

---

## Table of Contents

[[toc]]

---

## 1. Motivation

### 1.1 The trigger

The `resilience` library was designed as a composable toolkit for Go function calls. The core primitive — `type Option func(ctx context.Context, call func(context.Context) error) error` — has proven correct. It covers retry, timeout, and any future pattern as a simple function that wraps a call.

But when we attempted to design the `fallback` sub-package, we hit three walls:

1. **Events don't scale.** The `Events` struct has three hardcoded fields: `OnBeforeCall`, `OnAfterCall`, `OnBeforeWait`. Adding `OnFallbackExecuted` or `OnCircuitOpen` requires changing the core. Every new pattern = core change. This violates zero coupling.

2. **Plugin/Option boundary is unclear.** The documentation says "Option wraps, Plugin observes." But circuit breaker needs both: shared state (Plugin) and execution control (Option). The boundary breaks on the first real-world pattern that crosses it.

3. **No path for stream-based patterns.** Debounce, throttle, dedup — these require shared mutable state across calls. The current `Events` struct has no mechanism for sub-packages to store or retrieve state from the pipeline context.

These are not theoretical problems. They block concrete work: `fallback` for provider failover, `circuit` for production use, `debounce` for event-driven systems.

### 1.2 Current architecture

The current core (`resilience.go`) defines five types:

```go
// The universal primitive. Every pattern is this function.
type Option func(ctx context.Context, call func(context.Context) error) error

// Lifecycle observer. Lives on Client. Shared state across calls.
type Plugin interface {
    Name() string
    Events() Events
}

// Fixed set of lifecycle hooks. Three fields, hardcoded.
type Events struct {
    OnBeforeCall  func(ctx context.Context, attempt int)
    OnAfterCall   func(ctx context.Context, attempt int, err error, duration time.Duration)
    OnBeforeWait  func(ctx context.Context, option string, attempt int, wait time.Duration)
}

// Application-wide instance. Holds plugins. Immutable after creation.
type Client struct {
    plugins []Plugin
    events  []Events
}

// Per-call configuration. Holds options. Fresh each Do().
type CallBuilder struct {
    fn      func(context.Context) error
    client  *Client
    options []Option
    plugins []Plugin
}
```

The pipeline works as middleware: `Do()` wraps `fn` with options (last option innermost), attaches plugin events to context, and executes. Sub-packages like `retry` extract events from context via `EventsFromContext(ctx)` and emit lifecycle notifications via `EmitBeforeCall`, `EmitAfterCall`, `EmitBeforeWait`.

The OTEL plugin (`otel/otel.go`) implements `Plugin` and returns an `Events` struct with two callbacks: `OnAfterCall` (metrics) and `OnBeforeWait` (retry metrics). It cannot observe events from patterns that don't exist yet (fallback, circuit, etc.) because `Events` has no fields for them.

### 1.3 The bottleneck

The `Events` struct is the bottleneck. It is the single point where the core must know about every possible lifecycle event in advance. This makes the core a gatekeeper instead of an enabler.

Current state:

```
retry sub-package → emits OnBeforeCall, OnAfterCall, OnBeforeWait
                     ↓
              Events struct (3 fields, hardcoded in core)
                     ↓
              OTEL plugin (reads 2 of 3 fields)
```

What happens when we add fallback:

```
fallback sub-package → wants to emit OnFallbackExecuted, OnChainExhausted
                        ↓
                 Events struct ← MUST ADD NEW FIELDS (core change!)
                        ↓
                 OTEL plugin ← MUST ADD NEW CALLBACKS (plugin change!)
```

Every new pattern forces changes in two places: the core `Events` struct and every existing plugin. This is the opposite of zero coupling.

---

## 2. Research Process

### 2.1 How we got here

The research started as a practical question: how to implement `fallback` as a sub-package. Fallback is a resilience pattern — try a primary call, and if it fails, try alternatives in order. But implementing it exposed the `Events` scalability problem, which led to a full core redesign investigation.

### 2.2 Adversarial review format

The review followed the format established in [Devlog #7](../devlog/007-adversarial-architecture-review.md):

- **Arbiter** (Ruslan) — human with final authority. Challenged both sides. Sent disputed claims for verification.
- **Analyst** (Devin) — analyzed current codebase, identified gaps, proposed solutions, wrote prompts for the adversarial reviewer.
- **Adversarial Reviewer** (DeepSeek) — received structured prompts with full context. Attacked the current architecture, proposed alternatives, defended positions under cross-examination.

The review ran for 2+ hours across multiple rounds. Key debates are documented below.

### 2.3 Prompts and methodology

Four structured prompts were sent to DeepSeek:

**Prompt 1: Reactive programming patterns.** Compared `resilience`'s `Option` primitive with ReactiveX operators (RxJS, RxGo), Polly (.NET), resilience4j (Java), failsafe-go. Asked for concrete Go pseudocode for fallback variants (To, Chain, OnFunc, Race).

**Prompt 2: Error aggregation, events, hedge vs race, conditional chain.** Deep dive into `errors.Join` vs custom error types, event model extensibility, whether `Race` belongs in fallback or hedge, and conditional fallback with per-link error classification.

**Prompt 3: Adversarial review of the core.** Full attack on `type Option func(ctx, call) error`. Find every pattern that cannot be implemented. Compare with failsafe-go, Polly v8, resilience4j. Propose minimal core changes. Self-review the proposal.

**Prompt 4: Collision = ignorance principle.** Develop the insight that typed context keys prevent collision through Go's import graph, and apply this principle to the event model.

Each prompt included the full current source code of `resilience.go`, `retry/retry.go`, and `otel/otel.go` as context.

---

## 3. Key Debates

### 3.1 Option vs Policy — the central fight

**DeepSeek's opening position:** Replace `type Option func(ctx, call) error` with `interface Policy { Execute(ctx, call) error }`. Argument: Policy is a "real type" with identity, introspection, lifecycle methods. Option is "just a function" — no way to inspect it, no way to manage its lifecycle, no way to add methods later.

**Arbiter's challenge:**

> What is the difference between:
> ```go
> func retry(...) resilience.Option {
>     myRetrier := myCoreRetrier{...}
>     return func(ctx context.Context, call func(context.Context) error) error {
>         // stateful logic with shared state via closure
>     }
> }
> ```
> and:
> ```go
> func retry(...) resilience.Policy {
>     return myPolicyImplementor{...}
> }
> ```

DeepSeek initially argued that Policy enables introspection, lifecycle management, and type-safe composition. The arbiter systematically dismantled each argument:

1. **Introspection.** "Who holds the reference to the Policy? The core? Then the core must do mass type assertion to discover capabilities. That violates OCP and zero coupling." DeepSeek conceded.

2. **Lifecycle.** "A closure can capture a struct with `Close()`. The constructor returns both the Option and a cleanup function. No interface needed." DeepSeek conceded.

3. **Composition.** "Options compose via `With(a, b, c)`. The middleware chain is built at `Do()` time. Adding methods to Policy doesn't help — the chain is the composition mechanism." DeepSeek conceded.

4. **Forward compatibility.** DeepSeek's strongest argument: "With an interface, you can add optional methods via type assertion. With a function, you can't." The arbiter countered: "We can change the function signature: `func(ctx, core, call) error` where `core` provides new capabilities." Then immediately rejected his own proposal: "Context already is core. `context.WithValue` + typed keys = the same mechanism, already idiomatic in Go."

**DeepSeek's final position (after 2 hours):** "If you're ready to maintain a global state registry, test through context mocks, and give up admin handles — stay with functions. If you want a foundation for years — interfaces."

**Arbiter's ruling:** Stay with functions. The arguments for `interface Policy` reduce to convenience for library authors, not capability. Every capability DeepSeek demonstrated with Policy can be achieved with closures + context. The function signature is simpler, more composable, and idiomatic Go.

**Decision D1: `type Option func(ctx context.Context, call func(context.Context) error) error` is confirmed as the correct and only primitive. No `interface Policy`.**

### 3.2 Core object — proposed and rejected

During the Policy debate, an intermediate proposal emerged: add a `Core` object as a second parameter.

```go
type Option func(ctx context.Context, core *Core, call func(context.Context) error) error
```

`Core` would provide helpers: `core.Emit(event)`, `core.Get(key)`, `core.Set(key, val)`.

**Rejected because:** `context.Context` already serves this purpose. Go's standard library uses context as the carrier for request-scoped data. Adding a parallel mechanism (`Core`) creates two ways to pass data through the pipeline. The Go ecosystem (gRPC metadata, HTTP middleware, tracing) has standardized on context. Fighting this convention gains nothing.

**Decision D2: No `Core` parameter. Context is the core. Helpers operate on context: `resilience.Emit(ctx, event)`.**

### 3.3 Mass type assertion in the core

DeepSeek proposed that the core could discover optional capabilities of policies via type assertion:

```go
if inspectable, ok := policy.(Inspectable); ok {
    info := inspectable.Inspect()
    // ...
}
```

**Arbiter's challenge:** "What does mass type assertion in the core violate?"

Answer: **Open/Closed Principle.** Every new optional interface requires the core to add a new type assertion branch. The core must know about every possible capability in advance — the exact same problem as the fixed `Events` struct. The core becomes a registry of known interfaces instead of a generic pipeline.

It also violates **zero coupling between patterns.** If the core type-asserts for `CircuitBreakerPolicy`, it knows about circuit breakers. If it type-asserts for `FallbackPolicy`, it knows about fallback. The core accumulates knowledge about every pattern — the opposite of "each pattern is an independent sub-package."

**Decision D3: No type assertion in the core. The core is generic. It does not know about specific patterns.**

### 3.4 The collision principle

This is the key architectural insight that emerged from the debate.

**The Go linter rule:** "Do not use string literals as context keys." Why? Because two packages might accidentally use the same string. This is a **collision** — two parties that don't know about each other interfere.

**The solution:** Typed keys. `type myKey struct{}` is unique to the package that defines it. Two packages cannot accidentally create the same type. The Go compiler enforces uniqueness through the import graph.

**The insight applied to events:**

A **collision** is when two packages accidentally interfere — they don't know about each other. This is bad. The linter prevents it.

A **dependency** is when one package intentionally imports another's type. This is good. It's explicit, visible in `go.mod`, checked by the compiler.

In the event model:
- The **core** emits events as `any`. It doesn't know what types flow through it. No collision possible — the core doesn't participate in the type system of events.
- A **sub-package** (e.g., `fallback`) defines its own event types: `fallback.Exhausted{}`, `fallback.Switched{}`. These types are unique to the `fallback` package.
- A **listener** imports `fallback` and does `case fallback.Exhausted:`. This is an **explicit, intentional dependency**. The listener knows about fallback because it chose to import it.

No collision is possible because:
1. Event types are Go types, not strings
2. Go types are unique per package
3. Listeners explicitly import the packages whose events they care about
4. The compiler verifies the import graph

**Decision D4: "Collision = ignorance, dependency = knowledge." The event model uses typed events. The core is generic (`any`). Listeners are explicit (type switch + import). The compiler enforces correctness.**

---

## 4. Proposed Architecture

### 4.1 New core types

```go
// Option — unchanged. The universal primitive.
type Option func(ctx context.Context, call func(context.Context) error) error

// Listener replaces Plugin. A function that receives typed events.
// Listeners observe — they don't control execution.
// Each sub-package defines its own event types.
// Listeners do type switch to handle events they care about.
type Listener func(ctx context.Context, event any)

// Client holds listeners. Immutable after creation.
type Client struct {
    listeners []Listener
}

// NewClient creates a Client with the given listeners.
func NewClient(listeners ...Listener) *Client

// CallBuilder — same API, but uses listeners instead of plugins.
type CallBuilder struct {
    fn        func(context.Context) error
    client    *Client
    options   []Option
    listeners []Listener
}
```

### 4.2 New event model

**Core provides one helper:**

```go
// Emit sends an event to all listeners attached to the context.
// Sub-packages call this to notify listeners about lifecycle events.
// The core does not inspect, filter, or transform the event.
func Emit(ctx context.Context, event any) {
    listeners, _ := ctx.Value(listenersKey{}).([]Listener)
    for _, ln := range listeners {
        ln(ctx, event)
    }
}
```

**Sub-packages define their own event types:**

```go
// In package retry:
type Attempted struct {
    Attempt  int
    Err      error
    Duration time.Duration
    Wait     time.Duration
}

// In package fallback:
type Switched struct {
    From int  // index of failed provider
    To   int  // index of next provider
    Err  error
}

type Exhausted struct {
    Errors []error
}
```

**Listeners consume events via type switch:**

```go
// OTEL listener — lives in otel/ sub-package or user code
func OTELListener() resilience.Listener {
    return func(ctx context.Context, event any) {
        switch e := event.(type) {
        case retry.Attempted:
            retryTotal.Add(ctx, 1)
            retryWait.Record(ctx, e.Wait.Seconds())
        case fallback.Switched:
            fallbackSwitchTotal.Add(ctx, 1)
        case fallback.Exhausted:
            fallbackExhaustedTotal.Add(ctx, 1)
        }
    }
}

// User's custom listener — no library changes needed
func MySlackListener() resilience.Listener {
    return func(ctx context.Context, event any) {
        switch e := event.(type) {
        case fallback.Exhausted:
            slack.Send("All providers failed: %v", e.Errors)
        }
    }
}
```

### 4.3 What changes in the pipeline

**Before (current):**

```go
func (b *CallBuilder) Do(ctx context.Context) error {
    events := b.mergeEvents()           // collect Events structs from plugins
    if len(events) > 0 {
        ctx = withEvents(ctx, events)    // attach to context
    }
    // ... build middleware chain, execute
}
```

Sub-packages:
```go
events := resilience.EventsFromContext(ctx)
resilience.EmitBeforeCall(ctx, events, attempt)
// ... do work ...
resilience.EmitAfterCall(ctx, events, attempt, err, duration)
```

**After (proposed):**

```go
func (b *CallBuilder) Do(ctx context.Context) error {
    listeners := b.mergeListeners()       // collect Listener funcs
    if len(listeners) > 0 {
        ctx = withListeners(ctx, listeners) // attach to context
    }
    // ... build middleware chain, execute
}
```

Sub-packages:
```go
resilience.Emit(ctx, retry.Attempted{
    Attempt:  attempt,
    Err:      err,
    Duration: duration,
    Wait:     wait,
})
```

Three typed emitter functions (`EmitBeforeCall`, `EmitAfterCall`, `EmitBeforeWait`) collapse into one generic function (`Emit`). Three `Events` struct fields collapse into one `Listener` function type. The core shrinks. Sub-packages gain full autonomy over their event types.

### 4.4 Context carries mutable state

The context itself is immutable — `context.WithValue` creates a new context. But the **values stored in context** can be mutable. This is already used in the current codebase: `withEvents` stores a `[]Events` slice in context.

In the new model, sub-packages that need shared mutable state (circuit breaker, rate limiter, debounce) can store pointers to mutable structs in context:

```go
// Circuit breaker stores its state in context via the Client
type State struct {
    mu       sync.Mutex
    failures int
    open     bool
}

// The Option closure captures the state pointer
func Breaker(threshold int) resilience.Option {
    state := &State{}
    return func(ctx context.Context, call func(context.Context) error) error {
        state.mu.Lock()
        if state.open {
            state.mu.Unlock()
            return ErrCircuitOpen
        }
        state.mu.Unlock()
        
        err := call(ctx)
        
        state.mu.Lock()
        defer state.mu.Unlock()
        if err != nil {
            state.failures++
            if state.failures >= threshold {
                state.open = true
                resilience.Emit(ctx, Opened{After: state.failures})
            }
        } else {
            state.failures = 0
        }
        return err
    }
}
```

The state lives in the closure, not in the context. The context carries listeners for observability. The Option controls execution. Clean separation.

---

## 5. Fallback Sub-Package

### 5.1 API

```go
package fallback

import "github.com/thumbrise/resilience"

// ErrChainExhausted is a sentinel error indicating all alternatives failed.
var ErrChainExhausted = errors.New("fallback chain exhausted")

// To provides a single fallback alternative.
// If call fails, backupFn is executed.
func To(backupFn func(context.Context) error) resilience.Option

// Chain tries alternatives in order until one succeeds.
// If all fail, returns errors.Join(ErrChainExhausted, err1, err2, ...).
func Chain(backupFns ...func(context.Context) error) resilience.Option

// ToOnFunc provides a conditional single fallback.
// Fallback executes only when classify(err) returns true.
func ToOnFunc(classify func(error) bool, backupFn func(context.Context) error) resilience.Option

// ChainOnFunc provides a conditional ordered chain.
// Fallback to next alternative only when classify(err) returns true.
// Non-matching errors are returned immediately without trying alternatives.
func ChainOnFunc(classify func(error) bool, backupFns ...func(context.Context) error) resilience.Option
```

### 5.2 Error aggregation

When all alternatives in a `Chain` fail, the error is:

```go
errors.Join(ErrChainExhausted, err1, err2, err3)
```

This preserves all individual errors while providing a sentinel for matching:

```go
// Retry the entire chain on exhaustion
resilience.Do(ctx, primaryCall,
    fallback.Chain(secondary, tertiary),
    retry.On(fallback.ErrChainExhausted, 3, backoff.Exponential(...)),
)
```

`errors.Is(joinedErr, ErrChainExhausted)` returns `true` because `errors.Join` supports `Is`/`As` unwrapping. `errors.Is(joinedErr, someProviderErr)` also returns `true` — individual errors are preserved.

**Important interaction with retry:** If retry wraps fallback, and fallback returns `errors.Join(ErrChainExhausted, ErrRateLimit, ErrTimeout)`, then `retry.On(ErrRateLimit, ...)` will match on the joined error. This may cause false positives — retry triggers on a rate limit error from a fallback provider, not the primary.

**Mitigation:** Use `retry.On(fallback.ErrChainExhausted, ...)` to retry the entire chain, not individual provider errors. This is the recommended pattern. Document it prominently.

### 5.3 Events

```go
package fallback

// Switched is emitted when the primary call fails and a fallback is attempted.
type Switched struct {
    From int   // index of the failed function (0 = primary)
    To   int   // index of the next function to try
    Err  error // error from the failed function
}

// Exhausted is emitted when all alternatives in a Chain have failed.
type Exhausted struct {
    Errors []error // all errors, in order
}
```

### 5.4 No Race

`Race` (parallel execution, first success wins) is not included in `fallback`. It belongs in `hedge` — speculative parallel calls are a different pattern with different semantics (cancellation of losers, resource management). The roadmap already lists `hedge.After(100*ms)` as a separate sub-package.

---

## 6. Event system redesign — detailed specification

### 6.1 Current state (to be replaced)

```go
// resilience.go — current
type Events struct {
    OnBeforeCall func(ctx context.Context, attempt int)
    OnAfterCall  func(ctx context.Context, attempt int, err error, duration time.Duration)
    OnBeforeWait func(ctx context.Context, option string, attempt int, wait time.Duration)
}

type Plugin interface {
    Name() string
    Events() Events
}
```

Problems:
1. Adding a new event field to `Events` requires changing the core
2. Sub-packages cannot define their own events
3. `Plugin` interface couples observability to a fixed set of lifecycle hooks
4. Community cannot extend the event system without PRs to the core

### 6.2 New design

#### Core types

```go
// resilience.go — new

// Listener receives events emitted by options during execution.
// Events are typed — each sub-package defines its own event types.
// Listeners use type switch to handle events they care about.
//
// Listener is the replacement for Plugin. It is a function, not an interface,
// consistent with Option being a function.
type Listener func(ctx context.Context, event any)
```

#### Core helpers

```go
// Emit sends an event to all listeners attached to the context.
// Sub-packages call this to notify listeners about lifecycle events.
//
// Usage from sub-packages:
//
//     resilience.Emit(ctx, retry.Attempted{Attempt: 3, Wait: 5*time.Second})
//     resilience.Emit(ctx, fallback.Switched{From: 0, To: 1})
//
func Emit(ctx context.Context, event any) {
    listeners, _ := ctx.Value(listenersKey{}).([]Listener)
    for _, l := range listeners {
        l(ctx, event)
    }
}
```

#### Client changes

```go
type Client struct {
    listeners []Listener
}

func NewClient(listeners ...Listener) *Client {
    return &Client{listeners: listeners}
}
```

`Plugin` interface is removed. `Events` struct is removed. `EmitBeforeCall`, `EmitAfterCall`, `EmitBeforeWait` helpers are removed. All replaced by single `Emit(ctx, any)`.

#### Sub-package event types

Each sub-package defines its own event types as exported structs:

```go
// retry/events.go
package retry

type Attempted struct {
    Attempt  int
    Err      error
    Duration time.Duration
}

type WaitStarted struct {
    Attempt int
    Wait    time.Duration
}

type Exhausted struct {
    Attempts int
    LastErr  error
}
```

```go
// fallback/events.go
package fallback

type Switched struct {
    From int
    To   int
    Err  error
}

type Exhausted struct {
    Errors []error
}
```

#### OTEL listener (example)

```go
// otel/listener.go
package rsotel

func Listener() resilience.Listener {
    // init metrics...
    return func(ctx context.Context, event any) {
        switch e := event.(type) {
        case retry.Attempted:
            callTotal.Add(ctx, 1)
            callDuration.Record(ctx, e.Duration.Seconds())
            if e.Err != nil {
                callErrors.Add(ctx, 1)
            }
        case retry.WaitStarted:
            retryWait.Record(ctx, e.Wait.Seconds())
        case fallback.Switched:
            fallbackSwitches.Add(ctx, 1)
        }
    }
}
```

Community can write their own listeners without touching the core. A Telegram notification listener, a custom metrics listener, a debug logger — all are just `func(ctx, any)`.

### 6.3 The collision/knowledge principle

This design is grounded in a fundamental architectural principle discovered during adversarial review:

**Collision = ignorance. Dependency = knowledge.**

The Go linter rule "don't use string literals as context keys" exists because string keys can collide — two packages accidentally use the same string. Typed keys (`type myKey struct{}`) solve this through Go's import graph: the compiler verifies uniqueness.

The same principle applies to events:

- **Collision (bad):** Two packages accidentally emit events with the same string type `"retry"`. The listener cannot distinguish them. This is ignorance — packages don't know about each other.
- **Knowledge (good):** A listener imports `retry.Attempted` and `fallback.Switched` as typed structs. This is an explicit, conscious dependency visible in `go.mod` and verified by the compiler. The listener **knows** about these packages because it **chose** to know.

The core (`resilience.Emit(ctx, any)`) knows nothing about specific events. It is the enabler. Sub-packages define typed events. Listeners import the types they care about. The compiler enforces correctness. No runtime surprises. No collisions.

This is not "just another way to do events." This is transferring dependency verification from runtime to compile time.

### 6.4 What we lose (honest trade-offs)

1. **Type safety on emit.** `Emit(ctx, any)` accepts anything. A sub-package could accidentally emit a string or int. Mitigation: convention + code review. The core cannot enforce "only emit structs" at compile time.

2. **Discoverability.** With `Events` struct, IDE autocomplete shows all available events. With `any`, the user must read sub-package docs to know what events exist. Mitigation: each sub-package documents its events in `events.go`.

3. **No guaranteed event contract.** The core cannot enforce that a sub-package emits specific events. A retry implementation might forget to emit `retry.Attempted`. Mitigation: tests within each sub-package.

These trade-offs are acceptable. The alternative (fixed `Events` struct in core) has a worse trade-off: the core becomes a bottleneck for every new pattern.

---

## 7. Fallback sub-package — detailed specification

### 7.1 Location

`fallback/` sub-package in `thumbrise/resilience`, alongside `retry/` and `backoff/`.

### 7.2 Public API

```go
package fallback

// ErrChainExhausted is a sentinel error indicating all alternatives failed.
var ErrChainExhausted = errors.New("fallback chain exhausted")

// To returns an Option that calls backupFn when the primary call fails.
// Any error from the primary triggers the fallback.
func To(backupFn func(context.Context) error) resilience.Option

// Chain returns an Option that tries alternatives in order.
// If all fail, returns errors.Join(ErrChainExhausted, err1, err2, ...).
func Chain(backupFns ...func(context.Context) error) resilience.Option

// ToOnFunc returns an Option that calls backupFn only when classify(err) returns true.
// Non-matching errors pass through unchanged.
func ToOnFunc(classify func(error) bool, backupFn func(context.Context) error) resilience.Option

// ChainOnFunc returns an Option that tries alternatives in order,
// but only when classify(err) returns true for the failing call's error.
func ChainOnFunc(classify func(error) bool, backupFns ...func(context.Context) error) resilience.Option
```

### 7.3 Error aggregation

When all alternatives in a chain fail:

```go
return errors.Join(ErrChainExhausted, err1, err2, err3)
```

This allows:
- `errors.Is(err, ErrChainExhausted)` — check if chain exhausted
- `errors.Is(err, someSpecificErr)` — check if any alternative returned a specific error
- Retry can match on `ErrChainExhausted` specifically, avoiding false positives on individual provider errors

### 7.4 Events

```go
// fallback/events.go

// Switched is emitted when the primary call fails and a fallback is attempted.
type Switched struct {
    From int   // index of the failed call (0 = primary)
    To   int   // index of the next alternative
    Err  error // error from the failed call
}

// Exhausted is emitted when all alternatives in a chain have failed.
type Exhausted struct {
    Errors []error // all errors, in order
}
```

### 7.5 Composition with retry

```go
resilience.Do(ctx, primaryCall,
    fallback.Chain(secondary, tertiary),
    retry.On(fallback.ErrChainExhausted, 3, backoff.Exponential(1*time.Second, 30*time.Second)),
)
```

Retry wraps fallback. If the entire chain (primary → secondary → tertiary) fails, retry sees `ErrChainExhausted` and retries the whole chain. Retry does not retry individual alternatives — that is fallback's job.

### 7.6 No Race

`Race` (parallel execution, first success wins) belongs in `hedge/` sub-package, not in `fallback/`. Fallback is sequential by definition. Hedge is parallel by definition. Different sub-packages, different semantics.

---

## 8. Implementation order

### Phase 1: resilience — fallback sub-package (no event system changes)

1. Create `fallback/` sub-package with `To`, `Chain`, `ToOnFunc`, `ChainOnFunc`
2. Use current `Events` system (no changes to core yet)
3. Add `ErrChainExhausted` sentinel
4. Tests: unit tests for each function, composition tests with retry
5. Documentation in VitePress

This is unblocked. Can ship independently.

### Phase 2: resilience — event system redesign

1. Add `Listener` type and `Emit` helper to core
2. Add `listenersKey` context key and `withListeners` helper
3. Update `Client` to accept `Listener` instead of `Plugin`
4. Update `CallBuilder.Do` to attach listeners to context
5. Define event types in `retry/events.go`
6. Update `retry` to emit typed events via `resilience.Emit`
7. Rewrite `otel/` as a `Listener` instead of `Plugin`
8. Deprecate `Plugin`, `Events`, `EmitBeforeCall`, `EmitAfterCall`, `EmitBeforeWait`
9. Remove deprecated types in next major version

This is the largest change. It touches the core. It must be done carefully with backward compatibility period.

### Phase 3: resilience — fallback events

1. Define event types in `fallback/events.go` (`Switched`, `Exhausted`)
2. Update fallback functions to emit events via `resilience.Emit`
3. Update OTEL listener to handle fallback events

Depends on Phase 2.

---

## 9. Decisions log

This section records every decision made during the adversarial review process, including rejected alternatives.

### D-01: Option remains `func(ctx, call) error`

**Decision:** Keep the current function type. Do not replace with `interface Policy`.

**Context:** DeepSeek proposed replacing `Option` with `interface Policy { Execute(ctx, call) error }` for better introspection, lifecycle management, and testability.

**Adversarial review (2 hours):**
- Argument "users don't know about hidden state" — rejected. Users don't care about internals, they care about DSL.
- Argument "lifecycle management (Close/Reset)" — rejected. Stateful components (circuit breaker, rate limiter) are separate academic structs. The composable function from the extension package wraps them. Separation of concerns.
- Argument "introspection via optional interfaces" — rejected. Mass type assertion in the core violates OCP and zero coupling.
- Argument "forward compatibility" — partially valid, but `func(ctx, core, call) error` with a Core object achieves the same. And context already IS the core.

**Conclusion:** `func(ctx, call) error` with captured state in closures is sufficient. Interface adds complexity without proportional benefit. The function type is the universal extension point.

### D-02: Plugin replaced by Listener

**Decision:** Replace `interface Plugin { Name() string; Events() Events }` with `type Listener func(ctx context.Context, event any)`.

**Rationale:**
- Plugin couples observability to a fixed set of lifecycle hooks (3 fields in Events struct)
- Adding new events requires changing the core
- Listener is a function, consistent with Option being a function
- Community can write listeners without PRs to the core

### D-03: Events struct replaced by typed events + Emit

**Decision:** Replace fixed `Events` struct with generic `Emit(ctx, any)` and typed event structs per sub-package.

**Rationale:** The collision/knowledge principle. Typed events use Go's import graph for compile-time verification. The core knows nothing about specific events. Sub-packages define their own. Listeners import what they need.

**Trade-offs accepted:** Loss of type safety on emit, loss of IDE discoverability. Mitigated by convention and documentation.

### D-04: Context is the Core

**Decision:** Do not introduce a separate `Core` object. Use `context.Context` as the carrier for pipeline metadata (listeners, future shared state).

**Context:** During adversarial review, the idea of `func(ctx, core, call) error` was proposed and then rejected. Context already carries events via `withEvents`/`EventsFromContext`. This is idiomatic Go (same pattern as gRPC metadata, HTTP middleware). Adding a separate Core object duplicates what context already does.

### D-05: Fallback error aggregation via errors.Join

**Decision:** `errors.Join(ErrChainExhausted, err1, err2, ...)` instead of custom `ChainError` type.

**Rationale:** Standard library. Works with `errors.Is`/`errors.As`. Sentinel `ErrChainExhausted` allows retry to match specifically on chain exhaustion without false positives on individual provider errors.

### D-06: No Race in fallback

**Decision:** `Race` (parallel execution) belongs in `hedge/` sub-package, not `fallback/`.

**Rationale:** Fallback is sequential. Hedge is parallel. Different semantics, different sub-packages. Mixing them creates confusion.

### D-07: OTEL temporarily deprioritized

**Decision:** Event system redesign (Phase 2) comes after fallback (Phase 1). OTEL listener rewrite is part of Phase 2.

**Rationale:** Composable API design is the priority. Observability is important but should not constrain API design. Current OTEL plugin continues to work until Phase 2.

---

## 10. Open questions

### Q-01: Debounce and stream-based patterns

Can `func(ctx, call) error` express debounce, throttle, batch, dedup? These patterns require shared state across multiple `Do()` invocations and time-based triggers. Initial analysis suggests closures with captured state can handle this, but no concrete implementation has been validated.

**Status:** Requires further research. Separate RFC.

### Q-02: Events extensibility — discoverability

With generic `Emit(ctx, any)`, how does a user discover what events a sub-package emits? Convention (document in `events.go`) is the current answer. Is there a better mechanism that preserves type safety?

**Status:** Open. May be addressed by tooling (linter that checks event documentation).

### Q-03: Backward compatibility period for Plugin → Listener migration

How long should deprecated `Plugin`/`Events` types remain in the codebase? One minor version? One major version?

**Status:** To be decided during Phase 2 implementation.

---

## 11. References

### Repositories

- [thumbrise/resilience](https://github.com/thumbrise/resilience) — composable resilience for Go
- [thumbrise/multimod](https://github.com/thumbrise/multimod) — multi-module CLI tooling

### External libraries analyzed

- **failsafe-go** — Policy-based resilience for Go. Closest competitor. Uses interface-based design.
- **resilience4j** (Java) — Functional resilience library. Event system with typed events.
- **Polly v8** (.NET) — Resilience framework. `ShouldHandle` predicate for conditional policies.
- **RxGo** — ReactiveX for Go. Composable operators, error handling patterns.
- **RxJS** — ReactiveX for JavaScript. `catchError`, `retry`, `timeout` operators.

### Adversarial review sessions

- **Session 1:** DeepSeek — initial analysis of Rx patterns, fallback design, error aggregation
- **Session 2:** DeepSeek — Option vs Policy debate (2 hours). Conclusion: Option wins.
- **Session 3:** DeepSeek — Event system extensibility, collision/knowledge principle
- **Arbiter:** Ruslan Kokoev — challenged every argument, rejected weak reasoning, drove to correct conclusions

### Key documents

- [Adversarial Architecture Review (devlog 007)](https://thumbrise.github.io/resilience/devlog/007-adversarial-architecture-review) — "Adversarial review works — but only with an arbiter."
- [resilience roadmap](https://thumbrise.github.io/resilience/guide/roadmap) — current state and planned patterns
