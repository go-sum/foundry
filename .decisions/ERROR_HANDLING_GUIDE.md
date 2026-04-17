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
> [`DESIGN_GUIDE.md`](./DESIGN_GUIDE.md), and
> [`PATTERNS_PRINCIPLES.md`](./PATTERNS_PRINCIPLES.md).
>
> Use this guide to answer:
>
> - what should be returned versus logged
> - where errors should be classified and rendered
> - when panic is appropriate
> - which failures are notification-worthy

---

## 1. Purpose

This repository already has good building blocks for structured handling:

- `pkg/web/errors.go` defines transport-facing error categories
- `pkg/web/boundary.go` defines HTTP boundary classification, panic recovery,
  and rendering
- `pkg/web/adapt/logging.go` defines a baseline request log

What has been inconsistent is the policy above those primitives. Some code
returns wrapped errors, some code logs directly, some code warns on
degradation, and some construction-time paths still panic.

This guide standardizes that policy.

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
mechanical complement to the architectural policy in §4.

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

### Constructor-time validation errors

For invalid constructor arguments:

- use a sentinel (`ErrEmptySecrets`, `ErrInvalidMode`) when the condition is a
  named failure mode the caller can handle or report uniformly.
- use `fmt.Errorf` with positional or field context
  (`"pkg: Secrets[%d] must not be empty", i`) when the message must carry
  dynamic data that a sentinel cannot express.

Do not mix the two for conditions that are structurally identical.

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
- `pkg/web/adapt/logging.go`

### Boundary two-signal design

The HTTP boundary intentionally emits two log entries per request. These are
not duplicates — they serve different operational purposes and both must be
preserved:

- **Structured error event** (`http.error`): captures the causal error,
  request ID, status code, and error code. Used for alerting and trace
  correlation. Emitted by `pkg/web/boundary.go`.
- **Access log** (`http.request`): captures method, path, latency, and status
  for every request. Used for traffic analysis and audit. Emitted by
  `pkg/web/adapt/logging.go`.

The "do not log at every stack frame" rule in §4 applies to *intermediate*
layers (packages, services, handlers). It means: do not log a returned error
at each call site before the boundary sees it. It does not restrict the
boundary itself from emitting both signals.

### Boundary log requirements

When available, include:

- operation or event name
- package/component or subsystem
- request ID or correlation ID
- status and machine-readable code at transport edges
- causal error
- latency, method, and path for request logs

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
responding. The boundary renders the user-safe `Message`; the full error chain
is available only in the server log via `Unwrap()`.

**Do not embed untrusted input verbatim in log messages.** Use `slog` structured
attributes (`slog.String("path", p)`) rather than `fmt.Sprintf` string
interpolation. Embedded newlines or control characters in user input can inject
spurious log entries.

**Redact or categorise PII before logging.** Log field names and categories
(e.g. `"email: present"`, `"path: /user/42/avatar"`), not raw email addresses,
passwords, or user-supplied filenames.

**Auth-sensitive error paths must be constant-time up to the final comparison.**
When an early-exit could reveal whether a secret matched, rearrange branches so
MAC or signature verification runs unconditionally before expiry or format
checks. `pkg/web/secure/token.go` is the reference implementation of this
pattern.

**`context.Canceled` (client disconnect) is not a server fault.** When the
client cancels the request, treat the resulting error as a non-fault event: do
not log at ERROR and do not notify. Classify using
`errors.Is(err, context.Canceled)`.

**`context.DeadlineExceeded` is ownership-dependent.** A deadline set by the
*server* against an upstream or database call represents a dependency timeout
— classify as 504, log at ERROR, and consider notification if persistent. A
deadline that originated from the *client* request context is non-fault and
should be treated like `context.Canceled`. Use the classification in
`pkg/web/errors.go` as the reference implementation for distinguishing these
at the boundary.

---

## 7. Panic Policy

### Panic is allowed for

- impossible internal states
- invalid constructor arguments that make the type unusable
- duplicate route or invalid route registration at assembly time
- explicit `Must*` APIs whose contract is documented as panic-on-failure

### Panic is not allowed for

- malformed request input
- dependency outages
- validation failures
- missing records
- any runtime condition a caller can reasonably handle

### Recovery rules

- recover at top-level boundaries only
- attach stack traces there
- convert the failure into a safe 5xx response or runtime failure signal
- do not continue from a panic in lower-level package code as if nothing happened

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
- `context.Canceled` from a client disconnect (see §6)
- `context.DeadlineExceeded` that originated from a client-set deadline (see §6)

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

The boundary emits its own structured error event and access log (§6). Do not
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

Use `errors.Join(ErrTransient, err)` to attach the transience marker alongside
the causal error. This produces a flat, readable error tree that reviewers can
follow without reasoning about multi-wrap ordering, and preserves full
`errors.Is` / `errors.As` compatibility on both values:

```go
// infrastructure package:
var ErrTransient = errors.New("transient failure")

// usage: wrap with context first, then join the marker
cause := fmt.Errorf("cache: get: %w", err)
return errors.Join(ErrTransient, cause)

// caller:
if errors.Is(err, cache.ErrTransient) {
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
  must not retry without explicit idempotency handling.

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
[`r-test.md`](../.claude/rules/r-test.md) §Error Path Coverage).

Do not assert error identity by string matching — always use `errors.Is` or
`errors.As`.

---

## 14. Review Checklist

When adding or refactoring behavior, confirm:

- the package returns errors instead of logging-and-returning the same failure
- wrapped errors identify the failed operation
- typed or sentinel errors are available where callers need branching
- HTTP status mapping happens at the handler or boundary layer
- panic is used only for invariants or documented `Must*` contracts
- intermediate layers (packages, services, handlers) do not log returned errors; the boundary emits the error event and access log
- notification thresholds distinguish actionable server-side faults from normal
  bad input
- sentinel errors are declared in a single `var ( ... )` block
- error message strings follow the `"<pkg>: <lowercase detail>"` convention
- `%w` is used when wrapping a returned error (never `%v` or `%s`)
- no stack traces or internal paths appear in client-facing responses
- no PII or untrusted input is embedded verbatim in log messages
- transient dependency failures are marked with `ErrTransient` where callers
  may retry

---

## 15. Sources And Current References

- `pkg/web/errors.go`
- `pkg/web/boundary.go`
- `pkg/web/adapt/logging.go`
- Go Code Review Comments: <https://go.dev/wiki/CodeReviewComments>
- Effective Go: <https://go.dev/doc/effective_go>
