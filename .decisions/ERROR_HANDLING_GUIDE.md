---
title: Error Handling Guide
description: Structured guidance for handling, logging, and notifying on unexpected behavior.
weight: 27
---

# Error Handling Guide

> This guide defines how unexpected behavior should be classified, handled,
> logged, and considered for notification across reusable packages and the
> starter application.
>
> It complements [`CLAUDE.md`](../CLAUDE.md),
> [`DESIGN_GUIDE.md`](./DESIGN_GUIDE.md),
> [`PATTERNS_PRINCIPLES.md`](./PATTERNS_PRINCIPLES.md), and
> [`UI_GUIDE.md`](./UI_GUIDE.md).
>
> Use this guide to answer:
>
> - what should be returned versus logged
> - where errors should be classified and rendered
> - when panic is appropriate
> - which failures are notification-worthy

---

## 0. Prescriptive Intent

This guide is **prescriptive**. It defines the policy that packages and the
application follow. No rule in this guide may be weakened; instead, drift is
surfaced in code review and addressed in the next refactor pass.

---

## 1. Purpose

This repository maintains building blocks for structured handling:

- `pkg/web/errors.go` defines transport-facing error categories and the `*web.Error` transport primitive (see 3.5)
- `pkg/web/boundary.go` defines HTTP boundary classification, panic recovery, and rendering
- `pkg/web/serve/logging.go` defines a baseline request log

The rendering half of the contract lives in `UI_GUIDE.md` and its reference
implementation at `starter/internal/view/errorpage/`. The server produces a
`*web.Error`; the error page consumes it. Both halves must be read together.

---

## 2. Core Principles

### Return by default

Unexpected behavior should usually be returned as an error to the owning
boundary. Do not log at every stack frame.

### Log at ownership boundaries

The code that can add operational context and decide the impact should log.
Most intermediate layers should wrap and return.

### Classify at transport boundaries

Reusable packages should not decide HTTP status codes or response shapes.
Handlers and transport boundaries own mapping from domain/runtime failure to
user-visible response.

### Notify only on actionable server-side failures

Notification is stricter than logging. A condition should trigger notification
only when someone can and should act on it.

### Panic only for programmer or invariant faults

Panics are reserved for impossible states, broken construction, or internal
contract violations. Expected runtime failures must return ordinary errors.

---

## 3. Shared Taxonomy

Use the same categories across packages and the app.

### Expected domain outcomes

These are ordinary, modeled outcomes:

- not found in a valid lookup
- validation failures
- version conflicts
- unauthorized or forbidden actions

Handling:

- return typed or sentinel errors
- do not log by default
- map to transport responses at the boundary
- do not notify

### Client or input misuse

These are malformed or unsupported requests from the caller:

- invalid content type
- oversized body
- malformed headers or form input
- unsupported parameter values

Handling:

- return contextual errors
- classify to 4xx at the boundary
- log only if the rate, source, or pattern matters operationally
- do not notify by default

### Dependency or integration failures

These are failures in storage, network, queue, filesystem, or downstream
services:

- database timeout
- failed write to store
- upstream 5xx or malformed dependency response
- response-body write failure

Handling:

- wrap with the operation and dependency context
- return to the owning boundary
- log at the boundary that understands impact; do not also log in intermediate layers
- notify only if actionable, repeated, security-relevant, or severe

### Internal invariant or configuration faults

These are server-side defects or invalid process setup:

- impossible state reached
- nil required dependency
- invalid startup configuration
- broken internal assumptions

Handling:

- fail fast at construction/startup when possible
- use panic only for invariant violations that must never continue at runtime
- recover panics only at top-level boundaries
- always log recovered runtime panics
- notify for production-impacting occurrences

---

## 3.5. The Transport Error Primitive: `*web.Error`

`*web.Error` (declared in `pkg/web/errors.go`) is the **canonical
transport-facing application error type**. It carries the HTTP status, a
machine-readable `Code`, a user-safe `Message`, an optional internal `Cause`,
and RFC 7807 problem-document fields (`TypeURI`, `Instance`).

### Fields in use

| Field | Purpose |
|---|---|
| `Status` | HTTP status code (set by `pkg/web` constructors) |
| `Code` | Machine-readable `web.Code` constant (see 12) |
| `Title` | Short human label; never rendered as the user message |
| `Message` | User-safe description; rendered by `PublicMessage()` |
| `Cause` | Internal causal error; **never sent to the client** |
| `Instance` | RFC 7807 instance URI (defaults to request path) |
| `TypeURI` | RFC 7807 type URI |
| `RetryAfter` | Duration hint for `Retry-After` header (0 means omit) |
| `Meta` | Extension members merged into the problem document |

