# r-test — Testing Excellence Rules

> Ruleset for `cc-test`. Read entirely before writing any test.

---

## Test Philosophy

- **Test at boundary, not implementation.** Assert what goes in and out. Never assert internal struct fields callers can't observe.
- **One test file per implementation file.** `user.go` → `user_test.go`, same package.
- **Table-driven tests** for any function with 3+ input variations.
- **Fakes over mocks.** Implement interface manually as `fake` struct. Never use mock generation libraries (gomock, testify/mock) — they couple tests to implementation details.
- **No test databases in service tests.** Service tests use fake repositories. Repository tests use real `test_data` PostgreSQL container.

---

## Test File Locations

Tests are **co-located with implementation** in same package and directory. Never create separate test directory layer:

| Code under test | Test file location |
|---|---|
| `internal/features/<name>/handler.go` | `internal/features/<name>/handler_test.go` |
| `internal/features/<name>/service.go` | `internal/features/<name>/service_test.go` |
| `internal/repository/<file>.go` | `internal/repository/<file>_test.go` |
| `internal/view/page/<file>.go` | `internal/view/page/<file>_test.go` |

Repository tests for `internal/repository/` require real `test_data` PostgreSQL container. Tests for external shared modules are maintained upstream — do not write them here.

---

## Handler Tests

Test full HTTP contract: status code, redirect URL, rendered HTML.

### Setup
```go
e := echo.New()
req := httptest.NewRequest(http.MethodGet, "/users", nil)
rec := httptest.NewRecorder()
c := e.NewContext(req, rec)
```

### What to Assert
1. **Status code** — exact match (`http.StatusOK`, `http.StatusSeeOther`, etc.)
2. **Redirect target** — `rec.Header().Get("Location")` for 3xx responses
3. **HTML content** — exact string match on rendered gomponent output
4. **HTMX partial mode** — set `HX-Request: true` header; assert fragment-only output

### HTML Assertions
HTML entities MUST be encoded in test strings:
```go
// ✅ correct
assert.Contains(t, body, "O&#39;Brien")     // apostrophe
assert.Contains(t, body, "AT&amp;T")        // ampersand
assert.Equal(t, expected, rec.Body.String()) // exact match preferred
```
**Never use substring matching when exact match is possible.** Substring tests create false positives when surrounding HTML changes.

### Error Path Coverage (mandatory)
Every handler test file MUST include test cases for:
- Service returns domain error (e.g., `model.ErrUserNotFound` → expect 404)
- Service returns unexpected error → expect 500
- Invalid path parameter (malformed UUID) → expect 400
- Missing required form field → expect re-render with validation errors
- Unauthorized access (if route is protected) → expect redirect to login

---

## Service Tests

Test that service passes correct values to repository and returns correct domain model.

### Fake Repository Pattern
```go
type fakeUserRepo struct {
    users  []model.User
    called string    // track which method was called
    arg    any       // capture last argument
    err    error     // inject error to return
}

func (f *fakeUserRepo) GetByID(ctx context.Context, id uuid.UUID) (model.User, error) {
    f.called = "GetByID"
    f.arg = id
    if f.err != nil { return model.User{}, f.err }
    for _, u := range f.users { if u.ID == id { return u, nil } }
    return model.User{}, model.ErrUserNotFound
}
```

### What to Assert
1. **Correct values passed to repo** — inspect `fakeRepo.arg` after service call
2. **Correct model returned** — field-by-field equality, not just non-nil
3. **Domain errors propagated** — `errors.Is(err, model.ErrUserNotFound)`
4. **Business rules enforced** — service rejects invalid state before calling repo

### Error Injection
```go
repo.err = model.ErrEmailTaken
_, err := svc.Update(ctx, input)
assert.ErrorIs(t, err, model.ErrEmailTaken)
```

---

## Repository Tests

Test against real database. Use `test_data` PostgreSQL container (test_network).

### Test Environment
- Database URL from `TEST_DATABASE_URL` env var
- Extensions installed via `db/init-test/01-extensions.sql`
- Use `t.Cleanup()` to truncate tables between tests

### What to Assert
1. **db → model mapping** — verify every field maps correctly (especially nullable fields)
2. **Constraint violations** — insert duplicate email; assert `model.ErrEmailTaken`
3. **Not found** — query non-existent ID; assert `model.ErrUserNotFound`
4. **Roundtrip** — create then read back; assert equality

### Cleanup Pattern
```go
func truncateUsers(t *testing.T, pool *pgxpool.Pool) {
    t.Helper()
    t.Cleanup(func() {
        _, _ = pool.Exec(context.Background(), "TRUNCATE users CASCADE")
    })
}
```

---

## Middleware Tests

### What to Assert
1. **Context values set** — after middleware runs, `c.Get(key)` returns expected value
2. **Next called** — use flag in next handler to verify middleware proceeds
3. **Next NOT called** — for guard middleware (auth check), verify 401/403 on failure
4. **Response shape** — status code and body of blocked requests

### Pattern
```go
called := false
next := func(c *echo.Context) error {
    called = true
    return nil
}
err := middleware(next)(c)
assert.True(t, called)
assert.NoError(t, err)
```

---

## View / Component Tests

### Tools
Use project's component test utilities to convert `g.Node` to string (see `README.md` for relevant package path).

### What to Assert
- Exact rendered HTML string (not substring) for stable components
- Presence of required attributes (`id`, `aria-*`, `data-*`)
- Absence of sensitive data in rendered output
- Correct HTML entity encoding in dynamic content

### HTML Entity Rules
```go
// Input: user.Name = "O'Brien & Associates"
// Expected rendered: O&#39;Brien &amp; Associates
want := `<td>O&#39;Brien &amp; Associates</td>`
assert.Equal(t, want, rendered)
```

---

## Security Test Requirements

Security-sensitive code requires **both** pass and fail test cases:

| Feature | Required Tests |
|---------|---------------|
| CSRF validation | Valid token passes; missing/invalid token → 403 |
| Auth guard | Authenticated user passes; unauthenticated → redirect |
| Rate limiter | Under limit passes; over limit → 429 |
| Origin check | Same-origin passes; cross-origin → 403 |
| Input validation | Valid input passes; invalid input → 422 with field errors |

---

## Coverage Requirements

- Every **exported function** has at least one test
- Every **error path** in handlers has dedicated test case
- Every **domain error** sentinel tested for correct HTTP mapping
- **No test skips** without comment explaining when skip will be removed

---

## Running Tests

```bash
# Full suite (always run after any change)
GONOSUMDB='*' /usr/local/go/bin/go test ./...

# Single feature (for iteration during development)
/usr/local/go/bin/go test ./internal/features/<name>/...

# With race detector (run before committing)
/usr/local/go/bin/go test -race ./...

# Repository tests require test_data container
# Ensure docker compose --profile test up -d before running repo tests
```

Always run **full suite** before declaring task complete — not just changed package.
