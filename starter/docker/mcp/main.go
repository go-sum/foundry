package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	gitignore "github.com/sabhiram/go-gitignore"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const useWatcher = true // true = fsnotify event-driven, false = TTL polling

// --- Parameter types ---------------------------------------------------------

type DocumentSymbolParams struct {
	Content string `json:"content"`
}

type FileSymbolParams struct {
	Path      string `json:"path"`
	Workspace string `json:"workspace"`
}

type WorkspaceSymbolParams struct {
	Workspace string `json:"workspace"`
	Query     string `json:"query"`
}

type DefinitionParams struct {
	Workspace string `json:"workspace"`
	Symbol    string `json:"symbol"`
}

type ReferencesParams struct {
	Workspace string `json:"workspace"`
	Symbol    string `json:"symbol"`
}

type DecisionsListParams struct {
	Workspace string `json:"workspace"`
}

type DecisionsReadParams struct {
	Workspace string `json:"workspace"`
	Document  string `json:"document"`
	Section   string `json:"section"`
}

type DecisionsSearchParams struct {
	Workspace  string `json:"workspace"`
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
}

// --- AST helpers -------------------------------------------------------------

type Symbol struct {
	Name     string
	Kind     string
	Line     int
	Children []Symbol
}

func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprString(e.X)
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	default:
		return "?"
	}
}

func parseSymbols(content string) ([]Symbol, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		f, _ = parser.ParseFile(fset, "", content, parser.ParseComments|parser.AllErrors)
		if f == nil {
			return nil, fmt.Errorf("parse: %w", err)
		}
	}

	var symbols []Symbol
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			sym := Symbol{Name: d.Name.Name, Line: fset.Position(d.Pos()).Line, Kind: "Function"}
			if d.Recv != nil && len(d.Recv.List) > 0 {
				sym.Kind = "Method"
				sym.Name = "(" + exprString(d.Recv.List[0].Type) + ")." + d.Name.Name
			}
			symbols = append(symbols, sym)

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					sym := Symbol{Name: s.Name.Name, Line: fset.Position(s.Pos()).Line, Kind: "Type"}
					switch st := s.Type.(type) {
					case *ast.StructType:
						sym.Kind = "Struct"
						for _, field := range st.Fields.List {
							ft := exprString(field.Type)
							for _, name := range field.Names {
								sym.Children = append(sym.Children, Symbol{
									Name: name.Name + " " + ft,
									Kind: "Field",
									Line: fset.Position(name.Pos()).Line,
								})
							}
						}
					case *ast.InterfaceType:
						sym.Kind = "Interface"
						for _, method := range st.Methods.List {
							for _, name := range method.Names {
								sym.Children = append(sym.Children, Symbol{
									Name: name.Name,
									Kind: "Method",
									Line: fset.Position(name.Pos()).Line,
								})
							}
						}
					}
					symbols = append(symbols, sym)

				case *ast.ValueSpec:
					kind := "Var"
					if d.Tok == token.CONST {
						kind = "Const"
					}
					for _, name := range s.Names {
						symbols = append(symbols, Symbol{Name: name.Name, Kind: kind, Line: fset.Position(name.Pos()).Line})
					}
				}
			}
		}
	}
	return symbols, nil
}

func formatSymbols(symbols []Symbol, indent string) string {
	var sb strings.Builder
	for _, sym := range symbols {
		fmt.Fprintf(&sb, "%s%-12s %s (line %d)\n", indent, sym.Kind, sym.Name, sym.Line)
		if len(sym.Children) > 0 {
			sb.WriteString(formatSymbols(sym.Children, indent+"  "))
		}
	}
	return sb.String()
}

// --- Workspace index ---------------------------------------------------------

func skipDir(name string) bool {
	switch name {
	case "vendor", ".git", "testdata", "node_modules", ".cache", "__pycache__", ".idea", ".vscode":
		return true
	}
	return false
}

// loadGitIgnore reads .gitignore and .git/info/exclude from the workspace root
// and compiles them into a single GitIgnore matcher. Returns nil if no rules exist.
func loadGitIgnore(workspace string) *gitignore.GitIgnore {
	candidates := []string{
		filepath.Join(workspace, ".gitignore"),
		filepath.Join(workspace, ".git", "info", "exclude"),
	}
	var lines []string
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil {
			lines = append(lines, strings.Split(string(data), "\n")...)
		}
	}
	if len(lines) == 0 {
		return nil
	}
	return gitignore.CompileIgnoreLines(lines...)
}

