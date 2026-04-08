---
layout: home

hero:
  name: resilience
  text: Composable fault tolerance library for Go
  tagline: '<a href="https://github.com/thumbrise/resilience/releases"><img src="https://img.shields.io/github/v/release/thumbrise/resilience?label=latest&color=blue" alt="Latest Release" style="display:inline-block;vertical-align:middle;margin-bottom:8px" /></a><br>Open source resilience middleware for Go. Retry, backoff — battle-tested. Circuit breaker, rate limiter — on the roadmap. One primitive: func(ctx, call) error.'
  actions:
    - theme: brand
      text: Quick Start
      link: /guide/getting-started
    - theme: alt
      text: Devlog
      link: /devlog/
    - theme: alt
      text: GitHub
      link: https://github.com/thumbrise/resilience

features:
  - icon: 🧱
    title: One Primitive
    details: "Every resilience pattern is an Option: func(ctx, call) error. Retry, backoff — ready. Timeout, circuit breaker, rate limiter — same shape, planned."
  - icon: 🪶
    title: Zero Dependencies (target)
    details: "Target architecture: zero-dependency core, OTEL and future plugins in separate Go modules. Blocked on multimod (extracted to standalone repo) — today everything lives in one module. Tracking in devlog #3."
    link: /devlog/003-multimod-gap
    linkText: Why it's blocked →
  - icon: 🔌
    title: Two Extension Points
    details: "Option controls execution (per-call). Plugin observes execution (shared state). Two words, two contracts, no confusion."
  - icon: 📊
    title: OTEL Metrics Built-in
    details: "rsotel.Plugin() — one line. Calls, errors, retries, backoff waits — all metered. No OTEL SDK? Zero overhead (no-op instruments)."
  - icon: 🎯
    title: Per-Call State
    details: "Each Do() gets fresh Options. No shared mutable state. No data races by construction, not by mutex."
  - icon: 🚧
    title: Active Development
    details: "Born from killing a 1500-line framework inside autosolve. Retry and backoff are battle-tested. Circuit breaker, rate limiter, bulkhead — planned. This is a growing toolkit, not a finished product."
    link: /guide/roadmap
    linkText: See the roadmap →
---
