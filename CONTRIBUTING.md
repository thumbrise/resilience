# Contributing

## Requirements

- **Go 1.24.x** — exact major.minor match required.
- **[golangci-lint](https://golangci-lint.run/welcome/install/)** — install locally. Version must match `GOLANGCI_LINT_VERSION` in `Taskfile.yaml`.
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

# Run lint (requires golangci-lint installed locally)
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
otel/                  OTEL metrics Plugin (sub-package, same root module)
_tools/                Dev tools: license-eye, govulncheck (separate go.mod)
pkg/multimod/          Multi-module tooling (WIP, see docs/internals/multimod/)
docs/                  VitePress documentation site
```

## Multi-module (planned)

Currently the project is a single Go module. All packages (core, otel, multimod) share one `go.mod`. `go get` pulls all transitive dependencies including the OTEL SDK.

Zero-dependency core via separate Go modules is the [target architecture](docs/internals/multimod/), blocked on the multimod tool we're building. See [devlog #3](/docs/devlog/) for the full story.

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
