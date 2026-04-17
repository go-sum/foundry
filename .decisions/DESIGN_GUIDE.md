---
title: Design Principles
description: Architecture, ownership boundaries, runtime assembly, and routing/rendering guidance for this application.
weight: 20
---

# Design Principles

> This guide is the authoritative source for the current architecture and ownership rules.
>
> Read this together with [`CLAUDE.md`](../CLAUDE.md),
> [`ERROR_HANDLING_GUIDE.md`](./ERROR_HANDLING_GUIDE.md),
> [`PATTERNS_PRINCIPLES.md`](./PATTERNS_PRINCIPLES.md), and
> [`UI_GUIDE.md`](./UI_GUIDE.md).

---

## 1. Purpose

This is a server-rendered Go web application built around:

- Go types that directly model W3C Web API primitives (Request, Response, Headers, ReadableStream)
- Gomponents for HTML rendering
- HTMX for progressive enhancement
- Reusable external modules from `github.com/go-sum/*`

This guide answers:

- where code belongs
- which layer owns a domain
- how the application is assembled at runtime
- how routing, rendering, persistence, and startup behavior work today

This guide does **not** define low-level coding style, function structure, or
general design-pattern usage. Those rules now live in
[`PATTERNS_PRINCIPLES.md`](./PATTERNS_PRINCIPLES.md).

---

## 2. Current Architecture Overview

The application has one source zone:

- `internal/` for all code authored in this repository

External reusable modules from `github.com/go-sum/*` are ordinary Go
dependencies consumed via `go.mod`. They are not part of this repository.

Runtime assembly is centered in `internal/app`. The composition root wires:

- config loading
- logging
- asset registration
- security and middleware
- database pool and migrations
- sessions
- queue client and background services
- external auth, site, and storage modules wired in from `github.com/go-sum/*`
- app-owned feature modules and views

The current application is intentionally hybrid:

- some domains are provided by external modules and integrated into the app,
  such as auth, queue storage, sessions, senders, and site metadata handlers
- some domains are app-owned, such as contact flow, availability handling, and
  page composition

---

## 9. Rendering Model

The application supports multiple HTML response modes without splitting into
separate rendering stacks.

### Canonical rendering modes

| Mode | Handler pattern |
|------|-----------------|
| full page + HTMX partial | `view.Render(c, req, fullPage, partial)` |
| fragment-only | `render.Fragment(c, node)` or `render.FragmentWithStatus(c, status, node)` |
| HTMX removal | `c.String(http.StatusOK, "")` |
| JSON/problem | selected by the global error handler based on request headers |
| redirect | HTMX-aware redirect helpers |

### Rules

- Use `view.NewRequest(...)` to build request-scoped presentation state.
- Use `view.Render(...)` when one endpoint serves both full-page and HTMX
  partial modes.
- Use `render.Fragment(...)` only when the endpoint exists purely for fragment
  swapping.
- Let the global error handler decide between HTML, HTMX fragment, and problem
  JSON responses.

---

## 13. How The Guides Fit Together

Use the decision docs this way:

- [`DESIGN_GUIDE.md`](./DESIGN_GUIDE.md): where code belongs and how the app is
  assembled
- [`ERROR_HANDLING_GUIDE.md`](./ERROR_HANDLING_GUIDE.md): how unexpected
  behavior should be returned, logged, classified, and considered for
  notification
- [`PATTERNS_PRINCIPLES.md`](./PATTERNS_PRINCIPLES.md): how new code should be
  structured and maintained
- [`UI_GUIDE.md`](./UI_GUIDE.md): visual and UI composition guidance

---