type fileEntry struct {
	RelPath string
	Symbols []Symbol
	Lines   []string
	AST     *ast.File
	FSet    *token.FileSet
	ModTime time.Time
}

type workspaceIndex struct {
	mu           sync.RWMutex
	entries      []fileEntry
	bySymbol     map[string][]int // lowercase symbol name → indices into entries
	identInFiles map[string][]int // lowercase ident name → file indices
	workspace    string
	builtAt      time.Time
	ttl          time.Duration
	building     sync.Mutex
}

func newWorkspaceIndex(workspace string, ttl time.Duration) *workspaceIndex {
	idx := &workspaceIndex{workspace: workspace, ttl: ttl}
	idx.rebuild()
	return idx
}

func (ws *workspaceIndex) rebuild() {
	ws.building.Lock()
	defer ws.building.Unlock()

	gi := loadGitIgnore(ws.workspace)

	ws.mu.RLock()
	oldByPath := make(map[string]fileEntry, len(ws.entries))
	for _, e := range ws.entries {
		oldByPath[e.RelPath] = e
	}
	ws.mu.RUnlock()

	var paths []string
	modTimes := make(map[string]time.Time)
	_ = filepath.WalkDir(ws.workspace, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			if gi != nil {
				rel, _ := filepath.Rel(ws.workspace, path)
				if gi.MatchesPath(rel) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if gi != nil {
			rel, _ := filepath.Rel(ws.workspace, path)
			if gi.MatchesPath(rel) {
				return nil
			}
		}
		if strings.HasSuffix(path, ".go") {
			paths = append(paths, path)
			if info, err := d.Info(); err == nil {
				rel, _ := filepath.Rel(ws.workspace, path)
				modTimes[rel] = info.ModTime()
			}
		}
		return nil
	})

	type parseResult struct {
		entry fileEntry
		ok    bool
	}
	results := make([]parseResult, len(paths))
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())

	for i, p := range paths {
		rel, _ := filepath.Rel(ws.workspace, p)
		if old, exists := oldByPath[rel]; exists && old.ModTime.Equal(modTimes[rel]) {
			results[i] = parseResult{entry: old, ok: true}
			continue
		}
		wg.Add(1)
		go func(i int, p, rel string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			content, err := os.ReadFile(p)
			if err != nil {
				return
			}
			syms, _ := parseSymbols(string(content))
			fset := token.NewFileSet()
			f, _ := parser.ParseFile(fset, rel, content, 0)
			results[i] = parseResult{
				entry: fileEntry{
					RelPath: rel,
					Symbols: syms,
					Lines:   strings.Split(string(content), "\n"),
					AST:     f,
					FSet:    fset,
					ModTime: modTimes[rel],
				},
				ok: true,
			}
		}(i, p, rel)
	}
	wg.Wait()

	entries := make([]fileEntry, 0, len(paths))
	bySymbol := make(map[string][]int)
	for _, r := range results {
		if !r.ok {
			continue
		}
		idx := len(entries)
		entries = append(entries, r.entry)
		for _, sym := range r.entry.Symbols {
			key := strings.ToLower(sym.Name)
			bySymbol[key] = append(bySymbol[key], idx)
			for _, child := range sym.Children {
				ckey := strings.ToLower(child.Name)
				bySymbol[ckey] = append(bySymbol[ckey], idx)
			}
		}
	}

	identInFiles := make(map[string][]int)
	for fileIdx, entry := range entries {
		if entry.AST == nil {
			continue
		}
		seen := make(map[string]bool)
		ast.Inspect(entry.AST, func(n ast.Node) bool {
			ident, ok := n.(*ast.Ident)
			if !ok || seen[ident.Name] {
				return true
			}
			seen[ident.Name] = true
			key := strings.ToLower(ident.Name)
			identInFiles[key] = append(identInFiles[key], fileIdx)
			return true
		})
	}

	ws.mu.Lock()
	ws.entries = entries
	ws.bySymbol = bySymbol
	ws.identInFiles = identInFiles
	ws.builtAt = time.Now()
	ws.mu.Unlock()

	slog.Info("workspace index built", "files", len(entries))
}

