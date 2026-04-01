---
layout: home

hero:
  name: resilience
  text: Composable resilience for Go
  tagline: "Zero-dependency core. Retry, backoff, plugins. One primitive: func(ctx, call) error."
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
    details: "Every resilience pattern is an Option: func(ctx, call) error. Retry, timeout, circuit breaker, bulkhead — same shape. Compose like Lego."
  - icon: 🪶
    title: Zero Dependencies
    details: "Core package has zero external dependencies. OTEL, gRPC, circuit breaker — opt-in via separate Go modules. go get pulls only what you use."
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
    details: "Born from a real project. Extracted from autosolve after killing a 1500-line framework. Battle-tested patterns, clean API."
    link: /devlog/
    linkText: Read the story →
---
