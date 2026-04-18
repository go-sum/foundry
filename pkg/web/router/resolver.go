package router

// Resolver provides lazy route path/URL resolution.
// Closures returned by Path and URL resolve at call time,
// enabling handler wiring before route registration completes.
type Resolver struct {
	rt *Router
}

// NewResolver creates a Resolver backed by rt.
func NewResolver(rt *Router) *Resolver {
	return &Resolver{rt: rt}
}

// Path returns a closure that resolves the pattern for the named route.
// Panics if the name is not registered when the closure is called.
func (r *Resolver) Path(name string) func() string {
	return func() string {
		p, err := r.rt.Pattern(name)
		if err != nil {
			panic(err.Error())
		}
		return p
	}
}

// URL returns a closure that resolves an absolute URL (origin + path)
// for the named route with params substituted.
// Panics if the name is not registered or a required param is missing when called.
func (r *Resolver) URL(origin, name string, params map[string]string) func() string {
	return func() string {
		return origin + r.rt.MustReverse(name, params)
	}
}
