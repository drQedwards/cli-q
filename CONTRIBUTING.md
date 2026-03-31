# Contributing to Supermodel CLI

## Prerequisites

- Go 1.25+
- [golangci-lint](https://golangci-lint.run/welcome/install/) — `brew install golangci-lint`
- [goreleaser](https://goreleaser.com/install/) — `brew install goreleaser`
- [pre-commit](https://pre-commit.com/) (optional) — `brew install pre-commit && pre-commit install`

## Setup

```sh
git clone https://github.com/supermodeltools/cli
cd cli
go mod tidy
make build
```

## Day-to-day workflow

```sh
make build        # compile to dist/supermodel
make test         # run tests with race detector + coverage
make lint         # run golangci-lint
make fmt          # format all .go files
make tidy         # go mod tidy + verify
make release-dry  # full GoReleaser snapshot build across all platforms
```

## Conventions

- Keep `main.go` thin — it only calls `cmd.Execute()`.
- Put all commands under `cmd/`. One file per subcommand.
- Put reusable logic under `internal/`. Nothing in `internal/` should import `cmd/`.
- Write table-driven tests. Aim for coverage on non-trivial logic.
- Follow standard Go error handling — wrap with `fmt.Errorf("...: %w", err)`.

## Submitting a PR

1. Branch from `main`.
2. `make test` and `make lint` must both pass.
3. Fill in the PR template.
4. Keep PRs focused — one concern per PR.

Questions? Email [abe@supermodel.software](mailto:abe@supermodel.software).
