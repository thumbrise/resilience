---
title: "resilience Devlog #2 — The Task Runner Lifecycle Gap"
description: "Why no task runner has global hooks, why that matters, and why we used deps: [setup] anyway."
---

# #2 — The Task Runner Lifecycle Gap

> Six years of open issues. Zero solutions. Every project writes the same workaround.

## The Problem

resilience is a multi-module Go project. The core module has zero dependencies. Sub-modules (`otel/`, future `circuit/`, `grpc/`) have their own `go.mod` with external deps. For local development, Go needs a `go.work` file that lists all modules — otherwise cross-module imports don't resolve.

Simple need: before any task (lint, test, build), ensure `go.work` exists and includes all sub-modules.

## What We Tried

**Attempt 1: Manual `task setup`.** Run it once after clone. Works until you add a new sub-module and forget to re-run. Classic manual factor.

**Attempt 2: `deps: [setup]` on every task.** Taskfile's dependency mechanism. Setup runs before lint, test, generate — automatically. But you must remember to add `deps: [setup]` to every new task. Move the manual factor from "run a command" to "edit a YAML file". Better, but not solved.

**Attempt 3: Global hooks.** Surely a task runner has "run this before everything"? No. None of them do.

## The Ecosystem Survey

We checked every major task runner. The result was depressing.

**Make** (1976). Has `.DEFAULT` for missing targets. No mechanism for "before every target, even existing ones." People write `make init && make build` and pretend that's automation.

**Task** (go-task). [Issue #294](https://github.com/go-task/task/issues/294) — opened February 10, 2020. Requests global preconditions. Six years later: two PRs attempted (#1993, #2734), neither merged. The inclusion model (`Taskfile.Merge()`) treats everything as struct merging — vars, env, tasks. Lifecycle hooks don't fit this model. Adding them requires rethinking how included files inherit behavior. The maintainers chose to add Remote Taskfiles with TLS, LLM-optimized docs, and emoji output instead.

**Just** (Casey Rodarmor). Consciously rejected global hooks. "Just should be stateless, every command self-contained." Which means if you need to load env vars from a file, you repeat the loading logic in every recipe. Fifty recipes, fifty copies. The author calls this "self-sufficiency." We call it a design gap.

**Mage** (Go). Tasks are Go functions — full language power. But no middleware concept for tasks. Want to check a precondition before every task? Call the check function manually in every task function. You have goroutines, channels, generics — but not `BeforeAll`.

**Gulp** (JS). Has `series` and `parallel` for composition. No global "before everything." Plugin ecosystem works around it.

## The Root Cause

Every task runner treats tasks as isolated units with explicit dependencies. This is correct for the dependency graph. But it's incomplete — it ignores **lifecycle phases**.

Programming languages solved this decades ago:
- **Constructors/destructors** — guaranteed initialization before use
- **Middleware** — wrap every handler with cross-cutting concerns
- **DI containers** — lifecycle management as a first-class concept
- **`init()` in Go** — runs before `main()`, always

Task runners never adopted these ideas. They remained "glorified shell scripts" — which is fine until your project has invariants that must hold before any task runs.

## The Architectural Insight

The problem isn't missing features. It's a fundamental modeling gap.

Task runners model **what** to execute (commands) and **when** to execute it (dependencies). They don't model **phases of execution**:

```
Phase 1: Environment invariants (go.work exists, .env loaded, tools installed)
Phase 2: The actual task (lint, test, build)
Phase 3: Cleanup (optional)
```

A proper solution would separate these concerns:

```go
type TaskLifecycle interface {
    BeforeAll() error   // environment invariants — before any task
    BeforeTask() error  // per-task preconditions
    Execute() error     // the task itself
    AfterTask() error   // per-task cleanup
    AfterAll() error    // global cleanup
}
```

This is exactly what resilience does for function calls — `Option` wraps execution with behavior. The same pattern applies to task runners. Nobody has built it yet.

## What We Shipped

`deps: [setup]` on every task in `Taskfile.yaml`. The setup task auto-discovers all `go.mod` files and syncs `go.work`.

It works. It's a workaround. The manual factor moved from "remember to run setup" to "remember to add deps." Copy-pasting an existing task copies the deps too, so in practice it holds.

But we're aware: this is `make_u32_from_two_u16()` — a solution that works but doesn't solve the actual problem. The actual problem is that task runners don't have lifecycle phases.

## The Pattern

This is the same pattern that created resilience itself:

1. longrun was a 1500-line framework because Go had no composable resilience primitive → we built `resilience.Option`
2. Task runners have no lifecycle hooks because they model tasks as isolated units → someone will build a task runner with `BeforeAll`

Tools emerge from tools. ClickHouse emerged from Yandex internals. MapReduce emerged from Google's data processing. resilience emerged from autosolve's retry needs.

The task runner with lifecycle phases will emerge from someone's frustration with `deps: [setup]` in every task. Maybe ours.

---

*Workaround shipped. Problem documented. Clock is ticking.*
