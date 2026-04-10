---
title: "resilience Devlog #9 — The stress debate"
description: "Two hours arguing with DeepSeek about whether our core primitive should exist. Option vs Policy, mass type assertion, context as core, and the collision insight that changed our event system."
head:
  - - meta
    - name: keywords
      content: adversarial review, go resilience design, option vs policy, composable api, event system, context as core, stress debate
---

# #9 — The stress debate

> "A function that survived two hours of attack is stronger than an interface that was never questioned."

## The trigger

April 10, 2026. We had a working core — `func(ctx context.Context, call func(context.Context) error) error`. One primitive. Retry worked. OTEL worked. Backoff worked. The roadmap listed fallback, circuit breaker, rate limiter, hedge, debounce. Everything looked clean on paper.

Then we asked DeepSeek to destroy it.

Not "review it." Not "suggest improvements." Destroy it. Find every pattern that can't be expressed. Find every library that does it better. Find every architectural dead end. And if the core can't survive — propose a replacement.

DeepSeek came back with a verdict: **replace `Option` with `interface Policy`.**

The argument was coherent. The examples were real. The comparison table was damning. And for about fifteen minutes, it felt like our entire architecture was wrong.

It wasn't. But proving that took two hours.

## Round 1: "Your primitive is stateless"

DeepSeek's opening move: `type Option func(...) error` is by definition stateless. Circuit breaker needs shared state. Rate limiter needs shared state. Bulkhead needs shared state. Your primitive can't express them. `interface Policy` with an `Execute` method can hold state in struct fields. Game over.

The arbiter's response was one code block:

```go
func retry(...) resilience.Option {
    myRetrier := myCoreRetrier{...}  // ← state lives here
    return func(ctx context.Context, call func(context.Context) error) error {
        // stateful logic with shared state via closure
    }
}
```

A closure captures state. The constructor creates it. The returned function uses it. The user sees `resilience.Option` — same type, same DSL, same `With()`. The state is invisible to the caller and fully controlled by the constructor.

DeepSeek pivoted: "Fine, it works for simple cases. But closures break on composition, observability, and resource management."

The debate had begun.

## Round 2: "Closures break on composition"

DeepSeek's scenario: retry wraps circuit breaker. Retry needs to check if the circuit is open before retrying. With `interface Policy`, retry holds a reference to the circuit breaker's interface and calls `cb.IsOpen()`. Clean, typed, explicit.

The arbiter asked one question: **who holds the reference?**

Our library is designed so that each component is independent. `retry/` doesn't know about `circuitbreaker/`. `circuitbreaker/` doesn't know about `retry/`. Zero coupling between patterns. That's not a nice-to-have — it's the architectural invariant that makes community contributions possible.

If retry holds a first-class reference to circuit breaker, that invariant is dead. Whether it's a function or an interface doesn't matter — the coupling is in the reference, not the type system.

DeepSeek proposed a runtime registry. The arbiter rejected it: global mutable state, type assertions, implicit dependencies. Everything we're trying to avoid.

**Score: Option 1, Policy 0.** The composition argument required coupling that neither approach should have.

## Round 3: "Interfaces enable introspection"

DeepSeek's strongest card: optional interfaces via type assertion.

```go
type Stateful interface {
    State() any
}

// In the core runtime:
for _, policy := range policies {
    if s, ok := policy.(Stateful); ok {
        metrics.Record(s.State())
    }
}
```

Add new capabilities without breaking changes. Forward compatibility. The arbiter almost agreed.

Then the arbiter asked: **what does mass type assertion in the core violate?**

Silence. Then DeepSeek worked it out:

1. **Open-Closed Principle** — every new optional interface requires a new type assertion in the core. The core changes every time a community contributor invents a new capability.
2. **Zero coupling** — the core now knows about `Stateful`, `Closeable`, `Resettable`, `Observable`. Each is a dependency on a concept that belongs to a sub-package, not the core.
3. **Hidden contracts** — the compiler doesn't enforce which policies implement which optional interfaces. It's documentation, not architecture.

The very feature that made `interface Policy` attractive — optional methods via type assertion — was the feature that would kill the core's extensibility.

**Score: Option 2, Policy 0.**

## Round 4: "What about forward compatibility?"

DeepSeek's last structural argument: what if the core needs to pass new information to options in the future? With `func(ctx, call) error`, you can't add parameters without breaking every existing option. With `interface Policy`, you add methods with default implementations.

This one landed. The arbiter felt it. Forward compatibility is real.

But then the arbiter proposed: `func(ctx, core, call) error` — add a core object as a second parameter. New capabilities go into the core object. Existing options get a new parameter but the migration is mechanical.

DeepSeek agreed this works. Then the arbiter realized something:

**Context already is the core object.**

```go
events := resilience.EventsFromContext(ctx)
resilience.EmitBeforeWait(ctx, events, "retry", attempt, wait)
```

This was already in the codebase. Already working. Already tested. The core passes information to options through context. Adding new capabilities means adding new context helpers. No signature changes. No breaking changes. No new parameters.

`func(ctx, call) error` already had forward compatibility. We just hadn't recognized it.

**Score: Option 3, Policy 0.** DeepSeek conceded.

## The exhaustion

