---
title: Patterns and Coding Principles
description: Coding standards, design patterns, and maintainability rules for this application.
weight: 25
---

# Patterns and coding principles:

> This guide is the authoritative source for how new functions, types, packages,
> and features should be structured and maintained in this application.
>
> It complements [`DESIGN_GUIDE.md`](./DESIGN_GUIDE.md), which defines
> architecture and ownership, and [`ERROR_HANDLING_GUIDE.md`](./ERROR_HANDLING_GUIDE.md),
> which defines how unexpected behavior should be handled and logged.
>
> This guide is derived from:
>
> - Go best practices from Effective Go and Go Code Review Comments
> - recognized Go design-pattern guidance from Refactoring Guru
> - successful patterns already working in this codebase today

---

## 1. Purpose

Use this guide when creating or refactoring code to answer:

- how a new function should be shaped
- how dependencies and config should be owned
- when to add a pattern versus keeping code simple
- how to preserve maintainability, testability, and package boundaries

This guide is intentionally normative. When in doubt, follow the simplest
approach that keeps ownership clear and leaves the code easy to test.

---

## 2. Core Programming Principles

### DRY

Reduce repetition in behavior, policies, and data mapping, but do not create a
shared abstraction until the duplication is real and stable.

### YAGNI

Be conservative. Do not add hooks, wrappers, config keys, or extension points
for hypothetical future use.

### Separation of Concerns

Split code by responsibility:

- transport concerns in handlers
- orchestration and business rules in services
- persistence in owning stores/repositories
- presentation in views
- assembly in the composition root

### SOLID, applied pragmatically

- **SRP**: each function, type, and file should have one primary reason to change.
- **ISP**: prefer narrow interfaces with a clear consumer.
- **DIP**: depend on abstractions only where that reduces coupling at a real
  boundary; do not create speculative interfaces.

### Favor composition and encapsulation

Prefer small collaborating types over inheritance-style layering or broad
utility packages. Expose the smallest public API that the next caller actually
needs.

---

## 3. Function And Type Design

When creating a new function:

- keep it focused on one job
- make the happy path obvious
- prefer early returns over nested branching
- keep side effects explicit

### Function signatures

- Pass `context.Context` as the first argument for any function that performs I/O or depends on cancellation.
- Parse and validate external input at the system boundary.
- Pass typed values, not raw strings, once input crosses the boundary.
- Return ordinary Go errors; do not use panics for expected failures.

### Error handling

- Wrap propagated errors with `%w` and enough context to identify the failing operation.
- Use `errors.Is` and `errors.As`, never string matching on error messages.
- Keep domain errors near the owning domain.
- Map domain errors to transport responses at the handler layer.

### Type design

- Prefer concrete types by default.
- Add an interface only when a consumer needs a seam for substitution or the
  package already has multiple valid implementations.
- Do not create field-for-field wrapper structs that add no semantics.
- Keep helper functions private unless another package truly needs them.

---

## 4. Package Structure And Layer Discipline

### Layer rule

Do not let a lower layer depend on a higher one:

- handlers should not own data
- services should not render HTML
- repositories should not decide redirects or HTTP status codes
- views should not own business rules or persistence

---

## 5. Dependency Construction

Dependencies should be explicit and constructor-injected.

### Rules

- Constructors should accept the dependencies they need.
- The composition root should assemble concrete implementations.
- Feature code should receive collaborators through constructors, not by reading package globals.
- Hidden global state should be the exception, not the default.

### Config defaults

Use `cmp.Or` to apply zero-value defaults in constructors:

```go
cfg.TokenTTL = cmp.Or(cfg.TokenTTL, time.Hour)
cfg.HeaderName = cmp.Or(cfg.HeaderName, "X-CSRF-Token")
```

Do not use `if x == "" { x = "default" }` blocks — they add visual noise with no semantic gain.

`cmp.Or` applies to any `comparable` type whose zero value means "unset": `string`, `int`,
`time.Duration`, etc. It does **not** apply to `func` fields or nil-sentinel interface fields;
guard those with explicit `!= nil` checks.

### Singletons

Use the Singleton pattern sparingly and intentionally.

It is appropriate when:

- the process should own exactly one shared registry or coordinator
- the lifetime is truly application-wide
- the singleton does not hide request-specific or feature-specific state

Prefer a single runtime-owned instance over scattered globals when the resource is application-specific.

---

## 6. Recommended Design Patterns

Patterns are tools, not goals. Use them where they simplify the code that
exists today.

### Factory / registry

Use a factory or registry when protocol or provider selection is a real
requirement.

This is the preferred pattern for:

- sender/provider selection
- transport or backend selection
- constructing different implementations behind one small entry point

### Chain of Responsibility

Use a chain when the request or operation should pass through a sequence of
orthogonal behaviors.

This is the preferred pattern for:

- middleware stacks
- request guards
- layered cross-cutting policies such as rate limits, auth checks, and origin
  protection

### Adapter

Use adapters when an external-module interface must be satisfied by app-owned
rendering, session, form, or redirect behavior.

### Resource-owner singleton

Use a single shared instance when a resource should be created once and reused,
but keep ownership explicit.

This is appropriate for:

- registries
- connection pools
- long-lived runtime-managed services

Avoid using the Singleton pattern as an excuse to hide dependencies.

---

## 7. Concurrency And Lifecycle

Apply concurrency deliberately, not casually.

### Default stance

Prefer synchronous code first. Add goroutines, workers, or async dispatch only
when there is a clear correctness, latency, or throughput reason.

### Rules

- Every goroutine must have a clear owner.
- Every long-lived background activity must have a shutdown path.
- Cancellation should flow from a passed-in context.
- Do not create fire-and-forget goroutines from handlers or services.
- If work must outlive the request, hand it to an owned background subsystem
  such as the queue.

---

## 10. Testing And Refactoring Discipline

Code should be easy to test at the boundary that owns behavior.

### Rules

- Test behavior, not internal implementation details.
- Prefer fakes over mocks.
- Cover both happy paths and failure paths.
- Keep HTML assertions exact where practical, and account for HTML-encoded
  entities.
- When refactoring config structs or mappings, trace every caller and every
  mapping that depends on them.

### Refactor threshold

Refactor when code becomes:

- duplicated in a stable way
- difficult to test
- unclear about ownership
- too broad for one type or file to explain cleanly

Do not refactor just to increase abstraction count.

---

## 11. Anti-Patterns To Avoid

- speculative abstractions
- package globals used as hidden request-time dependencies
- duplicated defaults across app and external-module layers
- field-for-field wrapper types with no semantic change
- mirrored schemas or query code for a domain owned by an external module
- transport code that owns SQL or business rules
- business logic embedded in views
- unbounded goroutines or background work with no lifecycle
- hardcoded route paths where named routes already exist

If a simpler function, struct, or constructor solves the problem clearly, use
that instead of reaching for a pattern.

---

## 13. Sources

- Effective Go: <https://go.dev/doc/effective_go>
- Refactoring Guru, design patterns in Go: <https://refactoring.guru/design-patterns/go>