func (ws *workspaceIndex) ensureFresh() {
	if useWatcher {
		return // Watcher triggers rebuilds on change; no TTL needed.
	}
	ws.mu.RLock()
	stale := time.Since(ws.builtAt) > ws.ttl
	ws.mu.RUnlock()
	if stale {
		go ws.rebuild()
	}
}

// startWatcher starts an fsnotify-based file watcher that triggers idx.rebuild()
// on .go file changes with a 200ms debounce to batch rapid events.
// All non-skipped directories in workspace are watched recursively.
func startWatcher(idx *workspaceIndex) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("fsnotify unavailable, falling back to TTL", "err", err)
		return
	}

	// Add all non-skipped directories to the watcher.
	_ = filepath.WalkDir(idx.workspace, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if skipDir(d.Name()) {
			return filepath.SkipDir
		}
		_ = w.Add(path)
		return nil
	})

	var (
		timerMu sync.Mutex
		timer   *time.Timer
	)

	go func() {
		defer w.Close()
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					return
				}
				// Watch new directories as they are created.
				if event.Has(fsnotify.Create) {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						if !skipDir(filepath.Base(event.Name)) {
							_ = w.Add(event.Name)
						}
					}
				}
				// Trigger debounced rebuild on .go file changes.
				if strings.HasSuffix(event.Name, ".go") {
					timerMu.Lock()
					if timer != nil {
						timer.Stop()
					}
					timer = time.AfterFunc(200*time.Millisecond, func() {
						go idx.rebuild()
					})
					timerMu.Unlock()
				}
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				slog.Warn("fsnotify error", "err", err)
			}
		}
	}()
}

type wsSymbol struct {
	File   string
	Symbol Symbol
}

func (ws *workspaceIndex) searchSymbols(query string) []wsSymbol {
	ws.ensureFresh()
	q := strings.ToLower(query)
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	// Fast path for exact-name matches via bySymbol map
	if q != "" {
		if indices, ok := ws.bySymbol[q]; ok {
			seen := make(map[string]bool)
			var results []wsSymbol
			for _, idx := range indices {
				entry := ws.entries[idx]
				for _, sym := range entry.Symbols {
					if strings.ToLower(sym.Name) == q || strings.Contains(strings.ToLower(sym.Name), q) {
						key := entry.RelPath + "\x00" + sym.Name
						if !seen[key] {
							seen[key] = true
							results = append(results, wsSymbol{File: entry.RelPath, Symbol: sym})
						}
					}
					for _, child := range sym.Children {
						if strings.ToLower(child.Name) == q || strings.Contains(strings.ToLower(child.Name), q) {
							key := entry.RelPath + "\x00" + child.Name
							if !seen[key] {
								seen[key] = true
								results = append(results, wsSymbol{File: entry.RelPath, Symbol: child})
							}
						}
					}
				}
			}
			if len(results) > 0 {
				return results
			}
		}
	}

	// Slow path: substring scan across all entries
	seen := make(map[string]bool)
	var results []wsSymbol
	for _, entry := range ws.entries {
		for _, sym := range entry.Symbols {
			if q == "" || strings.Contains(strings.ToLower(sym.Name), q) {
				key := entry.RelPath + "\x00" + sym.Name
				if !seen[key] {
					seen[key] = true
					results = append(results, wsSymbol{File: entry.RelPath, Symbol: sym})
				}
			}
			for _, child := range sym.Children {
				if q == "" || strings.Contains(strings.ToLower(child.Name), q) {
					key := entry.RelPath + "\x00" + child.Name
					if !seen[key] {
						seen[key] = true
						results = append(results, wsSymbol{File: entry.RelPath, Symbol: child})
					}
				}
			}
		}
	}
	return results
}

type ref struct {
	File string
	Line int
	Text string
}

func lineAt(lines []string, n int) string {
	if n > 0 && n <= len(lines) {
		return strings.TrimSpace(lines[n-1])
	}
	return ""
}

