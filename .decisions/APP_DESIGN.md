---
title: Application Design Patterns
description: Governing patterns for handler, service, middleware, and testing layers in Go web applications.
weight: 22
---

# Application Design Patterns

> This guide is the authoritative source for how Go web application code should
> be designed at the handler, service, middleware, context, logging, and testing
> layers.
>
> It complements [`DESIGN_GUIDE.md`](./DESIGN_GUIDE.md), which defines
> architecture and ownership, [`ERROR_HANDLING_GUIDE.md`](./ERROR_HANDLING_GUIDE.md),
> which defines error classification and boundary behavior, and
> [`PATTERNS_PRINCIPLES.md`](./PATTERNS_PRINCIPLES.md), which defines function,
> type, and package structure.
>
> This guide is derived from:
>
> - production Go web patterns for handler, middleware, and service design
> - Go standard library idioms (`net/http`, `log/slog`, `context`, `httptest`)
> - `go-playground/validator` for structured input validation
> - successful patterns already working in this codebase today

---

## 1. Five Rules of Production Go

These rules are non-negotiable for any production Go web application. They
govern the fundamental shape of all application code.

### Rule 1: Zero Global State

All handlers are methods on structs. No package-level `var` holds mutable state.

**Allowed at package level:**

- constants
- pure functions (no side effects, no shared state)
- sentinel errors (`var ErrNotFound = errors.New(...)`)
- stateless validator instances
- enum-to-value maps (immutable after init)

**Forbidden at package level:**

- database connections or pools
- loggers
- HTTP clients
- config structs
- caches
- rate limiters
- any mutable state shared across requests

Every dependency a handler needs is injected through its struct or constructor:

```go
type UserHandler struct {
    svc    UserService
    logger *slog.Logger
}

func NewUserHandler(svc UserService, logger *slog.Logger) *UserHandler {
    return &UserHandler{svc: svc, logger: logger}
}
```

This makes the dependency graph explicit, testable, and free of hidden
coupling between packages.

### Rule 2: Explicit Error Handling

Every error is checked. Every propagated error is wrapped with context using
`fmt.Errorf`:

```go
user, err := s.repo.GetByID(ctx, id)
if err != nil {
    return User{}, fmt.Errorf("getting user %s: %w", id, err)
}
```

Wrapping convention: `"verbing noun: %w"` -- lowercase, no trailing period.

For HTTP APIs, use a structured application error type that carries status,
code, and a user-safe message. See
[`ERROR_HANDLING_GUIDE.md`](./ERROR_HANDLING_GUIDE.md) section 3.5 for the
canonical `*web.Error` type.

Never discard an error with `_ =`. If the error genuinely cannot be handled,
document why with a comment.

### Rule 3: Validation First

Validate at the boundary. Trust internal data.

- Handlers validate all external input (query params, form fields, JSON bodies,
  path parameters) before calling services.
- Services trust that their inputs have been validated. They enforce business
  rules (uniqueness, state transitions, authorization) but do not re-validate
  structural constraints.
- Repositories trust that their inputs are well-typed domain values.

Never validate the same constraint twice across layers. The boundary owns
structural validation; the service owns semantic validation.

### Rule 4: Testability

Every handler has a `_test.go` file. Tests use `httptest` with table-driven
patterns covering both happy paths and error paths.

A handler that cannot be tested with `httptest.NewRequest` and
`httptest.NewRecorder` has a design problem -- fix the design, not the test
approach.

### Rule 5: Documentation

Every exported symbol has a Go doc comment starting with its name:

```go
// UserHandler handles HTTP requests for user operations.
type UserHandler struct { ... }

// GetByID returns a user by their unique identifier.
func (h *UserHandler) GetByID(c echo.Context) error { ... }
```

Comments describe *what* and *why*, not *how*. Implementation details belong in
the code, not in doc comments.

---

## 2. Middleware Architecture

### Standard Signature

Middleware follows the standard `func(http.Handler) http.Handler` pattern:

```go
func RequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := uuid.NewString()
        ctx := context.WithValue(r.Context(), requestIDKey, id)
        w.Header().Set("X-Request-ID", id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

Key points:

- Accept a `http.Handler`, return a `http.Handler`.
- Call `next.ServeHTTP` to continue the chain. Forgetting this silently drops
  the request.
- Use `r.WithContext(ctx)` for context propagation -- never mutate the original
  request.
- Write response headers *before* calling `next.ServeHTTP` if the header must
  be present regardless of downstream behavior.

### Middleware Chain Ordering

Order matters. The outermost middleware runs first and wraps all inner behavior.

Canonical ordering (outermost to innermost):

1. **Recovery** -- catches panics from all downstream handlers and middleware
2. **Request ID** -- assigns a correlation identifier before any logging occurs
3. **Logger** -- wraps the response to capture status and duration
4. **Security** -- CORS, CSRF, rate limiting, origin checking
5. **Auth** -- validates credentials and populates user context
6. **Application-specific** -- feature flags, tenant resolution, caching headers

### Chain Composition

Compose middleware with a helper that applies them in declaration order:

```go
// Chain applies middleware in the order given. The first argument
// is the outermost middleware (runs first).
func Chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
    for i := len(mw) - 1; i >= 0; i-- {
        h = mw[i](h)
    }
    return h
}

// Usage:
handler := Chain(
    appHandler,
    Recovery,
    RequestID,
    Logger(logger),
    Auth(tokenVerifier),
)
```

The reverse iteration ensures the first middleware in the slice is the outermost
wrapper, matching the mental model of "Recovery runs first."

---

## 3. Context Propagation

### Type-Safe Context Keys

Always use an unexported type for context keys. Never use `string` or `int`
directly -- they collide across packages:

```go
// unexported type prevents collisions with other packages
type contextKey int

const (
    requestIDKey contextKey = iota
    userKey
    tenantKey
)
```

### Request ID Propagation

Assign the request ID as early as possible (outermost middleware). Include it in:

- all structured log entries for the request
- all outgoing HTTP requests via headers
- error responses as a support reference

```go
func RequestIDFrom(ctx context.Context) string {
    id, _ := ctx.Value(requestIDKey).(string)
    return id
}
```

Always check the type assertion with the comma-ok pattern. A missing or
wrong-typed context value must not panic.

### User Metadata in Context

After auth middleware validates credentials, store the authenticated user
identity in context:

```go
type User struct {
    ID       uuid.UUID
    Email    string
    Role     string
    TenantID uuid.UUID
}

func UserFrom(ctx context.Context) (User, bool) {
    u, ok := ctx.Value(userKey).(User)
    return u, ok
}

func MustUserFrom(ctx context.Context) User {
    u, ok := UserFrom(ctx)
    if !ok {
        panic("user not in context -- auth middleware missing")
    }
    return u
}
```

`MustUserFrom` is acceptable only in code paths guaranteed to run behind auth
middleware. In all other paths, use the two-return form and handle the missing
case explicitly.

### Multi-Tenant Context

For multi-tenant applications, resolve the tenant early (after auth) and
propagate via context:

```go
func TenantFrom(ctx context.Context) (uuid.UUID, bool) {
    id, ok := ctx.Value(tenantKey).(uuid.UUID)
    return id, ok
}
```

Repositories and services receive the tenant ID from context, never from global
state or ambient configuration.

### Passing Context Downstream

Context flows through every layer:

- **Database queries**: use context-accepting methods
  (`pool.Query(ctx, ...)`, `tx.Exec(ctx, ...)`)
- **Outgoing HTTP**: use `http.NewRequestWithContext(ctx, ...)`
- **Service calls**: accept `context.Context` as the first parameter

Never use `context.Background()` in services or repositories. It severs
cancellation propagation and timeout enforcement.

### Context Timeout and Cancellation

Use `context.WithTimeout` for operations with bounded latency expectations:

```go
func (s *OrderService) Create(ctx context.Context, input OrderInput) (Order, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    return s.repo.Insert(ctx, input)
}
```

For HTTP-level timeouts, prefer timeout middleware that wraps the entire request
lifecycle rather than per-operation timeouts in every handler.

---

## 4. Structured Logging

### Setup

Use `log/slog` (Go 1.21+) with handler selection based on environment:

```go
func NewLogger(env string) *slog.Logger {
    var handler slog.Handler
    switch env {
    case "production":
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelInfo,
        })
    default:
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelDebug,
        })
    }
    return slog.New(handler)
}
```

JSON in production for machine parsing. Text in development for human
readability.

### Log Levels

| Level | Value | Use |
|-------|-------|-----|
| Debug | -4 | Detailed diagnostic information; disabled in production |
| Info | 0 | Normal operations: startup, shutdown, successful requests |
| Warn | 4 | Client errors, safe degradations, unusual but handled conditions |
| Error | 8 | Server failures, dependency failures, recovered panics |

Do not use log level as a substitute for notification policy. See
[`ERROR_HANDLING_GUIDE.md`](./ERROR_HANDLING_GUIDE.md) section 8 for when to
notify.

### Logging Middleware

Wrap `http.ResponseWriter` to capture the status code, then log method, path,
status, duration, and request ID:

```go
type statusWriter struct {
    http.ResponseWriter
    status int
    written bool
}

