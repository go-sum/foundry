---
title: Web Concurrency, Performance, and Runtime Safety
description: Governing patterns for concurrent, high-throughput, safe Go web applications.
weight: 35
---

# Web Concurrency, Performance, and Runtime Safety

> This guide is the authoritative source for concurrency, performance, and
> runtime safety patterns in Go web applications.
>
> It complements [`ARCHITECTURE_GUIDE.md`](./ARCHITECTURE_GUIDE.md) (project
> structure, wiring, and shutdown), [`DESIGN_PATTERNS.md`](./DESIGN_PATTERNS.md) (handler,
> middleware, error taxonomy, and resilience patterns), and
> [`CODE_REVIEW.md`](./CODE_REVIEW.md) (review checklists).
>
> Read this together with [`CLAUDE.md`](../CLAUDE.md) for behavioral rules.
>
> Use this guide to answer:
>
> - how to safely share state across concurrent HTTP handlers
> - when and how to use goroutines, worker pools, and errgroup
> - how to implement rate limiting at multiple granularities
> - how to detect, prevent, and test for data races
> - which synchronization primitive to choose for a given problem

---

## 0. Prescriptive Intent

This guide is **prescriptive**. It defines the concurrency and safety patterns
that application code follows. Deviations are surfaced in code review and
addressed in the next refactor pass.

---

## 1. Core Concurrency Rules for Web Applications

### Goroutines are cheap but not free

Each goroutine starts with a 2-8 KB stack that grows as needed. Under
unbounded spawning, thousands of goroutines accumulate memory, exhaust file
descriptors, and cause OOM kills. Treat goroutine creation as a resource
allocation decision, not a zero-cost abstraction.

### Every goroutine must have a shutdown path

A goroutine without a termination mechanism is a goroutine leak. Every
goroutine must terminate through one of:

- `context.Context` cancellation
- a signal on a dedicated channel
- a `sync.WaitGroup` completion
- natural return from a bounded operation

### Prefer channels for communication between goroutines

Use channels to pass data and coordinate work between goroutines. Use mutexes
to protect shared state within a single goroutine's critical section. Do not
mix the two for the same concern.

### Use mutexes for state protection

When goroutines share mutable state that does not flow through a channel, protect
it with `sync.Mutex`, `sync.RWMutex`, or `sync/atomic` types. Choose the
narrowest primitive that covers the access pattern.

### Never spawn raw goroutines in HTTP handlers

An HTTP handler that calls `go func() { ... }()` directly creates an
uncontrolled goroutine with no backpressure, no lifecycle management, and no
panic recovery. Use a worker pool, `errgroup`, or an owned background subsystem
instead.

```go
// Do not do this in a handler.
func handleOrder(c echo.Context) error {
    go sendConfirmationEmail(order) // leaked goroutine, no recovery, no backpressure
    return c.JSON(http.StatusCreated, order)
}

// Use a worker pool or queue instead.
func handleOrder(c echo.Context) error {
    if !pool.TrySubmit(func() { sendConfirmationEmail(order) }) {
        slog.WarnContext(ctx, "worker pool full, email deferred")
    }
    return c.JSON(http.StatusCreated, order)
}
```

### Prefer synchronous code first

Do not add goroutines, workers, or async dispatch until there is a concrete
correctness, latency, or throughput requirement. Synchronous code is simpler
to reason about, test, and debug. Concurrency is justified when measured
performance or architectural needs demand it.

---

## 2. Handler Safety

Every HTTP request runs in its own goroutine. Any mutable state on the server
struct, in package-level variables, or in shared closures is accessed
concurrently by every in-flight request.

### What is safe without synchronization

| Resource | Why |
|---|---|
| Request-scoped locals | Each handler invocation has its own stack frame |
| `*sql.DB` / `*pgxpool.Pool` | Internal connection pooling with its own synchronization |
| Immutable config loaded at startup | No writes after initialization |
| `context.Context` values | Immutable once set; read-only in downstream code |

### What requires synchronization