### Rendering and logging rule

`PublicMessage()` is the user-safe accessor used when rendering error responses
to the client. `(*web.Error).Error()` returns `Message` (falling back to
`Title`) and never surfaces the internal `Cause`. **Do not call `.Error()` on a
`*web.Error` in intermediate code** — use `errors.Is` / `errors.As` for
branching and `errors.Unwrap` if you need the causal chain. The boundary itself
uses `fmt.Sprintf("%v", e.Cause)` for the `cause` log attribute.

```go
// ✅ correct — boundary uses Cause directly for structured logs
slog.Error("http.error", "cause", fmt.Sprintf("%v", e.Cause))

// ❌ wrong — call errors.Is/As instead of comparing message strings
slog.Warn("handler: something failed", "err", err.Error())
```

### Construction rule

Always use the constructors in `pkg/web/errors.go`
(`web.ErrBadRequest`, `web.ErrInternal`, `web.ErrConflict`, etc.). Do not
build ad-hoc `&web.Error{...}` structs outside `pkg/web`. If a new
status/code combination is needed, extend `pkg/web/errors.go` per the
stability rules in 12.

### Cross-reference note

The "Application" row in `.claude/rules/r-plan.md`'s Error Classification
table refers to this type from `pkg/web/errors.go`. There is no separate
`apperr` package.

---

## 4. Package Rules

Reusable packages should optimize for explicit behavior and clean composition.

### Packages should

- return ordinary Go errors
- wrap propagated errors with `%w` and enough operation context
- keep sentinel or typed errors near the owning domain
- document errors that callers are expected to branch on
- use `errors.Is` and `errors.As` for branching

### Packages should not

- decide HTTP status codes or render responses
- emit duplicate operational logs for returned failures
- notify external systems directly unless the package explicitly owns a
  background/runtime boundary
- panic for expected runtime failures

### Package-local logging is acceptable only when the package owns the local side effect

Acceptable examples:

- sanitizing unsafe header or cookie data and forcing a safe value
- failure while writing a response body inside an adapter that already owns the
  write path
- best-effort cleanup failure that will not be returned

In those cases:

- keep the message structured
- include the local operation and key identifiers
- avoid logging again higher up unless a separate boundary decision is being made

---

## 4.5. Error Declaration Conventions

These rules govern how errors are declared, named, and formatted. They are the
mechanical complement to the architectural policy in 4.

### Grouping

Declare all package-level sentinel errors in a single `var ( ... )` block per
file. Do not scatter individual `var ErrX = ...` lines across a file.

### Naming

- Use `Err` + noun: `ErrNotFound`, `ErrExpired`, `ErrInvalid`.
- For new or unexported sentinels, do not embed the package name in the
  identifier. The call site already supplies it: `secure.ErrExpired` reads as
  "secure: expired". Adding `Token` to get `secure.ErrTokenExpired` is
  redundant.
- Established exported sentinels that are already part of the public API
  contract (e.g. `secure.ErrTokenInvalid`) are exempt from renaming — the churn
  and breakage cost outweighs the style benefit. Apply this preference to new
  APIs only.
- When two sentinels in the same package would collide without a qualifier (e.g.
  `ErrInvalidToken` vs `ErrInvalidKey`), add the minimal disambiguating noun.

### Message format

Error strings follow the Go standard library convention:

```
"<pkg>: <lowercase, no trailing period>"
```

For sub-operations: `"<pkg>: <op>: <lowercase detail>"`. Consistent prefixes
keep log grep patterns stable and align with `errors.Is` chain display.

### Sentinel vs. inline `fmt.Errorf`

Use a package-level sentinel when any caller must branch on the condition with
`errors.Is` or `errors.As`.

Use inline `fmt.Errorf("<pkg>: <msg>: %w", cause)` when the error is
informational and no caller needs to detect it programmatically — especially
when dynamic context (an index, filename, or ID) must be included.

Do not create a sentinel for every failure mode. Prefer fewer, well-named
sentinels that represent real branching conditions.

### `%w` discipline

Always use `%w` when wrapping a returned error that callers should be able to
unwrap. Never use `%v` or `%s` on an `error` value in a wrapping
`fmt.Errorf` — doing so severs the error chain and breaks `errors.Is`.

Place `%w` at the end of the format string so the prefix reads as a natural
operation trail: `fmt.Errorf("orders: create: %w", err)`.

### Constructor-time validation errors

For invalid constructor arguments:

