# Foundry

## Architecture

### The monorepo

The `pkg/*` modules are separate Go modules developed here and published as independent `github.com/go-sum/*` packages.
The `starter/` application consumes them as first-class external dependencies (wired via `go.work` during development; `go.prod.mod` for production releases).

The starter application is both a reference implementation and a minimal application starter for application designs that leverage the monorepo packages

### Two Design Zones

| Zone | Location | What lives here |
|---|---|---|
| App-owned | `starter/` | Product logic, page composition, app-specific orchestration |
| Package-owned | `pkg/<name>/` | Reusable capabilities deployable independently |

Module versions are maintained in `.versions`.