| Resource | Risk | Solution |
|---|---|---|
| `map` on server struct | Concurrent read/write panics | `sync.RWMutex` or `sync.Map` |
| `[]T` on server struct | Concurrent append corrupts backing array | `sync.Mutex` |
| Counter or flag | Lost updates, torn reads | `atomic.Int64`, `atomic.Bool` |
| In-memory cache | Stale reads, corrupt entries | `sync.RWMutex` for read-heavy; `sync.Mutex` for write-heavy |
| Lazy-initialized singleton | Double initialization, partial state | `sync.Once` or `sync.OnceValue` |

### Atomic types for simple flags and counters

Use `atomic.Bool`, `atomic.Int64`, `atomic.Int32`, and `atomic.Pointer[T]` for
single-value state that does not require multi-field consistency.

```go
type Server struct {
    healthy    atomic.Bool
    reqCount   atomic.Int64
}

func (s *Server) handleHealth(c echo.Context) error {
    if !s.healthy.Load() {
        return c.NoContent(http.StatusServiceUnavailable)
    }
    s.reqCount.Add(1)
    return c.NoContent(http.StatusOK)
}
```

### Mutex for complex shared state

Use `sync.RWMutex` when reads vastly outnumber writes. Use `sync.Mutex` when
the read/write ratio is balanced or writes are frequent.

```go
type Server struct {
    mu    sync.RWMutex
    cache map[string]CachedItem
}

func (s *Server) handleGetCached(c echo.Context) error {
    key := c.Param("key")
    s.mu.RLock()
    item, ok := s.cache[key]
    s.mu.RUnlock()
    if !ok {
        return c.NoContent(http.StatusNotFound)
    }
    return c.JSON(http.StatusOK, item)
}

func (s *Server) handleInvalidate(c echo.Context) error {
    key := c.Param("key")
    s.mu.Lock()
    delete(s.cache, key)
    s.mu.Unlock()
    return c.NoContent(http.StatusOK)
}
```

---

## 3. Worker Pool Pattern

### Why worker pools

Worker pools solve three problems that raw goroutines do not:

1. **Bounded concurrency** -- a fixed number of workers prevents OOM under load
2. **Backpressure** -- a full job queue signals the caller to shed load
3. **Graceful shutdown** -- workers drain in-flight work before the process exits

### Implementation

```go
type WorkerPool struct {
    jobs chan func()
    wg   sync.WaitGroup
}

func NewWorkerPool(workers, queueSize int) *WorkerPool {
    p := &WorkerPool{
        jobs: make(chan func(), queueSize),
    }
    p.wg.Add(workers)
    for range workers {
        go p.worker()
    }
    return p
}

func (p *WorkerPool) worker() {
    defer p.wg.Done()
    for job := range p.jobs {
        func() {
            defer func() {
                if r := recover(); r != nil {
                    slog.Error("panic.goroutine",
                        "panic", r,
                        "stack", string(debug.Stack()),
                    )
                }
            }()
            job()
        }()
    }
}

// Submit blocks until the job is accepted or the pool is shut down.
func (p *WorkerPool) Submit(job func()) {
    p.jobs <- job
}

// TrySubmit returns false immediately if the queue is full.
func (p *WorkerPool) TrySubmit(job func()) bool {
    select {
    case p.jobs <- job:
        return true
    default:
        return false
    }
}

// Shutdown closes the job channel and waits for in-flight work to complete.
func (p *WorkerPool) Shutdown() {
    close(p.jobs)
    p.wg.Wait()
}
```

### Sizing guidance

| Workload type | Worker count | Rationale |
|---|---|---|
| CPU-bound (image resize, hashing) | `runtime.NumCPU()` | More workers than cores adds context-switch overhead |
| I/O-bound (HTTP calls, DB queries) | 10x-100x CPU count | Workers spend most time waiting; more workers keep throughput high |
| Mixed | Profile and measure | Start with 2x CPU count and adjust based on p99 latency |

### Queue sizing and backpressure

