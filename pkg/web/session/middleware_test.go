package session

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/cookiecodec"
)

// fakeErrorStore satisfies Store. Read always returns ErrSessionNotFound (triggers
// newSession path). Save always returns an error to simulate a storage failure.
// Delete always returns nil.
type fakeErrorStore struct {
	saveErr   error
	deleteErr error
}

type failNthSaveStore struct {
	base       *MemoryStore
	failOnCall int
	saveErr    error
	saveCalls  int
}

type failDeleteTokenStore struct {
	base            *MemoryStore
	failDeleteToken string
	deleteErr       error
}

type recordingSaveStore struct {
	savedAbsolute time.Time
}

// wrappedNotFoundStore returns a wrapped ErrSessionNotFound from Read so that
// a raw == comparison would fail while errors.Is succeeds — exercising the G1 fix.
type wrappedNotFoundStore struct{}

func (f *fakeErrorStore) Read(_ context.Context, _ string) ([]byte, int64, error) {
	return nil, 0, ErrSessionNotFound
}

func (f *fakeErrorStore) Save(_ context.Context, _ string, _ []byte, _ time.Time, _ time.Duration, _ int64) (string, error) {
	return "", f.saveErr
}

func (f *fakeErrorStore) Delete(_ context.Context, _ string) error {
	return f.deleteErr
}

func (s *failNthSaveStore) Read(ctx context.Context, token string) ([]byte, int64, error) {
	return s.base.Read(ctx, token)
}

func (s *failNthSaveStore) Save(ctx context.Context, token string, data []byte, absolute time.Time, idleTTL time.Duration, version int64) (string, error) {
	s.saveCalls++
	if s.saveCalls == s.failOnCall {
		return "", s.saveErr
	}
	return s.base.Save(ctx, token, data, absolute, idleTTL, version)
}

func (s *failNthSaveStore) Delete(ctx context.Context, token string) error {
	return s.base.Delete(ctx, token)
}

func (s *failDeleteTokenStore) Read(ctx context.Context, token string) ([]byte, int64, error) {
	return s.base.Read(ctx, token)
}

func (s *failDeleteTokenStore) Save(ctx context.Context, token string, data []byte, absolute time.Time, idleTTL time.Duration, version int64) (string, error) {
	return s.base.Save(ctx, token, data, absolute, idleTTL, version)
}

func (s *failDeleteTokenStore) Delete(ctx context.Context, token string) error {
	if token == s.failDeleteToken {
		return s.deleteErr
	}
	return s.base.Delete(ctx, token)
}

func (s *recordingSaveStore) Read(context.Context, string) ([]byte, int64, error) {
	return nil, 0, ErrSessionNotFound
}

func (s *recordingSaveStore) Save(_ context.Context, _ string, _ []byte, absolute time.Time, _ time.Duration, _ int64) (string, error) {
	s.savedAbsolute = absolute
	return "token", nil
}

func (s *recordingSaveStore) Delete(context.Context, string) error {
	return nil
}

func testMemoryConfig(t *testing.T) Config {
	t.Helper()
	return Config{
		Store: NewMemoryStore(),
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}
}

func testCookieConfig(t *testing.T) Config {
	t.Helper()
	codec, err := cookiecodec.New(cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("32-byte-key-for-aead-encryption!")},
		Mode:    cookiecodec.AEAD,
	})
	if err != nil {
		t.Fatalf("cookiecodec.New: %v", err)
	}
	return Config{
		Store: NewCookieStore(codec),
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}
}

func runRequest(t *testing.T, mw web.Middleware, cookieHeader string, fn func(*web.Context) (web.Response, error)) (web.Response, error) {
	t.Helper()
	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	if cookieHeader != "" {
		req.Headers.Set("Cookie", cookieHeader)
	}
	return mw(fn)(web.NewContext(context.Background(), req))
}

func recoverMiddleware(next web.Handler) web.Handler {
	return func(c *web.Context) (resp web.Response, herr error) {
		defer func() {
			if recover() != nil {
				resp = web.Text(http.StatusInternalServerError, "Internal Server Error")
				herr = nil
			}
		}()
		return next(c)
	}
}

func extractSetCookie(t *testing.T, resp web.Response) string {
	t.Helper()
	v := resp.Headers.Get("Set-Cookie")
	if v == "" {
		t.Fatal("response missing Set-Cookie")
	}
	return v
}