- if the constructor already returns `(..., error)`, report invalid arguments as
  an ordinary error. This is the default design for new constructors.
- use a sentinel (`ErrEmptySecrets`, `ErrInvalidMode`) when the condition is a
  named failure mode the caller can handle or report uniformly.
- use `fmt.Errorf` with positional or field context
  (`"pkg: Secrets[%d] must not be empty", i`) when the message must carry
  dynamic data that a sentinel cannot express.

Do not mix the two for conditions that are structurally identical.

Panic is reserved for constructor-like APIs that intentionally do not expose a
recoverable error path:

- documented `Must*` constructors or helpers
- assembly-time APIs that fail fast during process wiring
- invariant checks that indicate a programmer error rather than bad runtime
  input

Do not design a new constructor that both returns an error and panics for the
same class of invalid input.

### Const message strings

When the same literal error string would appear in two or more places within a
package, hoist it to a package-level `const` and build all error values from
it. Do not add a const layer for strings that are used exactly once.

### Typed vs. sentinel errors

Use a typed error (`type FooError struct { Field string }`) only when callers
must extract structured data from the error at runtime via `errors.As`. For all
other branching cases — where callers only need to detect the class of failure
— a sentinel is sufficient and preferred.

### `errors.Join` for multi-error accumulation

When a function collects multiple independent failures (e.g. a config
validator), use `errors.Join(errs...)` rather than concatenating strings. This
preserves `errors.Is` compatibility on each joined error and allows callers to
inspect individual failures. Requires Go 1.20+.

---

## 5. App And Service Rules

App-owned feature code sits between reusable packages and transport.

### App/domain code should

- add business-operation context when wrapping errors
- translate known package/domain errors into app semantics where needed
- return errors to the HTTP boundary rather than logging and returning the same
  failure
- keep user-safe copy separate from internal cause text

### App/domain code may log when it owns a non-HTTP boundary

Examples:

- background jobs
- queue consumers
- scheduled tasks
- startup/shutdown lifecycle

If the app code is still inside the HTTP request path, prefer returning and let
the request boundary emit its error event and access log.

---

## 6. HTTP And Runtime Boundary Rules

HTTP and runtime boundaries own classification, correlation, and final
operational signaling.

### HTTP boundaries should

- recover panics
- classify returned errors into transport-safe categories
- emit a structured error event and a separate access log (see below)
- render user-safe responses only
- preserve correlation identifiers such as request ID

Current reference implementation:

- `pkg/web/boundary.go`
- `pkg/web/errors.go`
- `pkg/web/serve/logging.go`

### Boundary two-signal design

The HTTP boundary intentionally emits two log entries per request. These are
not duplicates — they serve different operational purposes and both must be
preserved:

- **Structured error event** (`http.error`): captures the causal error,
  request ID, status code, and error code. Used for alerting and trace
  correlation. Emitted by `pkg/web/boundary.go`.
- **Access log** (`http.request`): captures method, path, latency, and status
  for every request. Used for traffic analysis and audit. Emitted by
  `pkg/web/serve/logging.go`.

The "do not log at every stack frame" rule in 4 applies to *intermediate*
layers (packages, services, handlers). It means: do not log a returned error
at each call site before the boundary sees it. It does not restrict the
boundary itself from emitting both signals.

### Avoid double-logging recovered panics

The boundary's panic-recovery path already emits a single `http.error` event
with the panic value and stack trace. An injected `OnPanic` hook must not
re-emit the same information at the same log level. The hook's purpose is
side-channel notification, metric bump, or alert fan-out — not a second
log entry for the same event. If the hook logs at all, use a separate
event name (e.g. `"panic.notify"`) so the two signals remain distinct in log
aggregation.

### Do not call `.Error()` on a `*web.Error` in intermediate code

Intermediate handlers and packages must not call `.Error()` on an error value
known to be a `*web.Error`. Use `errors.Is` or `errors.As` for branching; let
the boundary own cause rendering. See 3.5 for detail on why this leaks.

### Boundary log requirements

When available, include:

- operation or event name
- package/component or subsystem
- request ID or correlation ID
- status and machine-readable code at transport edges
- causal error
- latency, method, and path for request logs

For the canonical field list, see 6.2.

### Severity guidance

- `INFO`: successful requests and normal lifecycle events
- `WARN`: client-caused failures, safe degradations, or notable-but-contained
  behavior
- `ERROR`: server-side failures, dependency failures, recovered panics, failed
  writes, and unmet invariants

Do not use log level as a substitute for notification policy.

### Security requirements at boundaries