- **Small queue** (0-10): fast backpressure signal; callers learn immediately
  when the pool is saturated
- **Large queue** (100-1000): absorbs traffic bursts; risk of high memory usage
  and delayed processing if workers are slow

For HTTP handlers, prefer a small queue with `TrySubmit`. Return 503 Service
Unavailable when the pool is full rather than blocking the request goroutine:

```go
func (s *Server) handleWebhook(c echo.Context) error {
    ctx := c.Request().Context()
    payload := extractPayload(c)

    if !s.pool.TrySubmit(func() { s.processWebhook(payload) }) {
        slog.WarnContext(ctx, "worker pool full, rejecting webhook")
        return c.NoContent(http.StatusServiceUnavailable)
    }
    return c.NoContent(http.StatusAccepted)
}
```

### Graceful shutdown integration

Stop accepting new HTTP connections first, then drain the worker pool, then
close downstream resources:

```go
func run(ctx context.Context) error {
    pool := NewWorkerPool(runtime.NumCPU()*10, 100)

    srv := &http.Server{Addr: ":8080", Handler: router}
    go func() {
        <-ctx.Done()
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        srv.Shutdown(shutdownCtx) // stop accepting new requests
    }()

    err := srv.ListenAndServe()
    pool.Shutdown() // drain in-flight background work
    db.Close()      // close downstream resources last
    return err
}
```

### Error handling in workers

Workers must never silently discard errors. Log all failures with structured
logging:

```go
func (s *Server) processWebhook(payload WebhookPayload) {
    ctx := context.Background() // request context is already cancelled
    if err := s.webhookSvc.Deliver(ctx, payload); err != nil {
        slog.ErrorContext(ctx, "webhook delivery failed",
            "webhook_id", payload.ID,
            "err", err,
        )
    }
}
```

### Context propagation in workers

Workers must use their own context, not the request context. The request
context is cancelled as soon as the HTTP response is sent. Extract trace IDs,
user IDs, and other correlation data before submitting:

```go
func (s *Server) handleOrder(c echo.Context) error {
    ctx := c.Request().Context()
    orderID := c.Param("id")
    userID := auth.UserID(ctx)
    traceID := web.RequestID(c)

    s.pool.Submit(func() {
        bgCtx := context.Background()
        s.sendReceipt(bgCtx, orderID, userID, traceID)
    })
    return c.NoContent(http.StatusAccepted)
}
```

### Retry with exponential backoff

For transient failures in worker tasks, use exponential backoff with jitter.
See [`DESIGN_PATTERNS.md` section 5g](./DESIGN_PATTERNS.md) for the canonical backoff
policy and `pkg/web/retry` for the reference implementation.

---

## 4. errgroup as a Simpler Alternative

`errgroup.Group` from `golang.org/x/sync/errgroup` is the preferred tool for
fan-out/fan-in within a single request. It propagates errors, cancels on first
failure, and supports bounded concurrency.

### Basic usage

```go
func (s *Server) handleDashboard(c echo.Context) error {
    ctx := c.Request().Context()
    g, ctx := errgroup.WithContext(ctx)

    var stats Stats
    var recent []Order
    var alerts []Alert

    g.Go(func() error {
        var err error
        stats, err = s.statsSvc.Get(ctx)
        return err
    })
    g.Go(func() error {
        var err error
        recent, err = s.orderSvc.ListRecent(ctx, 10)
        return err
    })
    g.Go(func() error {
        var err error
        alerts, err = s.alertSvc.ListActive(ctx)
        return err
    })

    if err := g.Wait(); err != nil {
        return fmt.Errorf("dashboard: %w", err)
    }
    return c.JSON(http.StatusOK, DashboardResponse{stats, recent, alerts})
}
```

### Bounded concurrency with SetLimit

```go
func (s *Server) handleBatchProcess(c echo.Context) error {
    ctx := c.Request().Context()
    items := parseItems(c)

    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(10) // at most 10 concurrent goroutines

    for _, item := range items {
        g.Go(func() error {
            return s.processSvc.Process(ctx, item)
        })
    }
    return g.Wait()
}
```

