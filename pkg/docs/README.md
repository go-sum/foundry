+++
title = "Documentation library"
description = "HTTP handler for serving Hugo-generated documentation and a CLI for scaffolding and building the docs site."
weight = 20
+++

# docs

`github.com/go-sum/docs` serves Hugo-generated documentation over HTTP and provides a CLI tool for scaffolding and building the documentation site. The HTTP handler mounts at a configurable base path (default `/docs`) and serves pre-built static files from a `public/doc/` directory with content-type detection, cache control, and a custom 404 page. The CLI tool scaffolds a `.docs/` Hugo source skeleton and compiles it into the output directory. The `Routes()` function provides self-contained route registration that creates the handler and wires both routes in a single call.

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| [go-sum/web] | v0.0.0 | HTTP primitives (`web.Context`, `web.Response`, `web.Handler`) |
| [Cobra] | v1.10.2 | CLI command framework for `docs init` and `docs build` |

## Features

- `Handler` type that serves Hugo-built documentation pages and assets
- `Config` struct for configuring base path, cache headers, and public directory
- `DefaultConfig()` for sensible out-of-the-box settings
- `Routes()` function for self-contained route registration with a single call
- Automatic `index.html` resolution for clean URL paths (e.g., `/docs/guide` serves `guide/index.html`)
- Differentiated `Cache-Control` headers: assets cached for one hour, HTML pages served with `no-cache`
- Custom `404.html` fallback page for missing documentation routes
- Path traversal prevention via `..` rejection
- Content-type detection from file extensions with fallback to content sniffing
- CLI scaffolding command (`docs init`) that generates a complete `.docs/` Hugo site skeleton
- CLI build command (`docs build`) that invokes Hugo to compile documentation into `public/doc/`

---

## Installation

### Library (HTTP handler)

```bash
go get github.com/go-sum/docs
```

### CLI tool

Run directly from a project that vendors the module:

```bash
go run github.com/go-sum/docs/cli build
```

---

## Integration

This section walks through adding documentation to a foundry application from scratch.

### Step 1: Add the dependency

Add `github.com/go-sum/docs` to your `go.mod`:

```go
require github.com/go-sum/docs v0.0.0
```

For local workspace development, add a replace directive pointing at the package:

```go
replace github.com/go-sum/docs => ../pkg/docs
```

### Step 2: Scaffold the Hugo site

Run the `init` command from the project root to generate the `.docs/` directory:

```bash
go run github.com/go-sum/docs/cli init
```

This creates a complete Hugo site skeleton with layouts, CSS, JavaScript, and a starter content page. Edit `.docs/hugo.toml` to set the site title and description, then add markdown files under `.docs/content/`.

### Step 3: Register routes

In your route registration function, use `Routes()` to wire up both the index and wildcard routes in a single call:

```go
import (
    "github.com/go-sum/docs"
    "github.com/go-sum/web/router"
)

// In your route registration function:
router.Register(rt, docs.Routes(docs.DefaultConfig("public"))...)
```

`DefaultConfig("public")` creates a `Config` with the base path set to `/docs`, one-hour asset caching, and `no-cache` for HTML pages. The `"public"` argument specifies the top-level public directory; the handler serves files from the `doc/` subdirectory within it.

### Step 4: Add a build task

Add a `build:docs` task to your `Taskfile.yml`:

```yaml
build:docs:
  desc: "Build Hugo documentation"
  cmd: '{{.GO}} run github.com/go-sum/docs/cli build'
```

### Step 5: Build and run

Build the documentation site, then start the application:

```bash
task build:docs
```

The built documentation is now served at `/docs` when the application starts.

---

## HTTP Handler

### Config

The `Config` struct controls how the handler serves documentation files.

```go
type Config struct {
    PublicDir         string
    BasePath          string
    AssetCacheControl string
    PageCacheControl  string
}
```

| Field | Type | Description |
|-------|------|-------------|
| `PublicDir` | `string` | Top-level public directory containing the `doc/` subdirectory (e.g., `"public"`) |
| `BasePath` | `string` | URL prefix where documentation is mounted (e.g., `"/docs"`) |
| `AssetCacheControl` | `string` | `Cache-Control` header value for static assets (CSS, JS, images) |
| `PageCacheControl` | `string` | `Cache-Control` header value for HTML pages |

### DefaultConfig

`DefaultConfig(publicDir string)` returns a `Config` with sensible defaults:

