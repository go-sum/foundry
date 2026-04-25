# CLAUDE.md — Architectural Constitution

> A Go web application built on Go types that directly model W3C Web API primitives (Request, Response, Headers, ReadableStream) leveraging HTMX, and server-rendered HTML.
>
> See `README.md` for project-specific architecture, module names, and current state.

---

## Behavioral Rules (always enforced)

- ONLY do what has been asked — recommend and get approval before any additions
- NEVER create documentation files (`*.md`) unless explicitly requested
- NEVER hardcode API keys, secrets, or credentials in source files
- NEVER commit secrets, credentials, or `.env` files
- ALWAYS validate user input at system boundaries; sanitize file paths (prevent `../` traversal)
- ALWAYS ensure implementations leverage the project's shared security module
- ALWAYS run tests after making code changes
- ALWAYS trace ALL callers when refactoring Go config structs or YAML mappings
- ALWAYS account for HTML-encoded entities in test assertions for HTML output
- ALWAYS enforce exact-match test assertions — never substring matching
- ALWAYS use LSP (`mcp__gomcp__lsp_*`) ahead of Grep/Glob for Go code navigation
- FALLBACK to Grep only for non-code text or when `gomcp` MCP server is unavailable

---

## Guide Index
> Before writing code, depending on the requirement consult:
- [`ARCHITECTURE_GUIDE.md`](.decisions/ARCHITECTURE_GUIDE.md): project structure, dependency injection, routing, server design, graceful shutdown, and design patterns.
- [`DESIGN_PATTERNS.md`](.decisions/DESIGN_PATTERNS.md): handler, middleware, service, logging, error taxonomy, validation, testing, and resilience patterns.
- [`CODE_REVIEW.md`](.decisions/CODE_REVIEW.md): review checklists, severity calibration, verification protocol, and valid patterns.
- [`DATA_STORAGE.md`](.decisions/DATA_STORAGE.md): driver selection, connection pooling, migrations, transactions, and the repository pattern.
- [`WEB_DESIGN.md`](.decisions/WEB_DESIGN.md): concurrency, worker pools, rate limiting, race detection, and runtime safety.
- [`UI_GUIDE.md`](.decisions/UI_GUIDE.md): defines how UI should be designed and composed.

---

## MCP Server — gomcp (LSP)

Registered in `.mcp.json`. Available in all agents. Prefer over Grep/Glob for Go.

| Tool | Use |
|------|-----|
| `mcp__gomcp__lsp_workspace_symbols` | Find types, functions, interfaces by name |
| `mcp__gomcp__lsp_find_references` | All callers / all implementors |
| `mcp__gomcp__lsp_definition` | Jump to any symbol definition |
| `mcp__gomcp__lsp_document_symbols` | Inventory all symbols in a file |
| `mcp__gomcp__ping` | Verify server availability |

---

## Development Phase Guide

Invoke the right agent for each phase. Each agent reads its paired rules file first.

| Phase | Agent | Rules | When |
|-------|-------|-------|------|
| Analysis & Design | `cc-plan` | `.claude/rules/r-plan.md` | Before any code — layer assignment, architecture |
| Implementation | `cc-dev` | `.claude/rules/r-code.md` | After plan approved — write code in correct layers |
| Testing | `cc-test` | `.claude/rules/r-test.md` | After implementation — happy-path + failure tests |
| Architecture Review | `cc-plan` | `.claude/rules/r-plan.md` | After tests pass — refactor planning |

Agent flow: `cc-plan` → `cc-dev` → `cc-test` → (if issues) back to `cc-plan`

Agents and rules live in `.claude/agents/` and `.claude/rules/`.