func (ws *workspaceIndex) findDefinition(symbol string) []ref {
	ws.ensureFresh()
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	key := strings.ToLower(symbol)
	indices, ok := ws.bySymbol[key]
	if !ok {
		return nil
	}

	matchName := func(sym Symbol) bool {
		name := sym.Name
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			name = name[idx+1:]
		}
		return name == symbol || sym.Name == symbol
	}

	var results []ref
	for _, idx := range indices {
		entry := ws.entries[idx]
		for _, sym := range entry.Symbols {
			if matchName(sym) {
				results = append(results, ref{File: entry.RelPath, Line: sym.Line, Text: lineAt(entry.Lines, sym.Line)})
			}
			for _, child := range sym.Children {
				childName := child.Name
				if ci := strings.Index(childName, " "); ci >= 0 {
					childName = childName[:ci]
				}
				if childName == symbol {
					results = append(results, ref{File: entry.RelPath, Line: child.Line, Text: lineAt(entry.Lines, child.Line)})
				}
			}
		}
	}
	return results
}

func (ws *workspaceIndex) findReferences(symbol string) []ref {
	ws.ensureFresh()
	key := strings.ToLower(symbol)
	ws.mu.RLock()
	fileIdxs := ws.identInFiles[key]
	entries := make([]fileEntry, len(fileIdxs))
	for i, fi := range fileIdxs {
		entries[i] = ws.entries[fi]
	}
	ws.mu.RUnlock()

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	var results []ref

	for _, entry := range entries {
		if entry.AST == nil {
			continue
		}
		wg.Add(1)
		go func(e fileEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			var local []ref
			ast.Inspect(e.AST, func(n ast.Node) bool {
				ident, ok := n.(*ast.Ident)
				if !ok || ident.Name != symbol {
					return true
				}
				pos := e.FSet.Position(ident.Pos())
				local = append(local, ref{
					File: e.RelPath,
					Line: pos.Line,
					Text: lineAt(e.Lines, pos.Line),
				})
				return true
			})
			mu.Lock()
			results = append(results, local...)
			mu.Unlock()
		}(entry)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		if results[i].File != results[j].File {
			return results[i].File < results[j].File
		}
		return results[i].Line < results[j].Line
	})
	return results
}

// --- Decision documents ------------------------------------------------------

type DocSection struct {
	Number  string // "1", "2a", "5f" — empty if unnumbered
	Title   string
	Level   int // 2 for ##, 3 for ###
	Start   int // 1-based line number of header
	End     int // 1-based exclusive end line
	Content string
}

type DecisionDoc struct {
	Filename    string
	Title       string
	Description string
	Weight      int
	Sections    []DocSection
	RawContent  string
}

func parseFrontmatter(content string) (title, desc string, weight int) {
	lines := strings.Split(content, "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			break
		}
		line := lines[i]
		if after, ok := strings.CutPrefix(line, "title:"); ok {
			title = strings.Trim(strings.TrimSpace(after), "\"'")
		} else if after, ok := strings.CutPrefix(line, "description:"); ok {
			desc = strings.Trim(strings.TrimSpace(after), "\"'")
		} else if after, ok := strings.CutPrefix(line, "weight:"); ok {
			weight, _ = strconv.Atoi(strings.TrimSpace(after))
		}
	}
	return
}

// frontmatterEnd returns the 0-based index of the first line after the closing ---.
func frontmatterEnd(content string) int {
	lines := strings.Split(content, "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return 0
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return i + 1
		}
	}
	return 0
}

// parseSections parses ## and ### section boundaries starting at startLine (0-based).
// Returns sections with 1-based Start/End line numbers.
func parseSections(content string, startLine int) []DocSection {
	lines := strings.Split(content, "\n")
	var sections []DocSection

	for i := startLine; i < len(lines); i++ {
		line := lines[i]
		var level int
		var rest string
		switch {
		case strings.HasPrefix(line, "### "):
			level, rest = 3, line[4:]
		case strings.HasPrefix(line, "## "):
			level, rest = 2, line[3:]
		default:
			continue
		}

		// Close previous section at the line before this header (1-based).
		if len(sections) > 0 {
			sections[len(sections)-1].End = i + 1
		}

		// Extract section number from leading "N.", "2a.", etc.
		number, title := "", rest
		if parts := strings.SplitN(rest, ".", 2); len(parts) == 2 {
			candidate := strings.TrimSpace(parts[0])
			isNum := len(candidate) > 0 && candidate[0] >= '0' && candidate[0] <= '9'
			for _, ch := range candidate {
				if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')) {
					isNum = false
					break
				}
			}
			if isNum {
				number = strings.ToLower(candidate)
				title = strings.TrimSpace(parts[1])
			}
		}

		sections = append(sections, DocSection{
			Number: number,
			Title:  title,
			Level:  level,
			Start:  i + 1, // 1-based
		})
	}

	if len(sections) > 0 {
		sections[len(sections)-1].End = len(lines) + 1
	}

	// Fill Content for each section (Start and End are 1-based).
	for i := range sections {
		start := sections[i].Start - 1 // convert to 0-based for slicing
		end := min(sections[i].End-1, len(lines))
		sections[i].Content = strings.Join(lines[start:end], "\n")
	}

	return sections
}