### Panic recovery in errgroup goroutines

`errgroup` propagates returned errors but does **not** recover panics. An
unrecovered panic in an errgroup goroutine crashes the entire process. Install
a `recover` inside each goroutine:

```go
g.Go(func() (err error) {
    defer func() {
        if r := recover(); r != nil {
            slog.Error("panic.goroutine",
                "panic", r,
                "stack", string(debug.Stack()),
            )
            err = fmt.Errorf("recovered panic: %v", r)
        }
    }()
    return s.riskyOperation(ctx)
})
```

### When to use errgroup vs worker pool

| Scenario | Tool | Reason |
|---|---|---|
| Parallel queries within a single request | errgroup | Scoped to request lifetime; context cancellation propagates automatically |
| Fan-out to multiple APIs then combine results | errgroup | Need all results before responding; first error cancels remaining |
| Batch processing bounded items within request | errgroup with SetLimit | Bounded concurrency with automatic error collection |
| Background tasks that outlive the request | Worker pool | Request context is already cancelled; need independent lifecycle |
| Fire-and-forget work (emails, webhooks, analytics) | Worker pool | No result needed in the response; needs backpressure and shutdown |
| Long-running background processing | Worker pool or queue | Must survive process restarts; needs its own lifecycle management |

---

## 5. Rate Limiting

### Token bucket algorithm

The standard approach is `golang.org/x/time/rate`, which implements a token
bucket limiter. Each request consumes one token; tokens refill at a configured
rate. The burst parameter controls the maximum number of tokens available at
any instant.

### Global rate limiting middleware

Apply a global limiter to protect the server from aggregate overload:

```go
func RateLimit(rps float64, burst int) echo.MiddlewareFunc {
    limiter := rate.NewLimiter(rate.Limit(rps), burst)
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            if !limiter.Allow() {
                c.Response().Header().Set("Retry-After", "1")
                return c.NoContent(http.StatusTooManyRequests)
            }
            return next(c)
        }
    }
}
```

### Per-IP rate limiting

Per-IP limiting prevents a single client from consuming the global budget.

```go
type trackedLimiter struct {
    limiter  *rate.Limiter
    lastSeen time.Time
}

type IPRateLimiter struct {
    mu       sync.RWMutex
    limiters map[string]*trackedLimiter
    rps      rate.Limit
    burst    int
}

func NewIPRateLimiter(rps float64, burst int) *IPRateLimiter {
    rl := &IPRateLimiter{
        limiters: make(map[string]*trackedLimiter),
        rps:      rate.Limit(rps),
        burst:    burst,
    }
    go rl.cleanup()
    return rl
}

func (rl *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
    rl.mu.RLock()
    tl, ok := rl.limiters[ip]
    rl.mu.RUnlock()
    if ok {
        rl.mu.Lock()
        tl.lastSeen = time.Now()
        rl.mu.Unlock()
        return tl.limiter
    }

    // Double-check after acquiring write lock.
    rl.mu.Lock()
    defer rl.mu.Unlock()
    if tl, ok := rl.limiters[ip]; ok {
        tl.lastSeen = time.Now()
        return tl.limiter
    }
    limiter := rate.NewLimiter(rl.rps, rl.burst)
    rl.limiters[ip] = &trackedLimiter{limiter: limiter, lastSeen: time.Now()}
    return limiter
}

func (rl *IPRateLimiter) cleanup() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    for range ticker.C {
        rl.mu.Lock()
        cutoff := time.Now().Add(-10 * time.Minute)
        for ip, tl := range rl.limiters {
            if tl.lastSeen.Before(cutoff) {
                delete(rl.limiters, ip)
            }
        }
        rl.mu.Unlock()
    }
}
```

### IP extraction