```go
cfg := docs.DefaultConfig("public")
// cfg.PublicDir         = "public"
// cfg.BasePath          = "/docs"
// cfg.AssetCacheControl = "public, max-age=3600"
// cfg.PageCacheControl  = "no-cache"
```

### NewHandler

`NewHandler(cfg Config)` creates a `Handler` that serves documentation using the provided configuration.

```go
handler := docs.NewHandler(docs.DefaultConfig("public"))
```

### Handler.Serve

`Serve` is a `web.Handler` that serves a documentation page or asset for the current request. It reads the `path` parameter from the route context, resolves it to a file under `<PublicDir>/doc/`, and returns a `web.Response` with the appropriate status code, content type, and cache headers.

### Routes

`Routes(cfg Config)` returns a slice of `router.Node` values that register both the index and wildcard documentation routes under the configured base path. This is the recommended way to wire up documentation:

```go
router.Register(rt, docs.Routes(docs.DefaultConfig("public"))...)
```

The function creates a route group at `cfg.BasePath` containing two named routes:

| Route Name | Pattern | Purpose |
|-----------|---------|---------|
| `docs.index` | `GET <basePath>` | Documentation root page |
| `docs.show` | `GET <basePath>/{path...}` | All nested pages and assets |

### Path Resolution Rules

The handler resolves request paths to files under the `<PublicDir>/doc/` root using these rules:

| Request Path | Resolved File | Classification |
|-------------|---------------|----------------|
| `/docs` | `public/doc/index.html` | HTML page |
| `/docs/` | `public/doc/index.html` | HTML page |
| `/docs/guide` | `public/doc/guide/index.html` | HTML page |
| `/docs/guide/setup` | `public/doc/guide/setup/index.html` | HTML page |
| `/docs/css/main.css` | `public/doc/css/main.css` | Asset |
| `/docs/js/theme.js` | `public/doc/js/theme.js` | Asset |

Paths with a file extension are treated as assets. Paths without an extension are treated as HTML pages and resolve to the `index.html` file within the corresponding directory.

### Cache-Control Behaviour

| Content Type | Cache-Control Header |
|-------------|---------------------|
| Assets (paths with a file extension) | `public, max-age=3600` (1 hour) |
| HTML pages (paths without a file extension) | `no-cache` |

### Custom 404 Fallback

When a requested HTML page does not exist, the handler looks for a `404.html` file at the documentation root (`<PublicDir>/doc/404.html`). If found, it is served with HTTP status `404` and the correct `text/html` content type. If no custom 404 page exists, the handler returns a standard not-found error.

Missing assets (paths with a file extension) always return a not-found error without the custom 404 page.

### Path Traversal Security

Requests containing `..` anywhere in the path are rejected immediately with a not-found response. This prevents directory traversal attacks that attempt to read files outside the documentation root.

---

## CLI Tool

The `docs` CLI provides two subcommands for managing a Hugo documentation site.

```
docs
  init    Scaffold a .docs/ Hugo source directory
  build   Build Hugo documentation
```

### `docs init`

Scaffolds a barebones `.docs/` directory in the current working directory. The scaffolded directory contains Hugo layouts, CSS, JavaScript, and a starter content page ready to build.

```bash
go run github.com/go-sum/docs/cli init
```

**Behaviour:**

- Fails if `.docs/` already exists (prevents accidental overwrites)
- Copies the embedded template directory to `.docs/`
- Prints next-steps guidance on completion:

```
created .docs/
next steps:
  edit .docs/hugo.toml to set the title
  add markdown files under .docs/content/
  go run ./pkg/docs/cli build
```

### `docs build`

Invokes Hugo to compile the documentation source into the output directory. Removes any stale output before building.

```bash
go run github.com/go-sum/docs/cli build
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--source` | `string` | `.docs` | Hugo source directory |
| `--destination` | `string` | `public/doc` | Output directory for built documentation |

**Behaviour:**

- Resolves `--destination` to an absolute path
- Removes the existing destination directory (cleans stale output)
- Creates the parent directory of the destination if it does not exist
- Invokes `hugo --source <source> --destination <abs-destination> --quiet`
- Streams Hugo's stdout and stderr to the terminal

```bash
# Custom source and destination
go run github.com/go-sum/docs/cli build --source ./my-docs --destination ./dist/doc
```

---

## Directory Layout

### Scaffolded `.docs/` Skeleton

