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
- [`DESIGN_GUIDE.md`](.decisions/DESIGN_GUIDE.md): authoritative source for the current architecture and ownership rules.
- [`PATTERNS_PRINCIPLES.md`](.decisions/PATTERNS_PRINCIPLES.md): authoritative source for how new functions, types, packages, and features should be structured and maintained.
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