func TestMiddleware_MemoryStore_RoundTrip(t *testing.T) {
	cfg := testMemoryConfig(t)
	mw := Middleware(cfg)

	// Request 1: set a value.
	resp1, _ := runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Set("user", "alice")
		return web.Respond(http.StatusOK), nil
	})
	cookie := extractSetCookie(t, resp1)

	if strings.Contains(cookie, `"user"`) {
		t.Fatal("server-side session cookie must not embed payload")
	}

	// Request 2: read value back.
	var gotUser string
	_, _ = runRequest(t, mw, cookie, func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		v, ok, err := Get[string](sess, "user")
		if err != nil || !ok {
			t.Fatalf("Get user: err=%v ok=%v", err, ok)
		}
		gotUser = v
		return web.Respond(http.StatusOK), nil
	})
	if gotUser != "alice" {
		t.Fatalf("user = %q, want alice", gotUser)
	}
}

func TestMiddleware_CookieStore_RoundTrip(t *testing.T) {
	cfg := testCookieConfig(t)
	mw := Middleware(cfg)

	resp1, _ := runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Set("role", "admin")
		return web.Respond(http.StatusOK), nil
	})
	cookie := extractSetCookie(t, resp1)

	var gotRole string
	_, _ = runRequest(t, mw, cookie, func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		v, ok, err := Get[string](sess, "role")
		if err != nil || !ok {
			t.Fatalf("Get role: err=%v ok=%v", err, ok)
		}
		gotRole = v
		return web.Respond(http.StatusOK), nil
	})
	if gotRole != "admin" {
		t.Fatalf("role = %q, want admin", gotRole)
	}
}

func TestMiddleware_Destroy_ExpiresCookie(t *testing.T) {
	cfg := testMemoryConfig(t)
	mw := Middleware(cfg)

	resp1, _ := runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Set("x", 1)
		return web.Respond(http.StatusOK), nil
	})
	cookie := extractSetCookie(t, resp1)

	resp2, _ := runRequest(t, mw, cookie, func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		sess.Destroy()
		return web.Respond(http.StatusOK), nil
	})

	sc := resp2.Headers.Get("Set-Cookie")
	if !strings.Contains(sc, "Max-Age=0") {
		t.Fatalf("Set-Cookie = %q, want Max-Age=0", sc)
	}
}

func TestP0_13_Session_ClearSiteDataOnDestroy(t *testing.T) {
	cfg := testMemoryConfig(t)
	mw := Middleware(cfg)

	resp, _ := runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		sess.Destroy()
		return web.Respond(http.StatusOK), nil
	})

	csd := resp.Headers.Get("Clear-Site-Data")
	if csd == "" {
		t.Fatal("Clear-Site-Data header absent after Destroy")
	}
	if !strings.Contains(csd, "cookies") {
		t.Fatalf("Clear-Site-Data = %q, want to contain 'cookies'", csd)
	}
}

func TestP0_03_Session_Regenerate(t *testing.T) {
	cfg := testMemoryConfig(t)
	mw := Middleware(cfg)

	// Establish a session.
	resp1, _ := runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Set("uid", "42")
		return web.Respond(http.StatusOK), nil
	})
	cookie1 := extractSetCookie(t, resp1)
	token1 := cookieValue(cookie1, "sess")

	// Regenerate.
	resp2, _ := runRequest(t, mw, cookie1, func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		sess.Regenerate()
		return web.Respond(http.StatusOK), nil
	})
	cookie2 := extractSetCookie(t, resp2)
	token2 := cookieValue(cookie2, "sess")

	if token1 == token2 {
		t.Fatal("session token unchanged after Regenerate")
	}

	// Old token is gone from store.
	memStore := cfg.Store.(*MemoryStore)
	_, _, err := memStore.Read(context.Background(), token1)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("old token still readable after Regenerate: %v", err)
	}

	// Data preserved under new token.
	var uid string
	_, _ = runRequest(t, mw, cookie2, func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		v, ok, err := Get[string](sess, "uid")
		if !ok || err != nil {
			t.Fatalf("Get uid after regenerate: ok=%v err=%v", ok, err)
		}
		uid = v
		return web.Respond(http.StatusOK), nil
	})
	if uid != "42" {
		t.Fatalf("uid after regenerate = %q, want 42", uid)
	}
}