After running `docs init`, the following structure is created:

```
.docs/
  hugo.toml                          # Hugo configuration
  content/
    _index.md                        # Documentation home page
  layouts/
    _default/
      baseof.html                    # Base template with header, sidebar, main
      list.html                      # List page template (sections, home)
      single.html                    # Single page template (leaf pages)
    partials/
      sidebar.html                   # Navigation sidebar partial
    404.html                         # Custom not-found page
  assets/
    css/
      docs.css                       # Documentation layout styles
      theme-base.css                 # Base theme variables
      theme-slate.css                # Slate colour scheme
      chroma.css                     # Syntax highlighting styles
      chromastyles.css               # Additional Chroma token styles
    js/
      theme.js                       # Theme toggle (light/dark/system)
```

### Built Output

After running `docs build`, the compiled site appears under `public/doc/`:

```
public/
  doc/
    index.html                       # Documentation home page
    404.html                         # Custom not-found page
    css/
      ...                            # Compiled stylesheets
    js/
      ...                            # Compiled JavaScript
    <section>/
      index.html                     # Section listing page
      <page>/
        index.html                   # Individual documentation page
```

---

## Hugo Template Features

### Sidebar Navigation

The sidebar partial (`layouts/partials/sidebar.html`) renders a two-level navigation tree automatically derived from Hugo's content structure. Top-level sections are listed by `weight`, and each section expands to show its child pages and sub-sections. The current page receives the `is-active` CSS class and `aria-current="page"` attribute.

Ordering is controlled by the `weight` front-matter parameter in each content file:

```toml
+++
title = "Getting Started"
weight = 10
+++
```

### Theme Switching

The scaffolded site includes a three-state theme toggle that cycles through light, dark, and system modes. The preference is persisted in `localStorage` under the `themePreference` key. When set to `system`, the site reacts to the operating system's `prefers-color-scheme` media query in real time.

The theme toggle button is in the page header and uses distinct SVG icons for each state (sun for light, moon for dark, monitor for system).

### Syntax Highlighting

Hugo's built-in code fence highlighting is enabled via the `hugo.toml` configuration:

```toml
[markup.highlight]
  codeFences = true
  noClasses = false
```

The `noClasses = false` setting causes Hugo to emit CSS class names rather than inline styles, which are then styled by the included `chroma.css` and `chromastyles.css` stylesheets. This ensures syntax highlighting respects the active theme.

### Hugo Module Mounts

The scaffolded `hugo.toml` includes module mount configuration that maps source directories to Hugo's virtual filesystem:

```toml
[module]
  [[module.mounts]]
    source = "content"
    target = "content"
  [[module.mounts]]
    source = "layouts"
    target = "layouts"
  [[module.mounts]]
    source = "assets"
    target = "assets"
```

Additional mounts can pull in external markdown files, such as package READMEs from other modules in a workspace:

```toml
[[module.mounts]]
  source = "../pkg/web"
  target = "content/packages/web"
  includeFiles = ["README.md"]
```

This allows documentation to aggregate content from across the repository without duplicating files.

---

## Customizing the Base Path

To mount documentation at a custom URL prefix, provide a `Config` with the desired `BasePath`:

```go
cfg := docs.Config{
    PublicDir:         "public",
    BasePath:          "/api-docs",
    AssetCacheControl: "public, max-age=3600",
    PageCacheControl:  "no-cache",
}
router.Register(rt, docs.Routes(cfg)...)
```

The matching `hugo.toml` must set `baseURL` to the same prefix so that Hugo generates correct internal links:

```toml
baseURL = "/api-docs/"
```

---

## Testing

The handler package is designed for straightforward testing without an external Hugo build or running server. Tests cover:

- **Path resolution** -- verifies that `resolvePath` maps request paths to the correct filesystem paths, classifies assets vs. HTML pages, and rejects path traversal attempts
- **Page and asset serving** -- confirms correct HTTP status codes, response bodies, and content types for HTML pages, CSS files, JavaScript files, missing pages (custom 404), and missing assets
- **Cache-Control headers** -- asserts that HTML pages receive `no-cache` and assets receive `public, max-age=3600`
- **Error cases** -- empty root directory, path traversal with `..`, and missing files

```bash
go test github.com/go-sum/docs
```

[go-sum/web]: https://github.com/nicholasgasior/go-sum/tree/main/pkg/web
[Cobra]: https://cobra.dev/
