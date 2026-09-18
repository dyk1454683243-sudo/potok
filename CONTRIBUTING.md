# Contributing to Potok

Thanks for wanting to help. This guide covers the basics.

## Setup

1. Fork and clone the repository.
2. Install Go 1.25+.
3. From the repo root:

```bash
go test ./...
go build -o potok ./cmd/potok
go build -o potokd ./cmd/potokd
```

## Code style

- Keep packages under `internal/` focused and small.
- Prefer clear error messages for CLI failures.
- Add or update tests next to the code you change (`*_test.go`).

## Pull requests

1. Create a branch from `main` (`git checkout -b fix/short-description`).
2. Make a focused change (one concern per PR).
3. Run `go test ./...` before opening the PR.
4. Open a PR against `main` with a short summary of *why* the change is needed.

## Issues

- Check existing issues before opening a new one.
- Good first issues are labeled `good first issue`.

## Questions

Open an issue if something in the docs or setup is unclear.