func loadDecisionDocs(workspace string) ([]DecisionDoc, error) {
	dir := filepath.Join(workspace, ".decisions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var docs []DecisionDoc
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		raw := string(content)
		title, desc, weight := parseFrontmatter(raw)
		sections := parseSections(raw, frontmatterEnd(raw))
		docs = append(docs, DecisionDoc{
			Filename:    e.Name(),
			Title:       title,
			Description: desc,
			Weight:      weight,
			Sections:    sections,
			RawContent:  raw,
		})
	}

	sort.Slice(docs, func(i, j int) bool { return docs[i].Weight < docs[j].Weight })
	slog.Info("decision docs loaded", "count", len(docs))
	return docs, nil
}

// --- Response helpers --------------------------------------------------------

func toolText(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "error: " + msg}}}
}

func refsText(refs []ref) string {
	var sb strings.Builder
	for _, r := range refs {
		fmt.Fprintf(&sb, "%s:%d\t%s\n", r.File, r.Line, r.Text)
	}
	return sb.String()
}

func resolveWorkspace(param string) string {
	if param != "" {
		return param
	}
	if ws := os.Getenv("WORKSPACE"); ws != "" {
		return ws
	}
	return "/src/foundry"
}

// --- Decision document handlers ----------------------------------------------

func handleDecisionsList(docs []DecisionDoc) func(context.Context, *mcp.CallToolRequest, DecisionsListParams) (*mcp.CallToolResult, any, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ DecisionsListParams) (*mcp.CallToolResult, any, error) {
		if len(docs) == 0 {
			return toolText("no governing documents found in .decisions/"), nil, nil
		}
		var sb strings.Builder
		for _, doc := range docs {
			fmt.Fprintf(&sb, "%s  (weight: %d)\n", doc.Filename, doc.Weight)
			fmt.Fprintf(&sb, "  Title: %s\n", doc.Title)
			fmt.Fprintf(&sb, "  Description: %s\n", doc.Description)
			fmt.Fprintf(&sb, "  Sections: %d\n", len(doc.Sections))
			for _, s := range doc.Sections {
				indent := "    "
				if s.Level == 3 {
					indent = "      "
				}
				ref := s.Title
				if s.Number != "" {
					ref = "§" + s.Number + " " + s.Title
				}
				sb.WriteString(indent + ref + "\n")
			}
			sb.WriteString("\n")
		}
		return toolText(sb.String()), nil, nil
	}
}