**Stack traces must never reach the client.** Recovered panics capture the
stack with `runtime/debug.Stack()` for the server log. The client response
contains only a generic 5xx body.

**Error messages must not leak internal paths, SQL, schema, or file-system
structure.** Wrap internal cause text in a `*web.Error` (or equivalent) before
responding. The boundary renders the user-safe `Message`; the server retains the
full error tree for inspection via `errors.Is`, `errors.As`, and explicit
traversal where joined or multi-wrapped errors are used.

**Do not embed untrusted input verbatim in log messages.** Use `slog` structured
attributes (`slog.String("path", p)`) rather than `fmt.Sprintf` string
interpolation. Embedded newlines or control characters in user input can inject
spurious log entries.

**Redact or categorise PII before logging.** Log field names and categories
(e.g. `"email: present"`, `"path: /user/42/avatar"`), not raw email addresses,
passwords, or user-supplied filenames.

**Auth-sensitive error paths must be constant-time up to the final comparison.**
When an early-exit could reveal whether a secret matched, normalize inputs as
far as needed to obtain a candidate byte sequence, then run MAC or signature
verification before semantic checks such as expiry, scope, or claims
validation. Structural parse failures may still reject early; the constant-time
requirement applies once the input is in a form that can be verified.
`pkg/web/secure/token.go` is the reference implementation for "verify MAC
before expiry" within that constraint.

**`context.Canceled` (client disconnect) is not a server fault.** When the
client cancels the request, treat the resulting error as a non-fault event: do
not log at ERROR and do not notify. Classify using
`errors.Is(err, context.Canceled)`.

**`context.DeadlineExceeded` is ownership-dependent.** A deadline set by the
*server* against an upstream or database call represents a dependency timeout
— classify as 504, log at ERROR, and consider notification if persistent. A
deadline that originated from the *client* request context is non-fault and
should be treated like `context.Canceled`.

### Deadline provenance contract

The boundary cannot infer deadline ownership from bare `context.DeadlineExceeded`
alone. To make the distinction implementable, code that sets a server-owned
deadline must mark the resulting timeout before returning it.

Standard contract:

- if a package or service applies its own timeout to an upstream call, DB call,
  queue operation, or filesystem operation, it must not return bare
  `context.DeadlineExceeded`
- instead, it must wrap that timeout in a package-owned sentinel or typed error
  that marks it as a server-owned dependency timeout
- bare `context.DeadlineExceeded` reaching the HTTP boundary is treated as
  request-context timeout/cancellation unless the boundary has explicit timeout
  ownership metadata from the caller

Recommended forms:

- sentinel marker for branch-only cases: `web.ErrDependencyTimeout`
  (declared in `pkg/web/errors.go`)
- typed error when callers need metadata, e.g. dependency name or timeout
  duration

Boundary mapping rule:

- `errors.Is(err, context.Canceled)` -> non-fault client cancellation path
- `errors.Is(err, ErrDependencyTimeout)` or equivalent typed timeout marker ->
  504, `ERROR`, and notification consideration per 8
- bare `context.DeadlineExceeded` -> non-fault request-context timeout unless a
  higher boundary explicitly documented otherwise owns that deadline

---

## 6.1. Stack Traces for Non-Panic 5xx Errors

Recovered panics already capture and log a stack trace via
`runtime/debug.Stack()`. Non-panic 5xx errors can also carry a stack when the
`BoundaryConfig.CaptureStack` flag is `true`. When enabled, the stack is
included in the `http.error` log event under a `stack` attribute and is never
forwarded to the client. Disable it in latency-sensitive deployments to avoid
the per-request overhead.

---

## 6.2. Structured Error Event Schema

Every boundary (HTTP, job, queue, goroutine) must emit error events with a
consistent field set. Names are the stable contract; log aggregators and alert
rules must be able to pivot on them.

| Field | Type | When present | Notes |
|---|---|---|---|
| `event` | string | always | `"http.error"` for HTTP; `"job.error"`, `"queue.error"`, `"lifecycle.error"`, `"panic.goroutine"` for other boundaries |
| `severity` | log level | always | `WARN` for <500 and for 499/client-deadline; `ERROR` for ≥500 and recovered panics |
| `request_id` | string | when available | from `web.RequestID(c)` |
| `trace_id` | string | when tracing enabled | from OTel span context; see 6.3 |
| `span_id` | string | when tracing enabled | from OTel span context |
| `status` | int | HTTP only | HTTP response status code |
| `code` | string | HTTP only | `web.Code` constant value |
| `op` | string | always | operation name, e.g. `"orders.create"`, `"session.load"` |
| `subsystem` | string | always | package or component, e.g. `"session"`, `"ratelimit"`, `"file"` |
| `cause` | string | 5xx and server-owned timeouts | `fmt.Sprintf("%v", e.Cause)` — omit for client cancellation |
| `stack` | string | recovered panics; 5xx when enabled | `debug.Stack()` output — never forward to client |
| `dedupe_key` | string | notify-worthy events | stable string key for alert deduplication; see 8 |