Extract the client IP from `X-Forwarded-For` or `X-Real-IP` headers **only**
when the application runs behind a trusted reverse proxy. When exposed
directly, use `c.RealIP()` or `r.RemoteAddr`. Trusting forwarded headers
without a trusted proxy allows clients to spoof their IP and bypass rate
limits.

### Cleaning up stale limiters

The `trackedLimiter` pattern above tracks `lastSeen` per IP. A background
ticker removes entries that have not been seen within the cleanup window. This
prevents unbounded map growth from diverse client IPs.

### Rate limiting by API key or user

For authenticated endpoints, rate limit by user identity or API key rather than
IP. Tier-based limits allow differentiated service levels:

```go
type KeyRateLimiter struct {
    mu       sync.RWMutex
    limiters map[string]*trackedLimiter
    tiers    map[string]TierConfig
}

type TierConfig struct {
    RPS   float64
    Burst int
}

var defaultTiers = map[string]TierConfig{
    "free":       {RPS: 10, Burst: 20},
    "pro":        {RPS: 100, Burst: 200},
    "enterprise": {RPS: 1000, Burst: 2000},
}

func (krl *KeyRateLimiter) GetLimiter(apiKey, tier string) *rate.Limiter {
    krl.mu.RLock()
    tl, ok := krl.limiters[apiKey]
    krl.mu.RUnlock()
    if ok {
        krl.mu.Lock()
        tl.lastSeen = time.Now()
        krl.mu.Unlock()
        return tl.limiter
    }

    cfg := krl.tiers[tier]
    krl.mu.Lock()
    defer krl.mu.Unlock()
    if tl, ok := krl.limiters[apiKey]; ok {
        tl.lastSeen = time.Now()
        return tl.limiter
    }
    limiter := rate.NewLimiter(rate.Limit(cfg.RPS), cfg.Burst)
    krl.limiters[apiKey] = &trackedLimiter{limiter: limiter, lastSeen: time.Now()}
    return limiter
}
```

### Proper 429 responses

A 429 Too Many Requests response must include headers that tell the client
when to retry and what the limits are:

```go
func rateLimitResponse(c echo.Context, retryAfter time.Duration, limit, remaining int) error {
    h := c.Response().Header()
    h.Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
    h.Set("X-RateLimit-Limit", strconv.Itoa(limit))
    h.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
    return c.NoContent(http.StatusTooManyRequests)
}
```

### Combining rate limiters

Apply per-IP rate limiting inside a global rate limiter for defense in depth.
The global limiter protects aggregate server capacity; the per-IP limiter
prevents a single client from monopolizing it:

```go
func CombinedRateLimit(globalRPS float64, globalBurst int, ipRPS float64, ipBurst int) echo.MiddlewareFunc {
    global := rate.NewLimiter(rate.Limit(globalRPS), globalBurst)
    perIP := NewIPRateLimiter(ipRPS, ipBurst)

    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            if !global.Allow() {
                c.Response().Header().Set("Retry-After", "1")
                return c.NoContent(http.StatusTooManyRequests)
            }
            ip := c.RealIP()
            if !perIP.GetLimiter(ip).Allow() {
                c.Response().Header().Set("Retry-After", "1")
                return c.NoContent(http.StatusTooManyRequests)
            }
            return next(c)
        }
    }
}
```

---

## 6. Race Detection and Prevention

### Running the race detector

The Go race detector instruments memory accesses at compile time and reports
data races at runtime. It finds races that actually execute during the test
run, not static analysis of all possible races.

```bash
# Run tests with race detection.
go test -race ./...

# Build a race-instrumented binary for manual testing.
go build -race -o app-race ./cmd/server
```

The race detector adds approximately 5-15x CPU overhead and 5-10x memory
overhead. Do not run it in production. Run it in CI and during development.

### What it catches vs what it does not

| Detected | Not detected |
|---|---|
| Concurrent read/write to the same memory location | Logical races (correct synchronization, wrong business logic) |
| Concurrent write/write to the same memory location | Deadlocks |
| Missing mutex around map access | Starvation |
| Missing atomic for flag/counter updates | Livelock |
| | Races in code paths not exercised by the test |