func handleDecisionsRead(docs []DecisionDoc) func(context.Context, *mcp.CallToolRequest, DecisionsReadParams) (*mcp.CallToolResult, any, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, p DecisionsReadParams) (*mcp.CallToolResult, any, error) {
		if p.Document == "" {
			return toolError("document is required"), nil, nil
		}
		var doc *DecisionDoc
		for i := range docs {
			if strings.EqualFold(docs[i].Filename, p.Document) {
				doc = &docs[i]
				break
			}
		}
		if doc == nil {
			return toolError(fmt.Sprintf("document %q not found; use decisions_list to see available documents", p.Document)), nil, nil
		}

		if p.Section == "" {
			lineCount := strings.Count(doc.RawContent, "\n")
			var sb strings.Builder
			if lineCount > 500 {
				fmt.Fprintf(&sb, "// %s (%d lines) — use section parameter for targeted retrieval\n\n", doc.Filename, lineCount)
			}
			sb.WriteString(doc.RawContent)
			return toolText(sb.String()), nil, nil
		}

		q := strings.ToLower(strings.TrimSpace(p.Section))
		var matched *DocSection
		// Exact number match first.
		for i := range doc.Sections {
			if strings.ToLower(doc.Sections[i].Number) == q {
				matched = &doc.Sections[i]
				break
			}
		}
		// Fall back to title substring match.
		if matched == nil {
			for i := range doc.Sections {
				if strings.Contains(strings.ToLower(doc.Sections[i].Title), q) {
					matched = &doc.Sections[i]
					break
				}
			}
		}
		if matched == nil {
			return toolError(fmt.Sprintf("section %q not found in %s; use decisions_list to see available sections", p.Section, p.Document)), nil, nil
		}

		// For a level-2 (##) section, include all its ### subsections.
		content := matched.Content
		if matched.Level == 2 {
			rawLines := strings.Split(doc.RawContent, "\n")
			endLine := len(rawLines)
			for _, s := range doc.Sections {
				if s.Start > matched.Start && s.Level == 2 {
					endLine = s.Start - 1
					break
				}
			}
			start := max(matched.Start-1, 0)
			if endLine > len(rawLines) {
				endLine = len(rawLines)
			}
			content = strings.Join(rawLines[start:endLine], "\n")
		}

		var sb strings.Builder
		sectionRef := matched.Title
		if matched.Number != "" {
			sectionRef = "§" + matched.Number + " " + matched.Title
		}
		fmt.Fprintf(&sb, "// %s — %s (lines %d–%d)\n\n", doc.Filename, sectionRef, matched.Start, matched.End-1)
		sb.WriteString(content)

		// Surface markdown cross-references to other .decisions/ files.
		var related []string
		seenLinks := make(map[string]bool)
		for line := range strings.SplitSeq(content, "\n") {
			s := line
			for {
				i := strings.Index(s, "](")
				if i < 0 {
					break
				}
				rest := s[i+2:]
				before, after, ok := strings.Cut(rest, ")")
				if !ok {
					break
				}
				link := strings.TrimPrefix(before, "./")
				if strings.HasSuffix(link, ".md") && !seenLinks[link] {
					seenLinks[link] = true
					related = append(related, link)
				}
				s = after
			}
		}
		if len(related) > 0 {
			sb.WriteString("\n\n// Related: " + strings.Join(related, ", "))
		}

		return toolText(sb.String()), nil, nil
	}
}

func handleDecisionsSearch(docs []DecisionDoc) func(context.Context, *mcp.CallToolRequest, DecisionsSearchParams) (*mcp.CallToolResult, any, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, p DecisionsSearchParams) (*mcp.CallToolResult, any, error) {
		if p.Query == "" {
			return toolError("query is required"), nil, nil
		}
		max := p.MaxResults
		if max <= 0 {
			max = 10
		}

		terms := strings.Fields(strings.ToLower(p.Query))
		type scored struct {
			doc     *DecisionDoc
			section *DocSection
			score   int
		}
		var results []scored

		for i := range docs {
			doc := &docs[i]
			for j := range doc.Sections {
				s := &doc.Sections[j]
				titleLow := strings.ToLower(s.Title)
				contentLow := strings.ToLower(s.Content)
				score := 0
				for _, term := range terms {
					if strings.Contains(titleLow, term) {
						score += 10
					}
					if strings.Contains(contentLow, term) {
						score++
					}
				}
				if score > 0 {
					results = append(results, scored{doc, s, score})
				}
			}
		}

		sort.Slice(results, func(i, j int) bool {
			if results[i].score != results[j].score {
				return results[i].score > results[j].score
			}
			return results[i].doc.Weight < results[j].doc.Weight
		})
		if len(results) > max {
			results = results[:max]
		}
		if len(results) == 0 {
			return toolText(fmt.Sprintf("no results found for %q", p.Query)), nil, nil
		}

		var sb strings.Builder
		for rank, r := range results {
			sectionRef := r.section.Title
			if r.section.Number != "" {
				sectionRef = "§" + r.section.Number + " " + r.section.Title
			}
			fmt.Fprintf(&sb, "[%d] %s — %s (line %d)\n", rank+1, r.doc.Filename, sectionRef, r.section.Start)
			preview := 0
			for _, line := range strings.Split(r.section.Content, "\n")[1:] {
				if strings.TrimSpace(line) == "" {
					continue
				}
				sb.WriteString("    > " + line + "\n")
				preview++
				if preview >= 3 {
					break
				}
			}
			sb.WriteString("\n")
		}
		return toolText(sb.String()), nil, nil
	}
}

// --- Main --------------------------------------------------------------------

