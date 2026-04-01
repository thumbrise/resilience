# Contributing

## Requirements

- **Go 1.24.x** — exact major.minor match required.
- **Docker** — golangci-lint runs in a container to guarantee consistent results regardless of your local setup.
- **[Task](https://taskfile.dev/)** — task runner. Install: `go install github.com/go-task/task/v3/cmd/task@latest`
- **Node.js** (optional) — only for commitlint and docs build.

## First time setup

```bash
git clone https://github.com/thumbrise/resilience.git
cd resilience
go test ./...
```

## Development workflow

```bash
# Run tests
go test ./... -v

# Run lint (Docker, no local golangci-lint needed)
task lint

# Fix license headers
task generate
```

## Project structure

```
resilience.go          Core: Option, Plugin, Events, Client, CallBuilder, Do
sleepctx.go            Context-aware sleep (used by sub-packages)
backoff/               Pure math: Exponential, Constant, Default
retry/                 Retry Option: On, OnFunc, WithWaitHint
otel/                  OTEL metrics Plugin (separate go.mod, external deps)
_tools/                Dev tools: license-eye, govulncheck (separate go.mod)
pkg/multimod/          Multi-module tooling (WIP, see docs/internals/multimod/)
docs/                  VitePress documentation site
```

## Multi-module (planned)

Currently the project is a monorepo with a single root module. Only `otel/` is a separate module (to keep core zero-deps).

Full multi-module support (separate `go.mod` per sub-package) is planned and blocked on the [multimod](docs/internals/multimod/) tool we're building. See [devlog #3](/docs/devlog/) for the full story.

## Writing an Option

Any resilience pattern is a function:

```go
func MyPattern() resilience.Option {
    return func(ctx context.Context, call func(context.Context) error) error {
        // before
        err := call(ctx)
        // after
        return err
    }
}
```

## Writing a Plugin

Plugins observe — they don't control execution:

```go
type myPlugin struct{}

func (p *myPlugin) Name() string { return "my-plugin" }

func (p *myPlugin) Events() resilience.Events {
    return resilience.Events{
        OnAfterCall: func(ctx context.Context, attempt int, err error, d time.Duration) {
            // observe
        },
    }
}
```

## Commit messages

Conventional commits. English only.

```
feat(retry): add jitter support
fix(backoff): handle overflow on large attempts
docs: update getting-started guide
```

Only `feat` and `fix` trigger releases. See [REVIEW.md](REVIEW.md) for full guidelines.

## Tests

- All tests use `package xxx_test` (blackbox only).
- Bug fix = test first: red test commit, then fix commit. Never combined.
- Run `task test` before pushing.

## Code review

See [REVIEW.md](REVIEW.md) — structural review rules, naming conventions, hard thresholds.
