# Code Review Instruction: Architectural Compliance Review Against DESIGN_GUIDE.md
### at 02:30 cat .decisions/CODE_REVIEW.md | claude --permission-mode bypassPermissions -p "code review"

You are performing an **architectural compliance review** of a Go monorepo application.

Your objective is to review the repository and determine whether the implementation complies with the architectural ownership, layering, runtime assembly, and rendering principles defined in `.decisions/DESIGN_GUIDE.md`. Do not implement the changes. If issues are found, create a plan to resolve the issues

---

## 1. Primary Objective

Audit the monorepo to verify that:

1. package ownership boundaries are respected
2. runtime composition responsibilities are correctly assigned
3. rendering patterns align with the rendering model
4. reusable packages remain modular and do not leak application concerns
5. the starter application composes packages correctly without violating ownership boundaries
6. infrastructure concerns are separated from domain concerns
7. the monorepo follows the intended "starter app + reusable packages" architecture

The authoritative architecture guide is:

* `DESIGN_GUIDE.md`

Use this guide as the source of truth for:

* ownership rules
* layering boundaries
* runtime assembly rules
* routing/rendering responsibilities
* infrastructure composition expectations

Treat each package as a reusable module with a clearly defined ownership boundary.

---

The `starter/` application must act as the **composition root**.

It should be responsible for wiring:

* config loading
* logging
* asset registration
* middleware
* database connections
* session setup
* queue services
* route registration
* package integration

Reusable packages under `pkg/*` must **not** perform application bootstrapping.

Flag violations such as:

* packages starting services during import/init
* packages registering routes automatically
* packages owning app startup logic
* packages coupling directly to starter-specific infrastructure

---

### B. Package Ownership Boundaries

Each `pkg/*` module must own only its domain.

Examples:

#### `pkg/assets`

Should own:

* asset build pipelines
* manifest resolution
* asset helpers

---

### C. Dependency Direction

Dependencies must flow inward toward reusable primitives.

Expected dependency direction:

* `starter` → may depend on all packages
* `pkg/queue` → may depend on `pkg/db`, `pkg/kv`
* `pkg/web` → should not depend on `starter`
* `pkg/componentry` → should not depend on `pkg/web`, `pkg/db`
* `pkg/db` → should not depend on `pkg/web`

Flag:

* circular dependencies
* upward dependencies into `starter`
* UI packages importing infrastructure packages
* infrastructure packages importing application code

---

### D. Runtime Assembly Rules

Per the design guide, runtime assembly belongs in the application layer.

Verify that:

* service construction happens in `starter/`
* dependencies are injected into packages
* packages expose constructors/factories rather than self-bootstrapping
* no package silently initializes infrastructure internally

Flag:

* hidden singletons
* implicit globals
* package-level infrastructure initialization
* init-based bootstrapping

---

### E. Rendering Responsibility

The rendering model must respect separation between:

* UI composition
* HTTP transport
* application orchestration

Verify:

* `pkg/componentry` owns UI components only
* `pkg/web` owns transport/routing only
* `starter` coordinates route + handler composition

Flag:

* components directly handling HTTP requests
* web package embedding domain rendering logic
* handlers mixed into UI packages

---

### F. Infrastructure Isolation

Infrastructure modules should be isolated and composable.

Verify that:

* `pkg/db`, `pkg/kv`, `pkg/notification`, `pkg/queue` are independent services
* integrations occur via interfaces or explicit constructors
* no hidden assumptions about global app state

Flag:

* direct hard-coded cross-package assumptions
* infrastructure modules tightly coupled to each other
* runtime dependencies hidden behind globals

---

### G. Reusability Compliance

All `pkg/*` packages must remain reusable outside `starter`.

Flag any reusable package that assumes:

* starter routes
* starter config paths
* starter environment variables
* starter-specific templates
* starter-specific global state

The review should identify anything that prevents extracting the package for reuse.

---

## 4. Required Output Format

Return findings in the following format:

---

### 1. Architectural Compliance Summary
### 2. Boundary Violations
### 3. Dependency Violations
### 4. Runtime Assembly Violations
### 5. Refactor Recommendations

---

## 5. Review Standard

Be opinionated and strict.
If a package mixes concerns, flag it.
If a reusable package contains application logic, flag it.
If composition happens outside `starter`, flag it.
If dependency direction is wrong, flag it.
Architectural purity always takes precedence over convenience.