func TestMiddleware_Regenerate_SaveFailurePreservesExistingSession(t *testing.T) {
	base := NewMemoryStore()
	defer base.Stop()

	store := &failNthSaveStore{
		base:       base,
		failOnCall: 2,
		saveErr:    errors.New("store unavailable"),
	}
	cfg := Config{
		Store: store,
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}
	mw := Middleware(cfg)

	resp1, err := runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Set("uid", "42")
		return web.Respond(http.StatusOK), nil
	})
	if err != nil {
		t.Fatalf("establish session: %v", err)
	}
	cookie := extractSetCookie(t, resp1)

	resp2, err := runRequest(t, mw, cookie, func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		sess.Regenerate()
		return web.Respond(http.StatusOK), nil
	})
	if err == nil {
		t.Fatal("expected regenerate save failure, got nil")
	}
	if sc := resp2.Headers.Get("Set-Cookie"); sc != "" {
		t.Fatalf("Set-Cookie = %q, want empty after failed regenerate save", sc)
	}

	var uid string
	_, err = runRequest(t, mw, cookie, func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		v, ok, getErr := Get[string](sess, "uid")
		if !ok || getErr != nil {
			t.Fatalf("Get uid after failed regenerate: ok=%v err=%v", ok, getErr)
		}
		uid = v
		return web.Respond(http.StatusOK), nil
	})
	if err != nil {
		t.Fatalf("reuse original session after failed regenerate: %v", err)
	}
	if uid != "42" {
		t.Fatalf("uid after failed regenerate = %q, want 42", uid)
	}
}

func TestMiddleware_Regenerate_DeleteFailurePreservesExistingSession(t *testing.T) {
	base := NewMemoryStore()
	defer base.Stop()

	store := &failDeleteTokenStore{
		base:      base,
		deleteErr: errors.New("delete unavailable"),
	}
	cfg := Config{
		Store: store,
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}
	mw := Middleware(cfg)

	resp1, err := runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Set("uid", "42")
		return web.Respond(http.StatusOK), nil
	})
	if err != nil {
		t.Fatalf("establish session: %v", err)
	}
	cookie := extractSetCookie(t, resp1)
	store.failDeleteToken = cookieValue(cookie, "sess")

	resp2, err := runRequest(t, mw, cookie, func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		sess.Regenerate()
		return web.Respond(http.StatusOK), nil
	})
	if err == nil {
		t.Fatal("expected regenerate delete failure, got nil")
	}
	if sc := resp2.Headers.Get("Set-Cookie"); sc != "" {
		t.Fatalf("Set-Cookie = %q, want empty after failed old-token delete", sc)
	}

	var uid string
	_, err = runRequest(t, mw, cookie, func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		v, ok, getErr := Get[string](sess, "uid")
		if !ok || getErr != nil {
			t.Fatalf("Get uid after failed old-token delete: ok=%v err=%v", ok, getErr)
		}
		uid = v
		return web.Respond(http.StatusOK), nil
	})
	if err != nil {
		t.Fatalf("reuse original session after failed old-token delete: %v", err)
	}
	if uid != "42" {
		t.Fatalf("uid after failed old-token delete = %q, want 42", uid)
	}
}

func TestP0_04_Session_CookieAEADNoPlaintext(t *testing.T) {
	cfg := testCookieConfig(t)
	mw := Middleware(cfg)

	resp, _ := runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Set("secret", "hunter2")
		return web.Respond(http.StatusOK), nil
	})

	cookie := extractSetCookie(t, resp)
	if strings.Contains(cookie, "hunter2") {
		t.Fatal("session payload visible in cookie value (no encryption)")
	}
	if strings.Contains(cookie, "secret") {
		t.Fatal("session key visible in cookie value (no encryption)")
	}
}

func TestMiddleware_Flash_CrossRequest(t *testing.T) {
	cfg := testCookieConfig(t)
	mw := Middleware(cfg)

	resp1, _ := runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Flash("notice", "saved!")
		return web.Respond(http.StatusOK), nil
	})
	cookie := extractSetCookie(t, resp1)

	var notice string
	_, _ = runRequest(t, mw, cookie, func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		v, ok, err := FlashPop[string](sess, "notice")
		if !ok || err != nil {
			t.Fatalf("FlashPop: ok=%v err=%v", ok, err)
		}
		notice = v
		return web.Respond(http.StatusOK), nil
	})
	if notice != "saved!" {
		t.Fatalf("notice = %q, want 'saved!'", notice)
	}
}