Non-HTTP boundaries (jobs, queue consumers, background goroutines, startup)
use the same schema. Omit `status` and `code` when not applicable.

---

## 6.3. Tracing and Correlation (OpenTelemetry)

This section is guidance; the guide does not mandate a specific OTel SDK
version or distribution. Rules apply when a tracer is installed.

**Attach trace and span IDs to every boundary event.** When an OTel tracer is
active, extract `trace_id` and `span_id` from the current span context and
include them in the structured event (6.2) alongside `request_id`. Log
aggregators and tracing backends become pivotable on the same failure.

**Record errors on the active span.** On 5xx and recovered panics, set the
active span's status to `codes.Error` and call `span.RecordError(err)` before
the boundary returns the response. This surfaces the error in trace waterfall
views without requiring a separate query.

**Use OTel HTTP semantic conventions.** When emitting spans for HTTP server
requests, use the standard attribute names:

- `http.request.method`
- `http.response.status_code`
- `error.type` → set to the string value of the `web.Code` constant

The `code` field in 6.2 mirrors `error.type`, keeping log and trace backends
pivotable on the same identifier.

**Propagation: `request_id` vs `trace_id`.** These serve different audiences:

- `request_id` is the user-facing correlator — safe to include in error
  responses as a support reference; propagated via a custom header across
  service boundaries.
- `trace_id` is the tracing-backend correlator — used by engineers; not
  included in client-facing responses.

Always emit both where available. Do not replace one with the other.

The reference integration lives in `pkg/web/otelweb` — `otelweb.Middleware`
starts the server span, and `otelweb.ExtractTraceID` / `otelweb.ExtractSpanID`
supply the `BoundaryConfig` extractors. `otelweb.MakeOnError()` provides the
`BoundaryConfig.OnError` hook that calls `span.RecordError` and sets
`codes.Error` on 5xx responses.

---

## 7. Panic Policy

### Panic is allowed for

- impossible internal states
- documented `Must*` constructors and fail-fast assembly helpers
- duplicate route or invalid route registration at assembly time
- explicit `Must*` APIs whose contract is documented as panic-on-failure

### Panic is not allowed for

- malformed request input
- dependency outages
- validation failures
- missing records
- any runtime condition a caller can reasonably handle
- crossing a package boundary — a panic must never propagate out of a package
  to its caller; recover before returning if a package can panic internally

### Recovery rules

- recover at top-level boundaries only
- attach stack traces there
- convert the failure into a safe 5xx response or runtime failure signal
- recovery exists to convert a crash into a logged, classified error at a
  boundary — not to silently suppress failures and continue as if nothing happened

---

## 7a. Goroutine Panics

A panic inside a goroutine crashes the entire process unless a `recover` is
installed within that goroutine. The top-level HTTP boundary recovery does not
extend into goroutines started by package code.

### Rules

- Any goroutine started by a package must install its own `recover`.
- On recovery, log the panic and stack trace at ERROR, then either exit the
  goroutine cleanly or signal the failure via a channel or `errgroup.Group`.
- Never silently swallow a goroutine panic — at minimum log it.
- `errgroup` propagates *returned errors* from goroutines — it does not recover
  panics. An unrecovered panic in an errgroup goroutine still crashes the
  process. Install a `recover` within each goroutine regardless of whether
  errgroup is used. Prefer `errgroup` for fan-out error collection, but treat
  panic recovery as a separate, mandatory concern.

### Event naming for background boundaries

Use consistent event names in structured log output so aggregators can pivot
on source boundary:

| Boundary | Event name |
|---|---|
| HTTP boundary (panic or classified error) | `http.error` |
| Scheduled job | `job.<name>.error` |
| Queue consumer | `queue.<topic>.error` |
| Startup / shutdown lifecycle | `lifecycle.error` |
| Recovered goroutine panic (non-HTTP) | `panic.goroutine` |

Background error events must include at minimum `event`, `subsystem`, `cause`,
and `stack` (for recovered panics). Add `op` and `dedupe_key` where applicable
per the 6.2 schema.

---

## 8. Notification Policy

Notification is reserved for actionable conditions. Logging alone is the
default.