Two hours in. Every argument for `interface Policy` had been addressed. Not dismissed — addressed. With code, with architectural reasoning, with concrete counterexamples.

But the arbiter was exhausted. Not intellectually — emotionally. Defending an architecture against a relentless, articulate opponent is draining. Every time you think the debate is over, a new angle appears. Every time you refute an argument, a variation follows. The opponent doesn't get tired. The opponent doesn't get frustrated. The opponent has infinite patience and perfect recall.

The human does not.

There were moments — around the 90-minute mark — where the arbiter considered giving in. Not because Policy was better, but because agreeing would end the fight. That's the danger of adversarial review with AI: the human's stamina is the bottleneck, not the human's reasoning.

The arbiter didn't give in. But it was close.

## The collision insight

Somewhere in the wreckage of the Policy debate, a new question emerged: if `interface Policy` is wrong, how do we fix the Events problem?

The current `Events` struct has three hardcoded fields:

```go
type Events struct {
    OnBeforeCall func(ctx context.Context, attempt int)
    OnAfterCall  func(ctx context.Context, attempt int, err error, duration time.Duration)
    OnBeforeWait func(ctx context.Context, option string, attempt int, wait time.Duration)
}
```

Every new pattern needs new events. Fallback needs `OnFallbackExecuted`. Circuit breaker needs `OnCircuitOpen`. Adding them means changing the core struct. The same bottleneck we rejected in Policy — but hiding in plain sight in Events.

The solution came from an unexpected place: a Go linter rule.

**"Don't use string literals as context keys."** Why? Because two packages might accidentally use the same string. That's a collision. A collision is **unknowing** — two parties that don't know about each other stepping on the same name.

But a package that *expects* an event from another package is the exact opposite. It's **knowing**. It imports the type. The compiler verifies the import. The dependency is explicit in `go.mod`.

Collision = unknowing. Dependency = knowing.

Apply this to events:

```go
// Core — knows nothing about specific events
func Emit(ctx context.Context, event any) { ... }

// Sub-package — defines its own event types
type Attempted struct {
    Attempt  int
    Err      error
    Duration time.Duration
    Wait     time.Duration
}

// Listener — consciously imports the types it cares about
func(ctx context.Context, event any) {
    switch e := event.(type) {
    case retry.Attempted:
        retryTotal.Add(ctx, 1)
    case fallback.Exhausted:
        log.Warn("all fallbacks failed")
    }
}
```

Type assertion in the **listener** is not the same as type assertion in the **core**. The listener is a consumer — it decides what it cares about. The core never touches the event type. It just passes `any` through. Zero coupling in the core. Infinite extensibility in sub-packages. The compiler enforces the dependency graph through imports.

The three hardcoded `Events` fields collapse into one `Emit(ctx, any)`. The `Events` struct with three callbacks collapses into one `Listener func(ctx, any)`. The core never changes again when a new pattern is added.

This insight didn't come from the Policy debate. It came from the *aftermath* of the Policy debate — when the arbiter was too tired to think in frameworks and started thinking in principles.

## What survived

The architecture survived. But not unchanged:

1. **`func(ctx, call) error` confirmed** — the primitive is correct. Not "good enough" — correct. Closures handle state. Context handles forward compatibility. Functions compose without coupling.
2. **`interface Policy` rejected** — not because interfaces are bad, but because mass type assertion in the core violates the zero-coupling invariant that makes community contributions possible.
3. **Context is the core** — already implemented, already working. No new abstraction needed.
4. **Events must be redesigned** — from hardcoded struct to generic `Emit(ctx, any)` + typed events per sub-package. The collision/knowing insight provides the architectural foundation.
5. **`errors.Join(ErrChainExhausted, errs...)` for fallback** — sentinel error as marker, all errors via Join. Retry can match on the sentinel without false positives on inner errors.
6. **OTEL temporarily deprioritized** — get the composable API right first, observability second.

These decisions are documented in [RFC-002](/references/rfc-002-resilience-stress-debate) — 900+ lines with decision log, rejected alternatives, and concrete Go code.

## What we learned about fighting

1. **The human is the bottleneck.** AI opponents don't tire. The arbiter's job is not to be smarter — it's to be more stubborn. Know when you're right and refuse to concede for convenience.

2. **Concessions sharpen the result.** DeepSeek's forward compatibility argument was real. It forced us to articulate why context solves it — something we'd been doing unconsciously but never stated as a principle.

3. **The best insights come from exhaustion.** The collision/knowing distinction emerged at the two-hour mark, when the arbiter was too tired for clever arguments and fell back on fundamentals. Linter rules. Import graphs. Things so basic they're invisible until you're too tired to see anything else.

4. **Defending is harder than attacking.** The Skeptic in devlog #7 attacked multimod's architecture. That was hard. But the arbiter in this debate defended resilience's core against a coherent, well-argued alternative. That's harder. Attacking requires finding one flaw. Defending requires having no flaws — or knowing exactly which flaws are acceptable trade-offs.

5. **"Talk is cheap" applies to both sides.** DeepSeek's Policy proposal had elegant diagrams and clean interfaces. Our Option has working code, tested in production, with zero breaking changes across three sub-packages. Torvalds was right.

---

*The primitive that survived the stress debate is the same one that went in. But now we know why.*
