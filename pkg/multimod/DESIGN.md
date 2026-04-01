# multimod — Design

Multi-module management tool for Go monorepos. Zero config. Convention over configuration.

## Architecture

```
Boot → Kernel → Discovery → Executor → Runner → Command
```

| Component | Contract |
|---|---|
| **Boot** | Find project root. Pass to Kernel |
| **Kernel** | Orchestrate: Discovery → State → Executor. Stop on error |
| **Discovery** | Read files, build State. No judgments |
| **Executor** | Resolve command category, delegate to Runner |
| **FixableRunner** | Pre-invariant (Fixer) → Command → Post-invariant (Fixer) |
| **UnfixableRunner** | Validate State → Command. Die if State broken |
| **Fixer** | Fix what's broken, atomically (disk + model). Or return error |
| **Writer** | Fixer's tool. Write go.mod / go.work to disk |
| **Command** | Trust State. Do the work |

## Commands

```
multimod go <args>              — daily work, transparent proxy + multi-module iteration
multimod release <version>      — CI, strip replace + pin require + tag (--write to apply)
multimod generate               — templates → files from module model (--write to apply)
multimod verify                 — check + auto-fix state, print report
```

## State contract

Command receives State and trusts that:

1. Root module exists and is parsed
2. All sub-modules discovered and parsed
3. Dependency graph is acyclic (DAG)
4. Every sub-module has `replace` for all internal deps
5. `go.work` contains all modules
6. `go` directive is the same everywhere (root = source of truth)

## Fixable vs Unfixable

| Issue | Fixable | Action |
|---|---|---|
| Missing go.work | ✅ | Create |
| go.work missing module | ✅ | Add |
| go.work extra module | ✅ | Remove |
| Missing replace | ✅ | Add with relative path |
| go directive mismatch | ✅ | Sync to root |
| Cyclic dependency | ❌ | Error with hint |
| Corrupted go.mod | ❌ | Discovery dies |

## IO

- **stdout** — belongs to Go. Never touched.
- **stderr** — ours. `[multimod]` prefix. Silent when everything is ok.

## Full specification

See [docs/internals/multimod/](../../docs/internals/multimod/) for full spec, research, and rationale.