### Common web handler races

**Shared map without lock:**

```go
// Race: concurrent handlers read and write s.cache.
type Server struct {
    cache map[string]string
}

// Fix: protect with sync.RWMutex or use sync.Map.
type Server struct {
    mu    sync.RWMutex
    cache map[string]string
}
```

**Incrementing counter without atomic:**

```go
// Race: s.count++ is a read-modify-write, not atomic.
type Server struct {
    count int64
}

// Fix: use atomic.Int64.
type Server struct {
    count atomic.Int64
}
```

**Slice append without lock:**

```go
// Race: concurrent append may corrupt the backing array.
type Server struct {
    events []Event
}

// Fix: protect with sync.Mutex.
type Server struct {
    mu     sync.Mutex
    events []Event
}
```

**Lazy initialization without sync.Once:**

```go
// Race: multiple goroutines may initialize simultaneously.
func (s *Server) getClient() *http.Client {
    if s.client == nil {
        s.client = &http.Client{Timeout: 10 * time.Second}
    }
    return s.client
}

// Fix: use sync.Once or sync.OnceValue.
func (s *Server) getClient() *http.Client {
    return s.clientOnce.Do(func() *http.Client {
        return &http.Client{Timeout: 10 * time.Second}
    })
}
```

**Read-modify-write on struct field:**

```go
// Race: s.ready may be read while another goroutine writes it.
type Server struct {
    ready bool
}

// Fix: use atomic.Bool.
type Server struct {
    ready atomic.Bool
}
```

### Synchronization primitive decision table

| Access pattern | Primitive | When to use |
|---|---|---|
| Single counter (increment, load) | `atomic.Int64` | High-frequency counting with no multi-field consistency requirement |
| Single boolean flag | `atomic.Bool` | Feature flags, health status, shutdown signal |
| Read-heavy cache (reads >> writes) | `sync.RWMutex` | Multiple concurrent readers, infrequent writers |
| Write-heavy map | `sync.Mutex` | Frequent writes; RWMutex overhead not justified |
| Cross-goroutine data flow | Channels | Producer-consumer, fan-out/fan-in, signaling |
| One-time initialization | `sync.Once` or `sync.OnceValue` | Lazy singletons, expensive setup |
| Write-once-read-many or disjoint keys | `sync.Map` | Keys are stable after initial write; no iteration needed |
| Typed lazy singleton (Go 1.21+) | `sync.OnceValue[T]` | Type-safe replacement for `sync.Once` + package variable |

### sync.Mutex vs sync.RWMutex

Use `sync.RWMutex` only when reads significantly outnumber writes. `RWMutex`
has higher per-operation overhead than `Mutex` due to internal bookkeeping. If
the critical section is short and writes are frequent, a plain `Mutex` is
faster.

### sync.Map vs mutex-protected map

`sync.Map` is optimized for two patterns:

1. Keys are written once and read many times (stable key set)
2. Multiple goroutines read, write, and overwrite entries for disjoint key sets

For all other patterns, a `sync.Mutex`- or `sync.RWMutex`-protected
`map[K]V` is simpler, more type-safe, and often faster.

Do not use `sync.Map` as a default replacement for `map` -- use it only when
profiling shows contention on a mutex-protected map, or the access pattern
matches one of the two cases above.

### Testing for race conditions

Write tests that exercise concurrent access to confirm synchronization is
correct:

```go
func TestServer_ConcurrentCacheAccess(t *testing.T) {
    srv := NewServer()
    var wg sync.WaitGroup

    // Concurrent writers.
    for i := range 100 {
        wg.Add(1)
        go func() {
            defer wg.Done()
            key := fmt.Sprintf("key-%d", i)
            srv.SetCache(key, "value")
        }()
    }

    // Concurrent readers.
    for range 100 {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _ = srv.GetCache("key-50")
        }()
    }

    wg.Wait()
}
```

Write HTTP-level race tests using `httptest`:

```go
func TestHandler_ConcurrentRequests(t *testing.T) {
    srv := setupTestServer(t)
    ts := httptest.NewServer(srv.Handler())
    defer ts.Close()

    var wg sync.WaitGroup
    for range 50 {
        wg.Add(1)
        go func() {
            defer wg.Done()
            resp, err := http.Get(ts.URL + "/api/stats")
            if err != nil {
                t.Error(err)
                return
            }
            resp.Body.Close()
        }()
    }
    wg.Wait()
}
```

### CI integration

Run the race detector in CI on every pull request:

```bash
go test -race -count=1 ./...
```

Set `GORACE="halt_on_error=1"` to fail the CI job on the first detected race
rather than continuing and potentially masking subsequent races:

```bash
GORACE="halt_on_error=1" go test -race -count=1 ./...
```

Use `-count=1` to disable test caching -- cached results do not re-run the
race detector.

---

## 7. Critical Concurrency Anti-Patterns

### Goroutine leak: no way to stop

A goroutine that blocks forever on a channel or loop without checking a
cancellation signal leaks for the lifetime of the process.

```go
// Anti-pattern: no exit path.
go func() {
    for {
        item := <-ch
        process(item)
    }
}()

// Fix: use context cancellation.
go func() {
    for {
        select {
        case <-ctx.Done():
            return
        case item := <-ch:
            process(item)
        }
    }
}()
```

### Unbounded channel send: blocks forever

A send to an unbounded or full channel blocks the goroutine indefinitely if no
receiver drains it.

```go
// Anti-pattern: blocks if ch is full and no one reads.
ch <- result

// Fix: use select with context or default.
select {
case ch <- result:
case <-ctx.Done():
    return ctx.Err()
}
```

### Closing a channel multiple times

Closing an already-closed channel panics. Only the sender should close a
channel, and it should close it exactly once.

```go
// Anti-pattern: multiple goroutines may close.
close(ch)

// Fix: use sync.Once to guarantee single close.
var closeOnce sync.Once
closeOnce.Do(func() { close(ch) })
```

### Race condition on shared state

Any mutable state accessed by multiple goroutines without synchronization is a
data race. See section 6 for the full decision table.

### Missing WaitGroup

The caller returns before spawned goroutines complete, leading to lost work or
use-after-free on resources the goroutines depend on.

```go
// Anti-pattern: function returns while goroutines still run.
for _, item := range items {
    go process(item)
}
return // goroutines may still be running

// Fix: wait for all goroutines.
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func() {
        defer wg.Done()
        process(item)
    }()
}
wg.Wait()
return
```

### Context not propagated

Goroutines that ignore the parent context continue working after the caller
has cancelled, wasting resources and potentially writing stale results.

```go
// Anti-pattern: ignores cancellation.
go func() {
    result := expensiveQuery(context.Background()) // should use ctx
    ch <- result
}()

// Fix: propagate the parent context.
go func() {
    result := expensiveQuery(ctx)
    select {
    case ch <- result:
    case <-ctx.Done():
    }
}()
```

### Unbounded goroutine spawning in handlers

Spawning a goroutine per request under load creates thousands of goroutines
with no backpressure. Use a worker pool (section 3) or `errgroup` with
`SetLimit` (section 4).

### Using context.Background() instead of request context in handlers

Handlers must use `c.Request().Context()`, not `context.Background()`. The
request context carries deadlines, cancellation, and trace propagation. Using
`context.Background()` severs all of those.

```go
// Anti-pattern.
result, err := svc.Query(context.Background(), id)

// Fix.
result, err := svc.Query(c.Request().Context(), id)
```

### Goroutine leak from missing channel receiver

A goroutine that sends to a channel no one reads blocks forever. Use a buffered
channel of size 1 when the result may not be consumed:

```go
// Anti-pattern: blocks if caller times out and stops reading.
ch := make(chan Result)
go func() { ch <- computeResult() }()

// Fix: buffer of 1 lets the goroutine complete and exit.
ch := make(chan Result, 1)
go func() { ch <- computeResult() }()
```

