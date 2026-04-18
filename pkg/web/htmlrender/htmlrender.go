// Package htmlrender provides a web.Renderer backed by html/template.
package htmlrender

import (
	"bytes"
	"html/template"
	"io/fs"
	"sync"

	"github.com/go-sum/web"
)

// Config configures the Renderer.
type Config struct {
	// FS is the filesystem containing template files.
	FS fs.FS
	// Pattern is the glob pattern passed to template.ParseFS (e.g., "templates/*.html").
	Pattern string
	// FuncMap provides additional template functions available to all templates.
	FuncMap template.FuncMap
	// DevMode reloads templates on every Render call. Use during development only.
	DevMode bool
}

// Renderer implements web.Renderer using html/template.
// Create with New; zero value is not usable.
type Renderer struct {
	cfg  Config
	tmpl *template.Template
	mu   sync.RWMutex
}

// New creates a Renderer from cfg. Parses templates immediately.
// Returns an error if the initial template parse fails.
func New(cfg Config) (*Renderer, error) {
	tmpl, err := parse(cfg)
	if err != nil {
		return nil, err
	}
	return &Renderer{cfg: cfg, tmpl: tmpl}, nil
}

func parse(cfg Config) (*template.Template, error) {
	t := template.New("").Funcs(cfg.FuncMap)
	return t.ParseFS(cfg.FS, cfg.Pattern)
}

// Render executes the named template with data and returns an HTML response.
func (r *Renderer) Render(c *web.Context, status int, name string, data any) (web.Response, error) {
	if r.cfg.DevMode {
		r.mu.Lock()
		tmpl, err := parse(r.cfg)
		if err != nil {
			r.mu.Unlock()
			return web.Response{}, web.ErrInternal(err)
		}
		r.tmpl = tmpl
		r.mu.Unlock()
	}

	r.mu.RLock()
	tmpl := r.tmpl
	r.mu.RUnlock()

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	return web.HTMLBytes(status, buf.Bytes()), nil
}