func TestMiddleware_IsNew(t *testing.T) {
	cfg := testMemoryConfig(t)
	mw := Middleware(cfg)

	var wasNew bool
	resp1, _ := runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		wasNew = sess.IsNew()
		return web.Respond(http.StatusOK), nil
	})
	if !wasNew {
		t.Fatal("IsNew = false on first request, want true")
	}

	// No writes were made, so no Set-Cookie should be emitted.
	if sc := resp1.Headers.Get("Set-Cookie"); sc != "" {
		t.Fatalf("Set-Cookie = %q, want empty (no writes, no cookie)", sc)
	}

	// Second request without a session cookie: session is again fresh and new.
	var wasNewOnSecond bool
	_, _ = runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		wasNewOnSecond = sess.IsNew()
		return web.Respond(http.StatusOK), nil
	})
	if !wasNewOnSecond {
		t.Fatal("IsNew = false on second cookieless request, want true")
	}
}

// TestMiddleware_CommitError_Returns500 verifies Bug 4: when Store.Save returns an
// error during commit, the middleware returns an internal server error.
func TestMiddleware_CommitError_Returns500(t *testing.T) {
	cfg := Config{
		Store: &fakeErrorStore{saveErr: errors.New("store unavailable")},
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}
	mw := Middleware(cfg)

	_, err := runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		// Mark dirty so commit reaches Store.Save.
		_ = sess.Set("k", "v")
		return web.Respond(http.StatusOK), nil
	})

	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error when Store.Save fails, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusInternalServerError {
		t.Fatalf("error status = %d, want 500 when Store.Save fails", webErr.Status)
	}
	if !errors.Is(err, web.ErrTransient) {
		t.Fatalf("errors.Is(err, web.ErrTransient) = false; err = %v", err)
	}
}

// TestMiddleware_NewSession_NoWriteNoCookie verifies Bug 5 (no-mutation path): a new
// session where the handler makes no writes produces no Set-Cookie header.
func TestMiddleware_NewSession_NoWriteNoCookie(t *testing.T) {
	cfg := testMemoryConfig(t)
	mw := Middleware(cfg)

	resp, _ := runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		// Access the session but do not call Set, Flash, Unset, Regenerate, or Destroy.
		_, _ = FromContext(c)
		return web.Respond(http.StatusOK), nil
	})

	if sc := resp.Headers.Get("Set-Cookie"); sc != "" {
		t.Fatalf("Set-Cookie = %q, want empty (no writes on new session)", sc)
	}
}

// TestMiddleware_NewSession_WriteEmitsCookie verifies Bug 5 (mutation path): a new
// session where the handler calls Set does emit a Set-Cookie header.
func TestMiddleware_NewSession_WriteEmitsCookie(t *testing.T) {
	cfg := testMemoryConfig(t)
	mw := Middleware(cfg)

	resp, _ := runRequest(t, mw, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Set("k", "v")
		return web.Respond(http.StatusOK), nil
	})

	if sc := resp.Headers.Get("Set-Cookie"); sc == "" {
		t.Fatal("Set-Cookie absent, want cookie emitted after write on new session")
	}
}

func TestMiddleware_CommitSetsAbsoluteDeadline(t *testing.T) {
	store := &recordingSaveStore{}
	cfg := Config{
		Store: store,
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}

	before := time.Now()
	_, err := runRequest(t, Middleware(cfg), "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Set("k", "v")
		return web.Respond(http.StatusOK), nil
	})
	if err != nil {
		t.Fatalf("Middleware error = %v", err)
	}

	want := before.Add(time.Hour)
	if diff := store.savedAbsolute.Sub(want); diff < 0 || diff > 5*time.Second {
		t.Fatalf("saved absolute = %v, want approximately %v (diff = %v)", store.savedAbsolute, want, diff)
	}
}

func TestMiddleware_PanicDoesNotCommitSession(t *testing.T) {
	cfg := testMemoryConfig(t)
	sessionMW := Middleware(cfg)

	resp1, _ := runRequest(t, sessionMW, "", func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Set("user", "alice")
		return web.Respond(http.StatusOK), nil
	})
	cookie := extractSetCookie(t, resp1)

	handler := web.Chain(func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Set("user", "mallory")
		panic("boom")
	}, recoverMiddleware, sessionMW)

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.Headers.Set("Cookie", cookie)
	resp2, _ := handler(web.NewContext(context.Background(), req))
	if resp2.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp2.Status, http.StatusInternalServerError)
	}
	if sc := resp2.Headers.Get("Set-Cookie"); sc != "" {
		t.Fatalf("Set-Cookie = %q, want empty after panic", sc)
	}

	var gotUser string
	_, _ = runRequest(t, sessionMW, cookie, func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		v, ok, err := Get[string](sess, "user")
		if err != nil || !ok {
			t.Fatalf("Get user after panic: err=%v ok=%v", err, ok)
		}
		gotUser = v
		return web.Respond(http.StatusOK), nil
	})
	if gotUser != "alice" {
		t.Fatalf("user after panic = %q, want alice", gotUser)
	}
}