func (w *statusWriter) WriteHeader(code int) {
    if !w.written {
        w.status = code
        w.written = true
    }
    w.ResponseWriter.WriteHeader(code)
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

            next.ServeHTTP(sw, r)

            duration := time.Since(start)
            attrs := []slog.Attr{
                slog.String("method", r.Method),
                slog.String("path", r.URL.Path),
                slog.Int("status", sw.status),
                slog.Duration("duration", duration),
                slog.String("request_id", RequestIDFrom(r.Context())),
            }

            level := slog.LevelInfo
            if sw.status >= 500 {
                level = slog.LevelError
            } else if sw.status >= 400 {
                level = slog.LevelWarn
            }

            logger.LogAttrs(r.Context(), level, "http.request", attrs...)
        })
    }
}
```

### Level-Based Logging

Map response status to log level:

- 5xx: `Error` -- server-side failure
- 4xx: `Warn` -- client-caused issue
- 1xx-3xx: `Info` -- normal operation

### Child Loggers

Use `slog.With` to create request-scoped loggers that carry common attributes:

```go
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    logger := h.logger.With(
        slog.String("request_id", RequestIDFrom(r.Context())),
        slog.String("method", r.Method),
        slog.String("path", r.URL.Path),
    )
    // logger carries request context for all downstream log calls
}
```

### Best Practices

- Use consistent key names across the application (`request_id`, not sometimes
  `req_id` and sometimes `requestId`)
- Use `slog.Group` for namespacing related attributes
- Never log sensitive data: passwords, tokens, session IDs, PII
- Use the `"error"` key consistently for error values:
  `slog.String("error", err.Error())`
- Use `slog.ErrorContext(ctx, ...)` to associate log entries with the request
  context

### StatusWriter Considerations

If your application uses streaming, WebSocket upgrades, or server-sent events,
the status-capturing writer must also implement the interfaces the downstream
handler expects:

- `http.Flusher` for SSE and streaming responses
- `http.Hijacker` for WebSocket upgrades

Check interface satisfaction at compile time:

```go
var _ http.Flusher = (*statusWriter)(nil)
var _ http.Hijacker = (*statusWriter)(nil)
```

---

## 5. Centralized Error Handling

### The AppHandler Pattern

Define a custom handler type that returns an error, separating error handling
from business logic:

```go
// AppHandler is an HTTP handler that returns an error for centralized handling.
type AppHandler func(w http.ResponseWriter, r *http.Request) error

func (fn AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if err := fn(w, r); err != nil {
        handleError(w, r, err)
    }
}
```

This eliminates repetitive error-handling boilerplate in every handler.

### Domain Errors

Use a structured error type that carries a machine-readable code, user-safe
message, and optional detail:

```go
type AppError struct {
    Code    int    // HTTP status code
    Message string // user-safe message
    Detail  string // internal detail for logging (never sent to client)
}

func (e *AppError) Error() string {
    return e.Message
}
```

### Predefined Errors

Define constructor functions for common error categories:

```go
func ErrNotFound(msg string) *AppError {
    return &AppError{Code: 404, Message: msg}
}

func ErrUnauthorized(msg string) *AppError {
    return &AppError{Code: 401, Message: msg}
}

func ErrForbidden(msg string) *AppError {
    return &AppError{Code: 403, Message: msg}
}

func ErrBadRequest(msg string) *AppError {
    return &AppError{Code: 400, Message: msg}
}