### Notify when the event is

- production-impacting and server-side
- likely to require operator or developer action
- a repeated dependency failure or outage indicator
- a recovered panic
- a security-relevant event
- a data-loss, corruption, or consistency risk

### Do not notify for

- routine 4xx traffic
- expected domain errors
- one-off user mistakes
- failures already visible and self-healing without operator action
- `context.Canceled` from a client disconnect (see 6)
- `context.DeadlineExceeded` that originated from a client-set deadline (see 6)

### Required signal for future notifier hooks

Any future notifier integration should be able to carry:

- severity
- stable event or dedupe key
- operation/subsystem name
- environment
- correlation or request ID when available
- user-safe summary
- internal cause details for operators

This guide does not choose a vendor. It defines the threshold and the required
signal.

---

## 9. Decision Table

| Situation | Return | Log | Notify | Panic |
|----------|--------|-----|--------|-------|
| Validation or not-found outcome | Yes | No | No | No |
| Malformed request or unsupported input | Yes | Usually no | No | No |
| Dependency timeout or failed I/O | Yes | Yes, once at the boundary | Maybe, if actionable | No |
| Best-effort sanitization/degradation | Maybe | Yes, local warn if not returned | No | No |
| Startup misconfiguration | Fail startup | Yes | Maybe in deployed env | Sometimes |
| Recovered runtime panic | Converted at boundary | Yes, error with stack | Yes | Originating code may panic |
| Client context cancelled (`context.Canceled`) | Yes | No (or DEBUG) | No | No |
| Client-set deadline exceeded | Yes | No (or DEBUG) | No | No |
| Server-set deadline exceeded (upstream/DB) | Yes | Yes, at boundary (ERROR) | Maybe, if persistent | No |

---

## 10. Do This, Not That

### Return contextual errors from packages

Prefer:

```go
if err := store.Save(ctx, record); err != nil {
	return fmt.Errorf("orders: save order: %w", err)
}
```

Avoid:

```go
if err := store.Save(ctx, record); err != nil {
	slog.Error("save failed", "err", err)
	return err
}
```

### Map to transport errors at the boundary

Prefer:

```go
if errors.Is(err, domain.ErrConflict) {
	return web.Response{}, web.ErrConflict("order already exists")
}
return web.Response{}, fmt.Errorf("orders: create: %w", err)
```

Avoid returning raw HTTP concerns from deep package code.

### Do not log at every stack frame

The boundary emits its own structured error event and access log (6). Do not
also log in intermediate code — that creates redundant, confusing entries for
the same failure.

Avoid:

- package logs the error *and* returns it to the handler
- handler logs the error *and* returns it to the boundary
- boundary logs it again after an intermediate layer already did

### Warn for safe degradation, not for normal control flow

Prefer a warning when code forces a safe fallback or strips unsafe input.

Avoid warning on every expected 4xx-style branch.

### Use `errors.Is` and `errors.As` for branching — never `==`

Prefer:

```go
if errors.Is(err, ErrNotFound) { ... }
```

Avoid:

```go
if err == ErrNotFound { ... }  // breaks when err is wrapped
```

Use `errors.As` when you need to extract data from a typed error:

```go
var ferr *FieldError
if errors.As(err, &ferr) {
	// use ferr.Field
}
```

### Use `errors.Join` for accumulated validation failures

Prefer:

```go
var errs []error
if v.Name == "" {
	errs = append(errs, ErrNameRequired)
}
if v.Age < 0 {
	errs = append(errs, ErrAgeInvalid)
}
return errors.Join(errs...)
```

Avoid building a multi-line string manually — this severs `errors.Is`
compatibility on each individual failure.

### Group sentinel declarations; match message format

Prefer (one block, standard message format):

```go
var (
	ErrExpired = errors.New("secure: expired token")
	ErrInvalid = errors.New("secure: invalid token")
)
```

Avoid:

```go
var ErrTokenExpired = errors.New("Token Expired")   // wrong name, wrong format
var ErrTokenInvalid = errors.New("secure: invalid token")  // redundant noun
```

---

## 11. Retry And Transience

Not all dependency failures are equal. The boundary that decides whether to
retry must be able to distinguish transient from permanent failures.

### Convention

Use `errors.Join(web.ErrTransient, err)` to attach the transience marker
alongside the causal error. `web.ErrTransient` is declared in
`pkg/web/errors.go`. This produces a flat, readable error tree that reviewers
can follow without reasoning about multi-wrap ordering, and preserves full
`errors.Is` / `errors.As` compatibility on both values:

```go
// usage: wrap with context first, then join the marker
cause := fmt.Errorf("cache: get: %w", err)
return errors.Join(web.ErrTransient, cause)

// caller:
if errors.Is(err, web.ErrTransient) {
	// safe to retry — causal detail is still unwrappable
}
```

When callers need to extract structured metadata (e.g. retry delay, attempt
count), use a typed error instead of a sentinel and surface it via
`errors.As`:

```go
type TransientError struct {
	Cause      error
	RetryAfter time.Duration
}

func (e *TransientError) Error() string  { return e.Cause.Error() }
func (e *TransientError) Unwrap() error  { return e.Cause }

// caller:
var te *cache.TransientError
if errors.As(err, &te) {
	time.Sleep(te.RetryAfter)
}
```

Do not use `fmt.Errorf("...: %w: %w", ErrTransient, err)` — double-wrapping
with `%w` produces an error tree that is harder to read in reviews and reason
about in callers. Prefer the explicit forms above.

### Rules

- Only the package that owns the dependency classifies it as transient.
- Do not classify a failure as transient unless a retry at the same boundary is
  both safe and likely to succeed.
- `context.Canceled` is not transient — do not retry it.
- Idempotent operations may retry on `ErrTransient`; non-idempotent operations
  must not retry without explicit idempotency handling (see 11.2).

---

## 11.1. Backoff and Jitter

Fixed retry delays synchronize clients and amplify dependency outages.
Transient retries must use **exponential backoff with full jitter**:

```
delay = random_between(0, min(cap, base * 2^attempt))
```

Rules:

- Cap total attempts at a small number (≤3 in most cases).
- Cap cumulative delay against the remaining context deadline; never retry past
  the caller's deadline — check `ctx.Err()` before each attempt.
- On each retry, propagate the original `context` unchanged; do not create a new
  deadline inside the retry loop.

The reference implementation is `retry.Do(ctx, retry.Policy{...}, fn)` in
`pkg/web/retry`. It implements full-jitter exponential backoff and accepts an
optional `retry.BudgetChecker` to gate retries against a `retrybudget.Budget`.

---

## 11.2. Idempotency

Non-idempotent operations (create, payment capture, send-email, external
webhook dispatch) must not be retried unless an idempotency key is in place.

### Rules

- Before initiating a retryable non-idempotent operation, generate or accept an
  **idempotency key** — a stable identifier for this logical request (e.g. a
  UUID derived from the client-supplied request ID, or a deterministic hash of
  the immutable request parameters).
- The server must deduplicate by key using a store that outlives process
  restarts (a database row or a persistent cache entry). In-memory maps are
  not sufficient — they disappear on restart and allow double-execution after
  failover.
- Returning a cached response for a duplicate key is the correct behavior; do
  not re-execute the operation.
- The idempotency key's TTL must exceed the maximum retry window plus any
  observable clock skew between services.

Retrying a non-idempotent operation without a key is a correctness bug — reject
it at code review regardless of retry policy.

---

## 11.3. Retry Budget

Unlimited retries during a partial outage amplify load and turn a dependency
degradation into a full outage (retry storm).

### Rules

- Retries share a **global per-upstream budget**: a sliding-window token
  counter that limits the retry rate regardless of how many concurrent
  callers exist.
- When the budget is exhausted, fail fast with `ErrTransient` (or open a
  circuit breaker, 11a) rather than continuing to queue retries.
- Budget exhaustion is a **notification-worthy event** (8) — it indicates
  a dependency is sustaining elevated failure rates.
- Expose budget state in structured logs (`subsystem`, `op`, `budget_remaining`
  or equivalent) so operations teams can observe degradation before it becomes
  complete unavailability.

The reference implementation is `retrybudget.Budget` in `pkg/web/retrybudget`.
Wire it into `retry.Policy.Budget` to limit total retry throughput per upstream.

---

## 11a. Circuit Breakers and Bulkheads

Circuit breakers convert sustained transient failures into fast-fail responses,
protecting both the client and the failing upstream.

### Circuit breaker rules

- Open the breaker for a specific upstream when transient failure rate exceeds a
  configured **threshold** within a **time window** (e.g. 5 failures in 10
  seconds).
- While open, reject immediately with `errors.Join(web.ErrTransient, breaker.ErrBreakerOpen)`
  which classifies to 503 with a bounded `Retry-After` header via
  `web.ErrBreakerOpenResponse(retryAfter)`.
- After the breaker's **recovery window** expires, move to half-open: allow one
  probe request. On success, close the breaker; on failure, reset the recovery
  window and remain open.
