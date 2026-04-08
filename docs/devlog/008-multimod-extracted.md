---
title: "resilience Devlog #8 — multimod extracted"
description: "The tool that outgrew its host. multimod moved to a standalone repository — and we immediately felt its absence."
head:
  - - meta
    - name: keywords
      content: go multi-module extraction, golang monorepo tooling, multimod standalone, package extraction, go.work management
---

# #8 — multimod extracted

> "We deleted multimod and immediately drowned in the problems it solved."

## What happened

[multimod](https://github.com/thumbrise/multimod) — the multi-module management tool we built inside resilience — has moved to its own repository. The code, the docs, the RFC, the spec, the FAQ — everything lives at [github.com/thumbrise/multimod](https://github.com/thumbrise/multimod) now.

resilience is back to being what it should be: a fault tolerance library. One `go.mod`. No CLI tools. No release infrastructure. Just `func(ctx, call) error`.

## The timeline

Six devlogs led to this moment:

- [#3](/devlog/003-multimod-gap) — discovered the multi-module gap in Go
- [#4](/devlog/004-multimod-implementation) — built Discovery → State → Applier
- [#5](/devlog/005-multimod-release-design) — designed detached commit releases
- [#7](/devlog/007-adversarial-architecture-review) — stress-tested the architecture in an adversarial review

Each devlog added a piece. The tool grew. And at some point, multimod had more code than resilience itself. A CLI framework (cobra), a module parser (x/mod), a release pipeline, a standalone binary (multirelease), an RFC with 500+ lines. All living inside a retry library.

The extraction was overdue.

## The proof it was needed

We deleted `cmd/multimod/` and `cmd/multirelease/` from resilience. Within minutes:

- `go tool govulncheck ./...` stopped working — tools lived in `_tools/go.mod`, invisible without `go.work`
- `task lint:ci` broke — same reason
- `task test` broke — it was calling `go run ./cmd/multimod go test ./...`
- `_tools/go.mod` had stale `replace` directives pointing to deleted `otel/go.mod`

We spent an hour fixing what multimod handled automatically. The best proof a tool is needed is removing it and watching everything break.

## What resilience looks like now

```
resilience.go          Core: Option, Plugin, Events, Client, CallBuilder, Do
sleepctx.go            Context-aware sleep
backoff/               Pure math: Exponential, Constant, Default
retry/                 Retry Option: On, OnFunc, WithWaitHint
otel/                  OTEL metrics Plugin (sub-package, same root module)
_tools/                Dev tools: license-eye, govulncheck (separate go.mod)
go.work                Workspace: root + _tools
docs/                  VitePress documentation site
```

One module. `go get github.com/thumbrise/resilience` pulls everything including OTEL. That's the mono-module tax we pay until multimod ships stable releases.

## What multimod looks like now

```
github.com/thumbrise/multimod/
├── go.work                     # manages itself (dog-fooding)
├── multimod/                   # CLI binary #1 — dev-state guardian
├── multirelease/               # CLI binary #2 — publish-state creator
└── docs/                       # RFC, spec, FAQ, devlogs
```

Two modules, own `go.work`, multimod manages itself. Dog-fooding from day one. Build ✅, tests ✅, lint ✅. The tool works on its own codebase.

## The SEO redirect

Every old URL at `thumbrise.github.io/resilience/internals/multimod/` now has a meta-refresh redirect to the new docs at `thumbrise.github.io/multimod/`. No 404s. Bookmarks work. Google will follow the redirects and update its index.

The devlogs stay here — they're resilience history, not multimod docs.

## The pattern continues

| Problem | Status | Where |
|---|---|---|
| Resilience patterns in Go | ✅ Standalone | [resilience](https://github.com/thumbrise/resilience) |
| Package extraction | ✅ Proved | autosolve → resilience |
| Multi-module tooling | ✅ Extracted | [multimod](https://github.com/thumbrise/multimod) |
| Zero-dependency core | ⏳ Blocked on multimod | One `go.mod` for now |

Each entry: concrete pain → honest research → minimal primitive → extract when ready.

multimod followed the same path as resilience itself. Born inside a project. Grew beyond it. Extracted when the boundary was clear.

---

*The tool that outgrew its host now manages its own home.*