func ErrConflict(msg string) *AppError {
    return &AppError{Code: 409, Message: msg}
}
```

### Centralized Error Handler

Map known errors to specific responses; unknown errors become 500:

```go
func handleError(w http.ResponseWriter, r *http.Request, err error) {
    var appErr *AppError
    if errors.As(err, &appErr) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(appErr.Code)
        json.NewEncoder(w).Encode(map[string]string{
            "error":  appErr.Message,
            "detail": appErr.Detail,
        })
        return
    }

    // Unknown error -- log internally, return generic message
    slog.ErrorContext(r.Context(), "unhandled error",
        slog.String("error", err.Error()),
        slog.String("request_id", RequestIDFrom(r.Context())),
    )

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusInternalServerError)
    json.NewEncoder(w).Encode(map[string]string{
        "error": "internal server error",
    })
}
```

### Rules

- Never leak internal error details to clients. The user sees the `Message`;
  the server logs the full causal chain.
- Use a consistent JSON error response format: `{"error": "...", "detail": "..."}`
  where `detail` is omitted for unknown errors.
- Map domain errors to application errors at the handler layer, not in services
  or repositories.

For the canonical transport error type used in this project, see
[`ERROR_HANDLING_GUIDE.md`](./ERROR_HANDLING_GUIDE.md) section 3.5.

---

## 6. Recovery Middleware

Recovery middleware catches panics to prevent a single request from crashing
the entire server.

```go
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            defer func() {
                if rv := recover(); rv != nil {
                    stack := debug.Stack()
                    logger.ErrorContext(r.Context(), "panic recovered",
                        slog.Any("panic", rv),
                        slog.String("stack", string(stack)),
                        slog.String("request_id", RequestIDFrom(r.Context())),
                        slog.String("method", r.Method),
                        slog.String("path", r.URL.Path),
                    )

                    w.Header().Set("Content-Type", "application/json")
                    w.WriteHeader(http.StatusInternalServerError)
                    json.NewEncoder(w).Encode(map[string]string{
                        "error": "internal server error",
                    })
                }
            }()
            next.ServeHTTP(w, r)
        })
    }
}
```

### Rules

- Recovery must be the **outermost** middleware. If it is inside the logging
  middleware, the logger never sees the panic.
- Log the stack trace (`runtime/debug.Stack()`) at `Error` level.
- Never expose panic details to clients. The response is always a generic 500.
- Combined with the AppHandler pattern: AppHandler catches returned errors;
  Recovery catches panics. Together they cover both failure modes.

---

## 7. Input Validation

Use `go-playground/validator` at the HTTP boundary. Validate once, trust
downstream.

### Common Validation Tags

**String constraints:**

| Tag | Meaning |
|-----|---------|
| `required` | must not be zero value |
| `min=3` | minimum length 3 |
| `max=100` | maximum length 100 |
| `oneof=admin user guest` | must be one of the listed values |

**Format constraints:**

| Tag | Meaning |
|-----|---------|
| `email` | valid email format |
| `url` | valid URL format |
| `uuid` | valid UUID format |

**Numeric constraints:**

| Tag | Meaning |
|-----|---------|
| `gt=0` | greater than 0 |
| `gte=1` | greater than or equal to 1 |
| `lt=100` | less than 100 |
| `lte=99` | less than or equal to 99 |

### Custom Validators

Register domain-specific validation rules:

```go
func registerCustomValidators(v *validator.Validate) {
    v.RegisterValidation("slug", func(fl validator.FieldLevel) bool {
        return regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`).MatchString(fl.Field().String())
    })

    // With parameters
    v.RegisterValidation("currency", func(fl validator.FieldLevel) bool {
        allowed := map[string]bool{"USD": true, "EUR": true, "GBP": true}
        return allowed[fl.Field().String()]
    })
}
```

### JSON Tag Names in Errors

Register a tag name function so validation errors report JSON field names
instead of Go struct field names:

```go
v.RegisterTagNameFunc(func(fld reflect.StructField) string {
    name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
    if name == "-" {
        return ""
    }
    return name
})
```

### Nested Struct Validation

```go
type CreateOrderRequest struct {
    CustomerID string      `json:"customer_id" validate:"required,uuid"`
    Items      []OrderItem `json:"items"       validate:"required,min=1,dive"`
    Notes      *string     `json:"notes"       validate:"omitempty,max=500"`
    Address    Address     `json:"address"     validate:"required"`
}

type OrderItem struct {
    ProductID string `json:"product_id" validate:"required,uuid"`
    Quantity  int    `json:"quantity"   validate:"required,gt=0,lte=100"`
}
```

- `dive` validates each element inside a slice
- `required` on a nested struct validates the struct itself is present
- pointer fields with `omitempty` skip validation when nil

### Slice and Map Validation

```go
type Config struct {
    Tags     []string          `validate:"required,min=1,max=10,dive,required,min=1,max=50"`
    Settings map[string]string `validate:"required,dive,keys,required,min=1,endkeys,required"`
}
```

Use `dive` to enter the collection. For maps, `keys` and `endkeys` delimit key
validation from value validation.

### Cross-Field Validation

```go
type PasswordChange struct {
    NewPassword     string `json:"new_password"     validate:"required,min=8"`
    ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
}

type DateRange struct {
    StartDate time.Time `json:"start_date" validate:"required"`
    EndDate   time.Time `json:"end_date"   validate:"required,gtfield=StartDate"`
}
```

Available cross-field tags: `eqfield`, `nefield`, `gtfield`, `gtefield`,
`ltfield`, `ltefield`.

### Struct-Level Validation

For complex multi-field rules that cannot be expressed with tags:

```go
v.RegisterStructValidation(func(sl validator.StructLevel) {
    order := sl.Current().Interface().(CreateOrderRequest)

    total := 0
    for _, item := range order.Items {
        total += item.Quantity
    }
    if total > 1000 {
        sl.ReportError(order.Items, "items", "Items", "max_total_quantity", "")
    }
}, CreateOrderRequest{})
```

### Error Formatting

Format validation errors into structured, client-friendly responses:

```go
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

func formatValidationErrors(err error) []ValidationError {
    var ve validator.ValidationErrors
    if !errors.As(err, &ve) {
        return nil
    }

    out := make([]ValidationError, len(ve))
    for i, fe := range ve {
        out[i] = ValidationError{
            Field:   fe.Field(),
            Message: msgForTag(fe),
        }
    }
    return out
}

func msgForTag(fe validator.FieldError) string {
    switch fe.Tag() {
    case "required":
        return "this field is required"
    case "email":
        return "must be a valid email address"
    case "min":
        return fmt.Sprintf("must be at least %s characters", fe.Param())
    case "max":
        return fmt.Sprintf("must be at most %s characters", fe.Param())
    case "uuid":
        return "must be a valid UUID"
    case "oneof":
        return fmt.Sprintf("must be one of: %s", fe.Param())
    default:
        return fmt.Sprintf("failed on '%s' validation", fe.Tag())
    }
}
```

### Body Size Limiting

Always limit request body size to prevent resource exhaustion:

```go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
    r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB

    var input CreateRequest
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        return ErrBadRequest("invalid or oversized request body")
    }

    if err := h.validator.Struct(input); err != nil {
        return &ValidationResponse{Errors: formatValidationErrors(err)}
    }

    // proceed with validated input
}
```

---

## 8. Handler Testing

### httptest Fundamentals

Every handler test uses `httptest.NewRequest` and `httptest.NewRecorder`:

```go
func TestGetUser(t *testing.T) {
    svc := &fakeUserService{
        user: User{ID: testID, Name: "Alice"},
    }
    handler := NewUserHandler(svc, slog.Default())

    req := httptest.NewRequest(http.MethodGet, "/users/"+testID.String(), nil)
    rec := httptest.NewRecorder()

    handler.GetByID(rec, req)

    if rec.Code != http.StatusOK {
        t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
    }
}
```

### Table-Driven Tests

Cover both happy paths and error paths in a single test function:

```go
func TestCreateUser(t *testing.T) {
    tests := []struct {
        name       string
        body       string
        svcErr     error
        wantStatus int
        wantBody   string
    }{
        {
            name:       "success",
            body:       `{"name":"Alice","email":"alice@example.com"}`,
            wantStatus: http.StatusCreated,
        },
        {
            name:       "validation error missing name",
            body:       `{"email":"alice@example.com"}`,
            wantStatus: http.StatusUnprocessableEntity,
        },
        {
            name:       "duplicate email",
            body:       `{"name":"Alice","email":"taken@example.com"}`,
            svcErr:     ErrEmailTaken,
            wantStatus: http.StatusConflict,
        },
        {
            name:       "service failure",
            body:       `{"name":"Alice","email":"alice@example.com"}`,
            svcErr:     errors.New("db connection lost"),
            wantStatus: http.StatusInternalServerError,
        },
        {
            name:       "malformed JSON",
            body:       `{invalid`,
            wantStatus: http.StatusBadRequest,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := &fakeUserService{err: tt.svcErr}
            handler := NewUserHandler(svc, slog.Default())

            req := httptest.NewRequest(http.MethodPost, "/users",
                strings.NewReader(tt.body))
            req.Header.Set("Content-Type", "application/json")
            rec := httptest.NewRecorder()

            handler.Create(rec, req)

            if rec.Code != tt.wantStatus {
                t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
            }
        })
    }
}
```

### Middleware Testing

Test middleware in isolation by providing a controlled `next` handler:

```go
func TestRequestIDMiddleware(t *testing.T) {
    var capturedID string
    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        capturedID = RequestIDFrom(r.Context())
        w.WriteHeader(http.StatusOK)
    })

    handler := RequestID(next)
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    rec := httptest.NewRecorder()

    handler.ServeHTTP(rec, req)

    if capturedID == "" {
        t.Error("request ID not set in context")
    }
    if rec.Header().Get("X-Request-ID") == "" {
        t.Error("X-Request-ID header not set")
    }
}
```

Test the full middleware stack to verify ordering:

```go
func TestMiddlewareStack(t *testing.T) {
    var order []string
    makeMiddleware := func(name string) func(http.Handler) http.Handler {
        return func(next http.Handler) http.Handler {
            return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                order = append(order, name+":before")
                next.ServeHTTP(w, r)
                order = append(order, name+":after")
            })
        }
    }

    final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        order = append(order, "handler")
    })

    handler := Chain(final,
        makeMiddleware("first"),
        makeMiddleware("second"),
    )

    req := httptest.NewRequest(http.MethodGet, "/", nil)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)

    want := []string{"first:before", "second:before", "handler", "second:after", "first:after"}
    if !slices.Equal(order, want) {
        t.Errorf("order = %v, want %v", order, want)
    }
}
```

Test recovery middleware:

```go
func TestRecoveryMiddleware(t *testing.T) {
    panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        panic("unexpected failure")
    })

    handler := Recovery(slog.Default())(panicking)
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    rec := httptest.NewRecorder()

    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusInternalServerError {
        t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
    }
}
```

### Auth and Authorization Testing

Test the full range of authentication states:

```go
func TestAuthMiddleware(t *testing.T) {
    tests := []struct {
        name       string
        token      string
        wantStatus int
        wantCalled bool
    }{
        {
            name:       "valid token",
            token:      "valid-token",
            wantStatus: http.StatusOK,
            wantCalled: true,
        },
        {
            name:       "invalid token",
            token:      "invalid-token",
            wantStatus: http.StatusUnauthorized,
            wantCalled: false,
        },
        {
            name:       "missing token",
            token:      "",
            wantStatus: http.StatusUnauthorized,
            wantCalled: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            called := false
            next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                called = true
                w.WriteHeader(http.StatusOK)
            })

            verifier := &fakeTokenVerifier{valid: tt.token == "valid-token"}
            handler := Auth(verifier)(next)

            req := httptest.NewRequest(http.MethodGet, "/", nil)
            if tt.token != "" {
                req.Header.Set("Authorization", "Bearer "+tt.token)
            }
            rec := httptest.NewRecorder()

            handler.ServeHTTP(rec, req)

            if rec.Code != tt.wantStatus {
                t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
            }
            if called != tt.wantCalled {
                t.Errorf("next called = %v, want %v", called, tt.wantCalled)
            }
        })
    }
}
```

For role-based authorization, test that the correct role grants access and
incorrect roles are rejected:

```go
func TestRoleAuthorization(t *testing.T) {
    tests := []struct {
        name         string
        userRole     string
        requiredRole string
        wantStatus   int
    }{
        {"admin accessing admin route", "admin", "admin", http.StatusOK},
        {"user accessing admin route", "user", "admin", http.StatusForbidden},
        {"guest accessing user route", "guest", "user", http.StatusForbidden},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
            })

            handler := RequireRole(tt.requiredRole)(next)

            ctx := context.WithValue(context.Background(), userKey, User{Role: tt.userRole})
            req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
            rec := httptest.NewRecorder()

            handler.ServeHTTP(rec, req)

            if rec.Code != tt.wantStatus {
                t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
            }
        })
    }
}
```

### Integration Tests

For tests that require a real database:

```go
func TestUserRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    pool := setupTestDB(t)
    repo := NewUserRepository(pool)

    t.Run("create and retrieve", func(t *testing.T) {
        truncateUsers(t, pool)

        user, err := repo.Create(context.Background(), CreateUserInput{
            Name:  "Alice",
            Email: "alice@example.com",
        })
        if err != nil {
            t.Fatalf("create: %v", err)
        }

        got, err := repo.GetByID(context.Background(), user.ID)
        if err != nil {
            t.Fatalf("get: %v", err)
        }

        if got.Name != "Alice" {
            t.Errorf("name = %q, want %q", got.Name, "Alice")
        }
    })
}

func truncateUsers(t *testing.T, pool *pgxpool.Pool) {
    t.Helper()
    t.Cleanup(func() {
        _, _ = pool.Exec(context.Background(), "TRUNCATE users CASCADE")
    })
}
```

### File Upload Testing

Test multipart uploads with realistic payloads:

```go
func TestFileUpload(t *testing.T) {
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    part, err := writer.CreateFormFile("avatar", "photo.jpg")
    if err != nil {
        t.Fatal(err)
    }
    part.Write([]byte("fake image content"))
    writer.Close()

    req := httptest.NewRequest(http.MethodPost, "/upload", body)
    req.Header.Set("Content-Type", writer.FormDataContentType())
    rec := httptest.NewRecorder()

    handler.Upload(rec, req)

    if rec.Code != http.StatusOK {
        t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
    }
}
```

Test size limits:

```go
func TestFileUpload_TooLarge(t *testing.T) {
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    part, _ := writer.CreateFormFile("avatar", "huge.jpg")
    part.Write(make([]byte, 11<<20)) // 11 MB, exceeds 10 MB limit
    writer.Close()

    req := httptest.NewRequest(http.MethodPost, "/upload", body)
    req.Header.Set("Content-Type", writer.FormDataContentType())
    rec := httptest.NewRecorder()

    handler.Upload(rec, req)

    if rec.Code != http.StatusRequestEntityTooLarge {
        t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
    }
}
```

### Streaming Response Testing

For server-sent events or long-lived connections, use `httptest.Server`:

```go
func TestSSE(t *testing.T) {
    srv := httptest.NewServer(handler)
    defer srv.Close()

    resp, err := http.Get(srv.URL + "/events")
    if err != nil {
        t.Fatal(err)
    }
    defer resp.Body.Close()

    scanner := bufio.NewScanner(resp.Body)
    if !scanner.Scan() {
        t.Fatal("expected at least one event")
    }

    line := scanner.Text()
    if !strings.HasPrefix(line, "data:") {
        t.Errorf("line = %q, want data: prefix", line)
    }
}
```

### Test Fixtures and Golden Files

Place test fixtures in a `testdata/` directory adjacent to the test file. Go
tooling ignores `testdata/` directories during builds.

For golden file testing, accept an `-update` flag to regenerate expected output:

```go
var update = flag.Bool("update", false, "update golden files")

func TestRender(t *testing.T) {
    got := renderComponent(input)

    golden := filepath.Join("testdata", t.Name()+".golden")
    if *update {
        os.WriteFile(golden, []byte(got), 0644)
    }

    want, err := os.ReadFile(golden)
    if err != nil {
        t.Fatal(err)
    }

    if got != string(want) {
        t.Errorf("output mismatch:\ngot:  %s\nwant: %s", got, want)
    }
}
```

### Interface-Based Fakes

Define narrow interfaces at the consumer and implement fakes with function
fields for flexible test control:

```go
// Defined in the handler package -- narrow interface for what the handler needs
type UserService interface {
    GetByID(ctx context.Context, id uuid.UUID) (User, error)
    Create(ctx context.Context, input CreateUserInput) (User, error)
}

// Fake implementation for tests
type fakeUserService struct {
    user User
    err  error
}

func (f *fakeUserService) GetByID(_ context.Context, _ uuid.UUID) (User, error) {
    return f.user, f.err
}

func (f *fakeUserService) Create(_ context.Context, _ CreateUserInput) (User, error) {
    return f.user, f.err
}
```

Never use mock generation libraries. Hand-written fakes are simpler, more
readable, and do not couple tests to implementation details.

---

## 9. Self-Review Checklist

Before merging application-layer code, confirm every applicable item:

### Handlers

- [ ] All handlers are methods on a struct, not free functions with global state
- [ ] All dependencies are injected through the struct constructor
- [ ] Input is validated at the handler boundary before calling services
- [ ] Request body size is limited with `http.MaxBytesReader`
- [ ] Domain errors are mapped to appropriate HTTP status codes
- [ ] Unknown errors produce 500 with a generic message; internal details are logged
- [ ] Context is propagated from the request to all downstream calls
- [ ] Path parameters are parsed and validated (e.g., `uuid.Parse`)

### Services

- [ ] Services accept `context.Context` as the first parameter
- [ ] Services depend on repository interfaces, not concrete types
- [ ] Services return domain errors, not HTTP-aware errors
- [ ] Services do not import handler or transport packages
- [ ] Business rules are enforced in the service, not the handler or repository

### Middleware

- [ ] Recovery is the outermost middleware
- [ ] Request ID is assigned before logging middleware runs
- [ ] Context propagation uses `r.WithContext`, not request mutation
- [ ] Context keys are unexported typed constants, not strings
- [ ] Guard middleware (auth, rate limit) does not call `next` on failure
- [ ] Status-capturing writers implement `http.Flusher` if streaming is needed

### Logging

- [ ] JSON handler in production, text handler in development
- [ ] Log level matches severity: 5xx=Error, 4xx=Warn, else=Info
- [ ] No sensitive data in log output (passwords, tokens, PII)
- [ ] Consistent key names across all log calls
- [ ] Request ID included in every request-scoped log entry
- [ ] No duplicate logging between middleware and error handlers

### Validation

- [ ] Validation happens at the handler boundary only
- [ ] Validation errors produce structured field-level error responses
- [ ] Custom validators are registered at startup, not per-request
- [ ] `RegisterTagNameFunc` maps JSON tag names for client-friendly errors
- [ ] Nested structs use `dive` for slices and `required` for nested objects

### Testing

- [ ] Every handler has a `_test.go` file
- [ ] Table-driven tests cover happy path, validation errors, not-found, and 500
- [ ] Tests use `httptest.NewRequest` and `httptest.NewRecorder`
- [ ] Fakes implement the same interface the handler depends on
- [ ] No shared mutable state between test cases
- [ ] Integration tests use `t.Cleanup` for database teardown
- [ ] Error responses are asserted on status code, not just "not 200"

### Error Handling

- [ ] Every error is checked; no `_ =` without a documented reason
- [ ] Propagated errors are wrapped with `fmt.Errorf("context: %w", err)`
- [ ] `errors.Is` / `errors.As` used for branching; never `err.Error()` comparison
- [ ] Recovery middleware catches panics; AppHandler catches returned errors
- [ ] Stack traces are logged but never sent to clients
- [ ] `context.Canceled` is treated as a non-fault event

---

## 10. Anti-Patterns

These patterns cause bugs, test fragility, or security issues. Reject them in
code review.

### Context

- **String or int context keys.** They collide across packages. Always use an
  unexported typed constant.
- **Storing large objects in context.** Context carries request-scoped metadata
  (IDs, auth claims), not full domain objects, database results, or file
  contents.
- **Using `context.WithValue` for function parameters.** If a value is required
  by the function signature, make it an explicit parameter. Context is for
  cross-cutting concerns that flow through middleware, not for avoiding function
  arguments.

### Middleware

- **Writing the response before calling `next`.** This prevents downstream
  handlers from setting headers or status codes. Write *after* `next.ServeHTTP`
  unless the middleware is intentionally short-circuiting.
- **Forgetting to call `next.ServeHTTP`.** The request silently disappears. A
  middleware that does not call next must explicitly write a response.
- **Recovery middleware in the wrong position.** If Recovery is inside other
  middleware, panics in the outer middleware crash the process.

### Validation

- **Validating in the service layer.** Structural validation belongs at the
  handler boundary. Services enforce business rules (uniqueness, state
  transitions), not field-level constraints.
- **Not limiting request body size.** An attacker can send an arbitrarily large
  body. Always use `http.MaxBytesReader`.

### Testing

- **Calling handler methods directly in tests.** Use `ServeHTTP` through the
  standard `httptest` flow so middleware, routing, and response writing behave
  realistically.
- **Shared mutable test state.** Each test case must construct its own fakes
  and request/response objects. Shared state causes ordering-dependent failures.
- **Not testing error responses.** If a test only asserts on the happy path, a
  regression in error handling goes undetected.
- **Asserting on exact JSON strings.** Parse the JSON into a struct or map and
  assert on fields. Exact string comparison breaks when field ordering, spacing,
  or escaping changes.

### Error Handling

- **Logging and returning the same error.** This creates duplicate log entries
  for the same failure. Return the error and let the boundary log it once.
- **Leaking internal details in error responses.** SQL errors, file paths,
  stack traces, and dependency names must never appear in client-facing output.
- **Using `err == ErrFoo` instead of `errors.Is`.** Direct comparison breaks
  when the error is wrapped. Always use `errors.Is` or `errors.As`.

---

## Sources

- Go standard library: `net/http`, `log/slog`, `context`, `net/http/httptest`
- Go documentation: <https://go.dev/doc/effective_go>
- Go Code Review Comments: <https://go.dev/wiki/CodeReviewComments>
- go-playground/validator: <https://github.com/go-playground/validator>