- Breaker-open events are **notification-worthy** (8) as repeated-dependency
  indicators. Breaker-closed (recovery) events log at INFO and do not notify.

### Bulkhead rules

- Assign a **dedicated resource pool** (connection pool, worker pool, semaphore)
  per upstream or resource type. Do not share a single global pool across all
  dependencies.
- When a pool is exhausted, fail fast with a transient error rather than
  blocking indefinitely. Apply the deadline provenance contract (6) if the
  exhaustion is measured against a server-set timeout.
- Pool exhaustion is a signal worth metering (metric increment + log at WARN).
  Sustained exhaustion is notification-worthy.

### Error taxonomy

`web.ErrBreakerOpen` and pool-exhaustion errors are both classified as 503
with `CodeUnavailable` at the HTTP boundary. Include the dependency name
(upstream, store name) in the structured log event under `subsystem`.

Reference implementations: `pkg/web/breaker` (circuit breaker) and
`pkg/web/bulkhead` (semaphore pool).

---

## 12. Error Codes

`pkg/web/errors.go` defines `Code` string constants (`CodeInternal`,
`CodeForbidden`, `CodeNotFound`, ...) included in structured API error
responses. These codes are the stable, machine-readable contract for API clients.

### Stability rules

- Codes are **additive-only**: new codes may be added in any release.
- Codes are **never renamed or removed**: a client that branches on a code must
  not break after a server update.
- Codes are **never reused** for a different meaning.
- Document the condition and HTTP status mapping for every new code at the point
  of declaration.

### Usage

- Only boundaries (`pkg/web/errors.go` constructors and handler-layer mapping)
  assign codes. Package internals return plain or sentinel errors.
- Codes do not appear in log messages — they are transport-layer identifiers.

---

## 13. Test Policy

Every exported sentinel must have a test that:

1. triggers the documented condition, and
2. asserts the returned error satisfies `errors.Is(err, ErrX)`.

Every error path in a handler must have a dedicated test case (see
[`r-test.md`](../.claude/rules/r-test.md) Error Path Coverage).

Do not assert error identity by string matching — always use `errors.Is` or
`errors.As`.

---

## 14. Review Checklist

When adding or refactoring behavior, confirm:

- the package returns errors instead of logging-and-returning the same failure
- wrapped errors identify the failed operation
- typed or sentinel errors are available where callers need branching
- HTTP status mapping happens at the handler or boundary layer
- constructors that return an error report invalid arguments via error; panic is
  reserved for `Must*`, assembly-time, or invariant-only APIs
- intermediate layers (packages, services, handlers) do not log returned errors; the boundary emits the error event and access log
- notification thresholds distinguish actionable server-side faults from normal
  bad input
- sentinel errors are declared in a single `var ( ... )` block
- error message strings follow the `"<pkg>: <lowercase detail>"` convention
- `%w` is used when wrapping a returned error (never `%v` or `%s`)
- no stack traces or internal paths appear in client-facing responses
- no PII or untrusted input is embedded verbatim in log messages
- transient dependency failures are marked with `web.ErrTransient` where callers
  may retry
- server-owned deadlines are marked with `web.ErrDependencyTimeout` before they
  reach the boundary
- `*web.Error.Error()` is not called in intermediate code outside the boundary (see 3.5)
- recovered panics are not double-logged between the boundary and an `OnPanic` hook (see 6)
- structured error events conform to the 6.2 field schema
- when a tracer is installed, `trace_id` and `span_id` are included in structured events (see 6.3)

---

## 15. Sources And Current References

- `pkg/web/errors.go` — sentinels, `*web.Error`, `Classify`, constructors
- `pkg/web/boundary.go` — `ErrorBoundary`, `BoundaryConfig`
- `pkg/web/serve/logging.go` — access log (`http.request`)
- `pkg/web/retry` — `retry.Do`, `retry.Policy` (11.1)
- `pkg/web/breaker` — `breaker.Breaker` (11a)
- `pkg/web/bulkhead` — `bulkhead.Bulkhead` (11a)
- `pkg/web/retrybudget` — `retrybudget.Budget` (11.3)
- `pkg/web/idempotency` — `idempotency.Middleware`, `idempotency.Store` (11.2)
- `pkg/web/otelweb` — `otelweb.Middleware`, `otelweb.MakeOnError` (6.3)
- [`UI_GUIDE.md`](./UI_GUIDE.md) — error page rendering contract
- Go Code Review Comments: <https://go.dev/wiki/CodeReviewComments>
- Effective Go: <https://go.dev/doc/effective_go>