func main() {
	workspace := resolveWorkspace("")

	idx := newWorkspaceIndex(workspace, 30*time.Second)

	if useWatcher {
		startWatcher(idx)
	}

	docs, err := loadDecisionDocs(workspace)
	if err != nil {
		slog.Warn("could not load decision docs", "err", err)
	}

	srv := mcp.NewServer(&mcp.Implementation{Name: "gosdk", Version: "v2.0.0"}, nil)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "lsp_document_symbols",
		Description: "List all symbols in a Go source file. Pass the full file content as `content`. Use lsp_file_symbols instead when working with workspace files.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, p DocumentSymbolParams) (*mcp.CallToolResult, any, error) {
		if p.Content == "" {
			return toolError("content is required"), nil, nil
		}
		symbols, err := parseSymbols(p.Content)
		if err != nil {
			return toolError(err.Error()), nil, nil
		}
		if len(symbols) == 0 {
			return toolText("no symbols found"), nil, nil
		}
		return toolText(formatSymbols(symbols, "")), nil, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "lsp_file_symbols",
		Description: "List all symbols in a Go file by path relative to the workspace. Preferred over lsp_document_symbols — no need to pass file content.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, p FileSymbolParams) (*mcp.CallToolResult, any, error) {
		if p.Path == "" {
			return toolError("path is required"), nil, nil
		}
		ws := resolveWorkspace(p.Workspace)
		content, err := os.ReadFile(filepath.Join(ws, p.Path))
		if err != nil {
			return toolError(fmt.Sprintf("read %s: %v", p.Path, err)), nil, nil
		}
		symbols, err := parseSymbols(string(content))
		if err != nil {
			return toolError(err.Error()), nil, nil
		}
		if len(symbols) == 0 {
			return toolText("no symbols found"), nil, nil
		}
		return toolText(formatSymbols(symbols, "")), nil, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "lsp_workspace_symbols",
		Description: "Search for symbols across all Go files in the workspace. Uses an in-memory index (30s TTL). Empty `query` returns all symbols.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, p WorkspaceSymbolParams) (*mcp.CallToolResult, any, error) {
		results := idx.searchSymbols(p.Query)
		if len(results) == 0 {
			return toolText(fmt.Sprintf("no symbols found for query %q", p.Query)), nil, nil
		}
		sort.Slice(results, func(i, j int) bool {
			if results[i].File != results[j].File {
				return results[i].File < results[j].File
			}
			return results[i].Symbol.Line < results[j].Symbol.Line
		})
		var sb strings.Builder
		for _, r := range results {
			fmt.Fprintf(&sb, "%s:%d\t%-12s %s\n", r.File, r.Symbol.Line, r.Symbol.Kind, r.Symbol.Name)
		}
		return toolText(sb.String()), nil, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "lsp_definition",
		Description: "Find the declaration of a named symbol (function, type, method, field, var, const) across the workspace.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, p DefinitionParams) (*mcp.CallToolResult, any, error) {
		if p.Symbol == "" {
			return toolError("symbol is required"), nil, nil
		}
		defs := idx.findDefinition(p.Symbol)
		if len(defs) == 0 {
			return toolText(fmt.Sprintf("no definition found for %q", p.Symbol)), nil, nil
		}
		return toolText(refsText(defs)), nil, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "lsp_find_references",
		Description: "Find all identifier-level references to a symbol across the workspace.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, p ReferencesParams) (*mcp.CallToolResult, any, error) {
		if p.Symbol == "" {
			return toolError("symbol is required"), nil, nil
		}
		refs := idx.findReferences(p.Symbol)
		if len(refs) == 0 {
			return toolText(fmt.Sprintf("no references found for %q", p.Symbol)), nil, nil
		}
		return toolText(refsText(refs)), nil, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "decisions_list",
		Description: "List all governing documents in .decisions/ with metadata: title, description, weight, and section index. Use before decisions_read to find the right document and section.",
	}, handleDecisionsList(docs))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "decisions_read",
		Description: "Read a governing document or a specific section. Set `section` to a number (e.g. '5', '5a') or title substring for targeted retrieval. A level-2 section (##) includes all its subsections (###).",
	}, handleDecisionsRead(docs))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "decisions_search",
		Description: "Search all governing documents by keyword. Returns scored section matches with content preview. Use to find relevant guidance without knowing which document to consult.",
	}, handleDecisionsSearch(docs))

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, "ok") })

	port := os.Getenv("MCP_PORT")
	if port == "" {
		port = "8086"
	}
	addr := ":" + port
	slog.Info("MCP server listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
