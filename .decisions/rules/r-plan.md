# r-plan — Architecture & Planning Rules

> Ruleset for `cc-plan`. Read entirely before producing any plan.

---

## Pre-Planning Checklist

1. **Layer?** Transport / Service / Repository / Database
2. **Already exists?** Use LSP (`lsp_workspace_symbols`, `lsp_find_references`) before proposing new code.
3. **Minimum change?** No abstractions, helpers, or config keys not required by the task.

---

## Feature Development Sequence (8-step order)

1. **Model** — domain structs and error sentinels (shared only, or inline in feature package)
2. **SQL** — schema changes and queries (don't add schema owned by external modules — consume it)
3. **Repository** — `toXxxModel()` mapper; wrap sqlc queries; map DB errors to domain errors
4. **Service** — business rules; call repo via interface; return domain types
5. **Routes** — register named routes
6. **Handler** — parse → validate → service call → render/redirect
7. **Views** — full-page + HTMX partial; `view.Render(c, req, fullPage, partial)`
8. **Module wiring** — expose handlers; wire in `internal/app/`
9. **Tests** — handler, service, repository, view (delegate to `cc-test`; co-located)

Never skip or reorder steps.

---

## Route Design Rules

- Single source of truth: `internal/app/routes.go`
- Every route MUST have `Name` field (URL reversal, sitemap)
- Convention: `<resource>.<action>` (e.g., `user.list`, `home.show`)
- Group by scope: `public`, `protected`, `admin`
- Parameterized URL builders live alongside route names
- HTMX-only fragment endpoints are separate named routes

---

## Error Classification

| Category | Type | Location | When |
|----------|------|----------|------|
| Domain | named sentinels (`ErrUserNotFound`) | `internal/model/errors.go` | Expected business failures |
| Application | `*web.Error` with status+code | `pkg/web/errors.go` | Transport-facing, safe message |
| Infrastructure | plain `error` | anywhere | Unexpected; log + generic message |

Handlers map domain → application errors. Services return domain errors. Repositories return domain or plain errors.

---

## HTMX Rendering Strategy

- Every HTML route has **two modes**: full-page and partial
- `view.Render(c, req, fullPage, partial)` auto-detects HTMX via headers
- HTML regions for independent replacement — suffix with `Region` (e.g., `UserRowRegion`)
- HTMX attributes (`hx-get`, `hx-target`, `hx-swap`) belong in **views**, not handlers
- HTMX-only endpoints: `render.Fragment(c, component)`
- Post-form navigation: component library redirect pattern
- CSRF: inject via `hx-headers='{"X-CSRF-Token": "..."}'`

---

## Plan Output Format

Every plan MUST include:

```markdown
## Context
Why this change is needed; what problem it solves.

## Layer Placement
Which layer(s) are affected and why. Confirm no layer violations.

## Files to Modify / Create
| Action | File | What changes |

## Implementation Steps
Numbered, ordered steps. Each step references a specific file and function.

## New Types / Interfaces
Any new structs, interfaces, or error sentinels with their signatures.

## Test Plan
What cc-test should verify (happy path + failure cases).
```
