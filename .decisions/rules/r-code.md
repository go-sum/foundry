# r-code — Go/Echo/pgx Coding Rules

> Ruleset for `cc-dev`. Read entirely before writing code.

---

## Go 1.26 Patterns

## Code Style

### Naming
- Functions: verb-first (`GetByID`, `UpdateUser`, `ParseToken`)
- Errors: `Err` prefix + noun (`ErrUserNotFound`, `ErrEmailTaken`)
- Interfaces: noun + `-er` suffix (`UserRepository`, `TokenVerifier`)
- Constructors: `New` (exported), `new` (package-private)
- Test fakes: `fake` prefix (`fakeUserRepo`)

### Structure
- Early returns over nested `if`
- One exported function per exported concern — no multi-purpose helpers
- Constructors accept dependencies as parameters (no global singletons outside config)
- No pre-allocated slices/maps unless capacity is known at call site
- Prefer declarative constructs over imperative loops — use `cmp.Or`, `slices`, `maps` stdlib when available

### External Module Imports
Shared external modules are ordinary Go dependencies — import like any third-party package.

Importing into `internal/`: prefer smallest surface — accept interfaces rather than concrete types when seam exists.

Never reach into external module internals to bypass public API. To change behavior, upstream to respective repository.

### Where to Put New Code
Consult ARCHITECTURE_GUIDE for ownership model. All code in this repository is app-owned.