// cookieValue extracts the cookie value from a Set-Cookie header string for the named cookie.
func cookieValue(setCookie, name string) string {
	parts := strings.Split(setCookie, ";")
	if len(parts) == 0 {
		return ""
	}
	kv := strings.SplitN(strings.TrimSpace(parts[0]), "=", 2)
	if len(kv) != 2 {
		return ""
	}
	if strings.TrimSpace(kv[0]) != name {
		return ""
	}
	return strings.TrimSpace(kv[1])
}

// ---------------------------------------------------------------------------
// G1 — errors.Is for ErrSessionNotFound (not == comparison)
// ---------------------------------------------------------------------------

func (w *wrappedNotFoundStore) Read(_ context.Context, _ string) ([]byte, int64, error) {
	// Wrap ErrSessionNotFound — a bare == check would NOT match this.
	return nil, 0, fmt.Errorf("store layer: %w", ErrSessionNotFound)
}

func (w *wrappedNotFoundStore) Save(_ context.Context, _ string, _ []byte, _ time.Time, _ time.Duration, _ int64) (string, error) {
	return "token", nil
}

func (w *wrappedNotFoundStore) Delete(_ context.Context, _ string) error {
	return nil
}

// TestMiddleware_G1_WrappedErrSessionNotFound verifies that loadSession treats
// a wrapped ErrSessionNotFound identically to a bare one: the middleware must
// create a fresh session rather than return a 500.
func TestMiddleware_G1_WrappedErrSessionNotFound(t *testing.T) {
	cfg := Config{
		Store:          &wrappedNotFoundStore{},
		CookieTemplate: web.Cookie{Name: "sess", Path: "/", HTTPOnly: true},
		TTL:            time.Hour,
	}
	mw := Middleware(cfg)

	var wasNew bool
	_, err := runRequest(t, mw, "sess=unknown-token", func(c *web.Context) (web.Response, error) {
		sess, ok := FromContext(c)
		if !ok {
			t.Fatal("session not in context")
		}
		wasNew = sess.IsNew()
		return web.Respond(200), nil
	})

	if err != nil {
		t.Fatalf("expected no error from middleware when store returns wrapped ErrSessionNotFound; got %v", err)
	}
	if !wasNew {
		t.Fatal("expected a fresh session when store returns wrapped ErrSessionNotFound")
	}
}

func TestCommit_SaveVersionConflictMarkedTransient(t *testing.T) {
	cfg := Config{
		Store: &fakeErrorStore{saveErr: ErrVersionConflict},
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL:            time.Hour,
		MaxCookieBytes: defaultMaxSize,
	}

	sess := newSession()
	sess.token = "existing-token"
	sess.version = 7
	sess.fresh = false
	if err := sess.Set("k", "v"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	resp := web.Respond(http.StatusOK)
	err := commit(context.Background(), &resp, cfg, sess)
	if err == nil {
		t.Fatal("commit returned nil, want ErrVersionConflict")
	}
	if !errors.Is(err, web.ErrTransient) {
		t.Fatalf("errors.Is(err, web.ErrTransient) = false; err = %v", err)
	}
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("errors.Is(err, ErrVersionConflict) = false; err = %v", err)
	}
	if errors.Is(err, web.ErrDependencyTimeout) {
		t.Fatalf("errors.Is(err, web.ErrDependencyTimeout) = true, want false; err = %v", err)
	}
}

func TestCommit_SaveDeadlineMarkedDependencyTimeout(t *testing.T) {
	cfg := Config{
		Store: &fakeErrorStore{saveErr: context.DeadlineExceeded},
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL:            time.Hour,
		MaxCookieBytes: defaultMaxSize,
	}

	sess := newSession()
	if err := sess.Set("k", "v"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	resp := web.Respond(http.StatusOK)
	err := commit(context.Background(), &resp, cfg, sess)
	if err == nil {
		t.Fatal("commit returned nil, want dependency timeout")
	}
	if !errors.Is(err, web.ErrDependencyTimeout) {
		t.Fatalf("errors.Is(err, web.ErrDependencyTimeout) = false; err = %v", err)
	}
	if errors.Is(err, web.ErrTransient) {
		t.Fatalf("errors.Is(err, web.ErrTransient) = true, want false; err = %v", err)
	}
}