### Using time.Sleep for coordination

`time.Sleep` is not a synchronization primitive. It introduces flaky timing
dependencies and slows down tests.

```go
// Anti-pattern.
go updateCache()
time.Sleep(100 * time.Millisecond) // "wait" for cache update
readCache()

// Fix: use sync primitives.
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    updateCache()
}()
wg.Wait()
readCache()
```

---

## 8. Modern Go Concurrency Features

### sync.OnceValue and sync.OnceFunc (Go 1.21+)

`sync.OnceValue[T]` and `sync.OnceFunc` are type-safe replacements for the
`sync.Once` + package variable pattern. They eliminate the need for a separate
variable to store the result.

```go
// Before (sync.Once + variable).
var (
    configOnce sync.Once
    config     *Config
)

func GetConfig() *Config {
    configOnce.Do(func() {
        config = loadConfig()
    })
    return config
}

// After (sync.OnceValue).
var GetConfig = sync.OnceValue(func() *Config {
    return loadConfig()
})
```

`sync.OnceFunc` is the equivalent for functions that return no value:

```go
var initMetrics = sync.OnceFunc(func() {
    prometheus.MustRegister(requestCounter, latencyHistogram)
})
```

### errgroup.SetLimit (Go 1.20+)

`errgroup.Group.SetLimit(n)` caps the number of goroutines that can run
concurrently. This eliminates the need for a manual semaphore channel when
using errgroup for bounded fan-out. See section 4 for examples.

### Range over integer (Go 1.22+)

```go
// Before.
for i := 0; i < n; i++ { ... }

// After.
for i := range n { ... }
```

### Loop variable semantics (Go 1.22+)

Starting with Go 1.22, each iteration of a `for` loop creates a new variable.
The classic closure-capture bug no longer applies:

```go
// Before Go 1.22: all goroutines capture the same variable.
for _, item := range items {
    go func() {
        process(item) // bug: all goroutines see the last item
    }()
}

// Go 1.22+: each iteration has its own 'item'.
for _, item := range items {
    go func() {
        process(item) // correct: each goroutine has its own copy
    }()
}
```

---

## 9. Decision Checklist

Before merging code that introduces concurrency, confirm:

- every goroutine has an explicit shutdown path (context, channel, or WaitGroup)
- no raw `go func()` calls exist inside HTTP handlers
- shared mutable state is protected by the correct primitive (see section 6 table)
- worker pools use `TrySubmit` in handlers and return 503 when full
- errgroup goroutines install `recover` for panic safety
- background workers use their own context, not the request context
- trace IDs and correlation data are extracted before submitting to a worker pool
- rate limiters clean up stale entries to prevent unbounded map growth
- 429 responses include `Retry-After` headers
- CI runs `go test -race -count=1 ./...` with `GORACE="halt_on_error=1"`
- no `time.Sleep` is used for goroutine coordination
- no `context.Background()` is used where a request context is available
- channel direction is specified in function parameters (`chan<-`, `<-chan`)
- channels that may not be read use a buffer of 1 to prevent goroutine leaks
- `sync.Once` or `sync.OnceValue` is used for lazy initialization, not manual nil checks

---

## 10. Sources

- [`DESIGN_PATTERNS.md`](./DESIGN_PATTERNS.md) -- retry, circuit breaker, and resilience patterns
- Go Concurrency Patterns: <https://go.dev/blog/pipelines>
- Effective Go - Concurrency: <https://go.dev/doc/effective_go#concurrency>
- Go Data Race Detector: <https://go.dev/doc/articles/race_detector>
- golang.org/x/sync/errgroup: <https://pkg.go.dev/golang.org/x/sync/errgroup>
- golang.org/x/time/rate: <https://pkg.go.dev/golang.org/x/time/rate>
- Go 1.21 Release Notes (sync.OnceValue): <https://go.dev/doc/go1.21>
- Go 1.22 Release Notes (loop variable semantics): <https://go.dev/doc/go1.22>
