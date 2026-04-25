// Package componentry renders a living reference of every visual component in
// pkg/componentry/. It produces a pure g.Node with no HTTP or internal/ imports
// and follows the componentry module's documented tiered import DAG.
// The example() card self-embeds its own source in a "View source" <details> toggle
package componentry

import (
	_ "embed"
	"fmt"
	"strings"

	uiform "github.com/go-sum/componentry/form"
	"github.com/go-sum/componentry/icons"
	iconrender "github.com/go-sum/componentry/icons/render"
	"github.com/go-sum/componentry/interactive/accordion"
	"github.com/go-sum/componentry/interactive/breadcrumb"
	"github.com/go-sum/componentry/interactive/dialog"
	"github.com/go-sum/componentry/interactive/dropdown"
	"github.com/go-sum/componentry/interactive/pagination"
	"github.com/go-sum/componentry/interactive/tabs"
	"github.com/go-sum/componentry/interactive/theme"
	"github.com/go-sum/componentry/interactive/tooltip"
	htmx "github.com/go-sum/componentry/patterns/htmx"
	"github.com/go-sum/componentry/patterns/pager"
	"github.com/go-sum/showcase/componentry/demo"
	"github.com/go-sum/componentry/ui/core"
	"github.com/go-sum/componentry/ui/data"
	"github.com/go-sum/componentry/ui/feedback"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

//go:embed showcase.go
var showcaseSource string

var snippets map[string]string

func init() {
	snippets = make(map[string]string)
	var name string
	var buf strings.Builder
	for _, line := range strings.Split(showcaseSource, "\n") {
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(trimmed, "// src:"); ok {
			if after == "end" {
				snippets[name] = dedent(strings.TrimRight(buf.String(), "\n"))
				name = ""
				buf.Reset()
			} else {
				name = after
			}
			continue
		}
		if name != "" {
			buf.WriteString(line + "\n")
		}
	}
}

// dedent strips the common leading whitespace from all non-empty lines in s.
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	minIndent := len(s)
	for _, l := range lines {
		if trimmed := strings.TrimLeft(l, "\t "); len(trimmed) > 0 {
			indent := len(l) - len(trimmed)
			if indent < minIndent {
				minIndent = indent
			}
		}
	}
	for i, l := range lines {
		if len(l) >= minIndent {
			lines[i] = l[minIndent:]
		}
	}
	return strings.Join(lines, "\n")
}

// Showcase returns the full component showcase as a single renderable node.
func Showcase() g.Node {
	return h.Div(
		h.ID("top"),
		h.Class("max-w-4xl mx-auto space-y-12 py-8"),
		h.Div(
			h.Class("space-y-2"),
			h.H1(h.Class("text-2xl font-bold"), g.Text("Component Examples")),
			h.P(
				h.Class("max-w-2xl text-sm text-muted-foreground"),
				g.Text("Live reference for every visual component in pkg/componentry/, arranged to match the starter's default visual language."),
			),
			h.Div(
				h.Class("flex items-center gap-4 text-xs text-muted-foreground"),
				h.Span(h.Class("flex items-center gap-1.5"),
					h.Span(h.Class("relative flex size-2.5"),
						h.Span(h.Class("absolute inline-flex h-full w-full animate-ping rounded-full bg-amber-500 opacity-75")),
						h.Span(h.Class("relative inline-flex size-2.5 rounded-full bg-amber-600")),
					),
					g.Text("HTMX endpoint"),
				),
				h.Span(h.Class("flex items-center gap-1.5"),
					h.Span(h.Class("relative flex size-2.5"),
						h.Span(h.Class("absolute inline-flex h-full w-full animate-ping rounded-full bg-sky-400 opacity-75")),
						h.Span(h.Class("relative inline-flex size-2.5 rounded-full bg-sky-500")),
					),
					g.Text("JS controller"),
				),
			),
		),
		data.Card.Root(
			data.Card.Header(
				data.Card.Title(g.Text("Contents")),
				data.Card.Description(g.Text("Jump to a component family and compare the preferred variants side by side.")),
			),
			data.Card.Content(
				h.Ul(h.Class("columns-1 gap-x-6 space-y-1 text-sm sm:columns-2 lg:columns-3"),
					tocItem("accordion", "Accordion"),
					tocItem("alerts", "Alerts"),
					tocItem("avatars", "Avatars"),
					tocItem("badges", "Badges"),
					tocItem("breadcrumb", "Breadcrumb"),
					tocItem("buttons", "Buttons"),
					tocItem("cards", "Cards"),
					tocItem("dialog", "Dialog"),
					tocItem("dropdown", "Dropdown"),
					tocItem("font-loading", "Font Loading"),
					tocItem("form-fields", "Form Fields"),
					tocItem("head-builder", "Head Builder"),
					tocItem("htmx-patterns", "HTMX Patterns"),
					tocItem("labels", "Labels"),
					tocItem("pagination", "Pagination"),
					tocItem("pager", "Pager"),
					tocItem("popover", "Popover"),

					tocItem("progress", "Progress"),
					tocItem("separators", "Separators"),
					tocItem("skeleton", "Skeleton"),
					tocItem("tables", "Tables"),
					tocItem("theme", "Theme"),
					tocItem("tabs", "Tabs"),
					tocItem("toast", "Toast"),
					tocItem("tooltip", "Tooltip"),
				),
			),
		),

		// ── Accordion ───────────────────────────────────
		section("accordion", "Accordion",
			example("accordion-three-items", "Three items",
				// src:accordion-three-items
				accordion.Root(accordion.RootProps{},
					accordion.Item(
						accordion.Trigger(g.Text("Is it accessible?")),
						accordion.Content(g.Text("Yes. It uses native <details>/<summary> elements with WAI-ARIA semantics.")),
					),
					accordion.Item(
						accordion.Trigger(g.Text("Is it styled?")),
						accordion.Content(g.Text("Yes. It uses Tailwind utility classes with shadcn/ui design tokens.")),
					),
					accordion.Item(
						accordion.Trigger(g.Text("Is it animated?")),
						accordion.Content(g.Text("The chevron rotates on expand via CSS details[open] .details-chevron rule.")),
					),
				),
				// src:end
			),
		),

		// ── Alerts ──────────────────────────────────────
		section("alerts", "Alerts",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("alerts-default", "Default (dismissible)",
					// src:alerts-default
					feedback.Alert.Root(
						feedback.AlertProps{Variant: feedback.AlertDefault, Dismissible: true},
						feedback.Alert.Title(g.Text("Note")),
						feedback.Alert.Description(g.Text("Here is some helpful information.")),
					),
					// src:end
				),
				example("alerts-destructive", "Destructive (dismissible)",
					// src:alerts-destructive
					feedback.Alert.Root(
						feedback.AlertProps{Variant: feedback.AlertDestructive, Dismissible: true},
						feedback.Alert.Title(g.Text("Error")),
						feedback.Alert.Description(g.Text("Something went wrong. Please try again.")),
					),
					// src:end
				),
			),
		),

		// ── Avatars ──────────────────────────────────────
		section("avatars", "Avatars",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("avatars-fallback", "Fallback initials",
					// src:avatars-fallback
					h.Div(
						h.Class("flex gap-4"),
						core.Avatar.Fallback(g.Text("AB")),
					),
					// src:end
				),
				example("avatars-icon", "Lucide icon",
					// src:avatars-icon
					core.Icon(iconrender.PropsFor(icons.ChevronDown, core.IconProps{
						Size:  "size-10",
						Label: "User account",
					})),
					// src:end
				),
			),
		),

		// ── Badges ──────────────────────────────────────
		section("badges", "Badges",
			example("badges-variants", "Variants",
				// src:badges-variants
				h.Div(
					h.Class("flex flex-wrap gap-2"),
					core.Badge(core.BadgeProps{Children: []g.Node{g.Text("Default")}}),
					core.Badge(core.BadgeProps{Variant: core.BadgeSecondary, Children: []g.Node{g.Text("Secondary")}}),
					core.Badge(core.BadgeProps{Variant: core.BadgeDestructive, Children: []g.Node{g.Text("Destructive")}}),
					core.Badge(core.BadgeProps{Variant: core.BadgeOutline, Children: []g.Node{g.Text("Outline")}}),
				),
				// src:end
			),
		),

		// ── Breadcrumb ──────────────────────────────────
		section("breadcrumb", "Breadcrumb",
			example("breadcrumb-path", "Three-level path",
				// src:breadcrumb-path
				breadcrumb.Root(
					breadcrumb.List(
						breadcrumb.Item(breadcrumb.Link("/", g.Text("Home"))),
						breadcrumb.Item(breadcrumb.Separator()),
						breadcrumb.Item(breadcrumb.Link("/users", g.Text("Users"))),
						breadcrumb.Item(breadcrumb.Separator()),
						breadcrumb.Item(breadcrumb.Page(g.Text("Alice Johnson"))),
					),
				),
				// src:end
			),
		),

		// ── Buttons ──────────────────────────────────────
		section("buttons", "Buttons",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("buttons-variants", "Variants",
					// src:buttons-variants
					h.Div(
						h.Class("flex flex-wrap gap-2"),
						core.Button(core.ButtonProps{Label: "Default"}),
						core.Button(core.ButtonProps{Label: "Destructive", Variant: core.VariantDestructive}),
						core.Button(core.ButtonProps{Label: "Destructive Ghost", Variant: core.VariantDestructiveGhost}),
						core.Button(core.ButtonProps{Label: "Outline", Variant: core.VariantOutline}),
						core.Button(core.ButtonProps{Label: "Secondary", Variant: core.VariantSecondary}),
						core.Button(core.ButtonProps{Label: "Ghost", Variant: core.VariantGhost}),
						core.Button(core.ButtonProps{Label: "Link", Variant: core.VariantLink}),
					),
					// src:end
				),
				example("buttons-sizes", "Sizes",
					// src:buttons-sizes
					h.Div(
						h.Class("flex flex-wrap items-center gap-2"),
						core.Button(core.ButtonProps{Label: "Large", Size: core.SizeLg}),
						core.Button(core.ButtonProps{Label: "Default"}),
						core.Button(core.ButtonProps{Label: "Small", Size: core.SizeSm}),
					),
					// src:end
				),
				example("buttons-link", "Link (as <a>)",
					// src:buttons-link
					h.Div(
						h.Class("flex gap-2"),
						core.Button(core.ButtonProps{Label: "Go Home", Href: "/", Variant: core.VariantSecondary}),
						core.Button(core.ButtonProps{Label: "Users", Href: "/users", Variant: core.VariantGhost, Size: core.SizeSm}),
					),
					// src:end
				),
				example("buttons-disabled", "Disabled",
					// src:buttons-disabled
					h.Div(
						h.Class("flex gap-2"),
						core.Button(core.ButtonProps{Label: "Disabled", Disabled: true}),
						core.Button(core.ButtonProps{Label: "Disabled Outline", Variant: core.VariantOutline, Disabled: true}),
					),
					// src:end
				),
			),
		),

		// ── Cards ───────────────────────────────────────
		section("cards", "Cards",
			example("cards-anatomy", "Full card anatomy",
				// src:cards-anatomy
				data.Card.Root(
					data.Card.Header(
						data.Card.Title(g.Text("Card Title")),
						data.Card.Description(g.Text("Optional description text goes here.")),
					),
					data.Card.Content(
						h.P(g.Text("This is the main body of the card. Cards compose header, content, and footer sub-components.")),
					),
					data.Card.Footer(
						core.Button(core.ButtonProps{Label: "Action", Size: core.SizeSm}),
					),
				),
				// src:end
			),
		),

		// ── Dialog ──────────────────────────────────────
		section("dialog", "Dialog",
			example("dialog-modal", "Modal dialog with trigger",
				// src:dialog-modal
				dialog.Root(
					dialog.Trigger("example-dialog",
						core.Button(core.ButtonProps{Label: "Open Dialog"}),
					),
					dialog.Content("example-dialog",
						dialog.Header(
							dialog.Title("example-dialog", g.Text("Confirm Action")),
							dialog.Description("example-dialog", g.Text("This action cannot be undone. Are you sure you want to proceed?")),
						),
						dialog.Footer(
							dialog.Close(
								core.Button(core.ButtonProps{Label: "Cancel", Variant: core.VariantOutline}),
							),
							core.Button(core.ButtonProps{Label: "Confirm", Variant: core.VariantDestructive}),
						),
					),
				),
				// src:end
				capController,
			),
		),

		// ── Dropdown ────────────────────────────────────
		section("dropdown", "Dropdown",
			example("dropdown-native", "Native summary trigger",
				// src:dropdown-native
				dropdown.Root(
					dropdown.Props{},
					dropdown.Trigger(dropdown.TriggerProps{}, g.Text("Options")),
					dropdown.Content(
						dropdown.Label("Account"),
						dropdown.Item(dropdown.ItemProps{Label: "View Profile", Href: "#"}),
						dropdown.Item(dropdown.ItemProps{Label: "Edit Settings", Href: "#"}),
						dropdown.Separator(),
						dropdown.Item(dropdown.ItemProps{Label: "Sign Out", Href: "#"}),
					),
				),
				// src:end
			),
		),

		// ── Font Loading ─────────────────────────────────
		section("font-loading", "Font Loading",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("font-google", "Google Fonts — nodes",
					// src:font-google
					h.Div(
						h.Class("font-mono text-xs text-muted-foreground space-y-1"),
						g.Text("Renders: preconnect + stylesheet link tags"),
						h.Br(),
						g.Text("CSP StyleSrc: fonts.googleapis.com"),
						h.Br(),
						g.Text("CSP FontSrc: fonts.gstatic.com"),
					),
					// src:end
				),
				example("font-bunny", "Bunny Fonts (GDPR-friendly) — nodes",
					// src:font-bunny
					h.Div(
						h.Class("font-mono text-xs text-muted-foreground space-y-1"),
						g.Text("Renders: preconnect + stylesheet link tags"),
						h.Br(),
						g.Text("Single origin: fonts.bunny.net"),
					),
					// src:end
				),
				example("font-selfhosted", "Self-hosted @font-face",
					// src:font-selfhosted
					h.Div(
						h.Class("font-mono text-xs text-muted-foreground"),
						g.Text("Renders: preload link + @font-face <style> block"),
					),
					// src:end
				),
				example("font-csp", "CSP source collection",
					// src:font-csp
					h.Div(
						h.Class("font-mono text-xs text-muted-foreground"),
						g.Text("CollectCSPSources merges all providers and deduplicates within each directive."),
					),
					// src:end
				),
			),
		),

		// ── Head Builder ─────────────────────────────────
		section("head-builder", "Head Builder",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("head-structure", "Full <head> structure",
					// src:head-structure
					h.Div(
						h.Class("font-mono text-xs text-muted-foreground space-y-1"),
						g.Text("head.Head(Props{...}) renders:"),
						h.Ul(h.Class("mt-2 space-y-1 list-disc list-inside"),
							h.Li(g.Text("<meta charset=\"UTF-8\">")),
							h.Li(g.Text("<meta name=\"viewport\" ...>")),
							h.Li(g.Text("<title>, description, favicon")),
							h.Li(g.Text("Open Graph meta tags")),
							h.Li(g.Text("Canonical + robots directives")),
							h.Li(g.Text("<link> stylesheets with SRI")),
							h.Li(g.Text("<script> tags (defer/async)")),
							h.Li(g.Text("Extra slot: font nodes, theme script")),
						),
					),
					// src:end
				),
				example("head-og", "Open Graph tags",
					// src:head-og
					h.Div(
						h.Class("font-mono text-xs text-muted-foreground space-y-1"),
						g.Text("MetaProps.OG = &head.OpenGraph{"),
						h.Br(),
						g.Text(`  Title: "My Page",`),
						h.Br(),
						g.Text(`  Type: "website",`),
						h.Br(),
						g.Text(`  Image: "https://…/og.png",`),
						h.Br(),
						g.Text("}"),
					),
					// src:end
				),
				example("head-css", "Stylesheet with SRI",
					// src:head-css
					h.Div(
						h.Class("font-mono text-xs text-muted-foreground"),
						g.Text(`head.CSS(head.Stylesheet{`),
						h.Br(),
						g.Text(`  Href: "/static/app.css",`),
						h.Br(),
						g.Text(`  Integrity: "sha384-…",`),
						h.Br(),
						g.Text(`})`),
						h.P(h.Class("mt-2 text-muted-foreground"), g.Text(`→ renders: <link rel="stylesheet" href="/static/app.css">`)),
					),
					// src:end
				),
				example("head-script", "Script (deferred)",
					// src:head-script
					h.Div(
						h.Class("font-mono text-xs text-muted-foreground"),
						g.Text(`head.JS(head.Script{`),
						h.Br(),
						g.Text(`  Src: "/static/app.js",`),
						h.Br(),
						g.Text(`  Defer: true,`),
						h.Br(),
						g.Text(`})`),
						h.P(h.Class("mt-2 text-muted-foreground"), g.Text(`→ renders: <script src="/static/app.js" defer></script>`)),
					),
					// src:end
				),
			),
		),

		// ── HTMX Patterns ───────────────────────────────
		section("htmx-patterns", "HTMX Patterns",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("htmx-search", "Live search input",
					// src:htmx-search
					h.Div(
						uiform.Input(uiform.InputProps{
							ID:          "search-users",
							Name:        "q",
							Placeholder: "Search users...",
							Extra: htmx.Attrs(htmx.AttrsProps{
								Get:     demo.PathSearch,
								Target:  "#search-results",
								Trigger: "input changed delay:300ms",
							}),
						}),
						h.Div(h.ID("search-results")),
					),
					// src:end
					capHTMX,
				),
				example("htmx-validation", "Inline validation",
					// src:htmx-validation
					h.Div(
						uiform.Input(uiform.InputProps{
							ID:   "validate-email",
							Name: "email",
							Type: uiform.TypeEmail,
							Extra: htmx.Attrs(htmx.AttrsProps{
								Get:     demo.PathValidate + "?field=email",
								Target:  "#validate-field",
								Trigger: "blur",
							}),
						}),
						h.Div(h.ID("validate-field")),
					),
					// src:end
					capHTMX,
				),
				example("htmx-paginated", "Paginated table",
					// src:htmx-paginated
					h.Div(
						h.ID("paginate-region"),
						core.Button(core.ButtonProps{
							Label:   "Load first page",
							Variant: core.VariantOutline,
							Size:    core.SizeSm,
							Extra: htmx.Attrs(htmx.AttrsProps{
								Get:    demo.PathPaginate + "?page=1&per_page=5",
								Target: "#paginate-region",
								Swap:   "outerHTML",
							}),
						}),
					),
					// src:end
					capHTMX,
				),
				example("htmx-dependent", "Dependent select",
					// src:htmx-dependent
					h.Div(
						uiform.Select(uiform.SelectProps{
							ID:   "country",
							Name: "country",
							Options: []uiform.Option{
								{Value: "", Label: "Select a country…"},
								{Value: "se", Label: "Sweden"},
								{Value: "us", Label: "United States"},
								{Value: "de", Label: "Germany"},
							},
							Extra: htmx.Attrs(htmx.AttrsProps{
								Get:     demo.PathRegion,
								Target:  "#region-field",
								Trigger: "change",
								Params:  "country",
							}),
						}),
						h.Div(h.ID("region-field")),
					),
					// src:end
					capHTMX,
				),
				example("htmx-oob", "OOB toast on success",
					// src:htmx-oob
					h.Div(
						h.Class("space-y-3"),
						h.P(
							h.Class("text-xs text-muted-foreground"),
							g.Text("HTMX can update multiple page regions from one response. The server returns the toast with "),
							h.Code(h.Class("font-mono"), g.Text("hx-swap-oob")),
							g.Text(" so it is appended to the fixed #toast-container regardless of the primary swap target."),
						),
						core.Button(core.ButtonProps{
							Label:   "Trigger OOB toast",
							Variant: core.VariantOutline,
							Size:    core.SizeSm,
							Extra: htmx.Attrs(htmx.AttrsProps{
								Get:  demo.PathOOBToast,
								Swap: htmx.SwapNone,
							}),
						}),
					),
					// src:end
					capHTMX,
				),
			),
		),

		// ── Form Fields ──────────────────────────────────
		section("form-fields", "Form Fields",
			example("form-text", "Text input",
				// src:form-text
				uiform.Input(uiform.InputProps{
					ID:          "ex-text",
					Name:        "username",
					Placeholder: "e.g. alice",
				}),
				// src:end
			),
			example("form-email", "Email input (required)",
				// src:form-email
				uiform.Input(uiform.InputProps{
					ID:       "ex-email",
					Name:     "email",
					Type:     uiform.TypeEmail,
					Required: true,
				}),
				// src:end
			),
			example("form-error", "Input with error state",
				// src:form-error
				uiform.Input(uiform.InputProps{
					ID:       "ex-error",
					Name:     "password",
					Type:     uiform.TypePassword,
					Value:    "short",
					HasError: true,
				}),
				// src:end
			),
			example("form-select", "Select",
				// src:form-select
				uiform.Select(uiform.SelectProps{
					ID:       "ex-role",
					Name:     "role",
					Selected: "editor",
					Options: []uiform.Option{
						{Value: "admin", Label: "Admin"},
						{Value: "editor", Label: "Editor"},
						{Value: "viewer", Label: "Viewer"},
					},
				}),
				// src:end
			),
			example("form-switch", "Switch (toggle)",
				// src:form-switch
				h.Label(
					h.Class("flex items-center gap-2 text-sm cursor-pointer"),
					uiform.Switch(uiform.SwitchProps{
						ID:      "ex-switch",
						Name:    "enabled",
						Checked: true,
					}),
					g.Text("Enable feature"),
				),
				// src:end
			),
			example("form-textarea", "Textarea",
				// src:form-textarea
				uiform.Textarea(uiform.TextareaProps{
					ID:          "ex-bio",
					Name:        "bio",
					Placeholder: "Tell us about yourself…",
					Rows:        4,
				}),
				// src:end
			),
			example("form-fieldset", "FieldSet — radio group",
				// src:form-fieldset
				uiform.FieldSet(uiform.FieldSetProps{
					ID:     "ex-contact",
					Legend: "Preferred contact",
				},
					h.Label(h.Class("flex items-center gap-2 text-sm cursor-pointer"),
						uiform.Radio(uiform.RadioProps{ID: "ex-contact-email", Name: "contact", Value: "email", Checked: true}),
						g.Text("Email"),
					),
					h.Label(h.Class("flex items-center gap-2 text-sm cursor-pointer"),
						uiform.Radio(uiform.RadioProps{ID: "ex-contact-phone", Name: "contact", Value: "phone"}),
						g.Text("Phone"),
					),
				),
				// src:end
			),
			example("form-optgroups", "Select with opt-groups",
				// src:form-optgroups
				uiform.Select(uiform.SelectProps{
					ID:       "ex-role-grouped",
					Name:     "role",
					Selected: "admin",
					Groups: []uiform.OptGroup{
						{Label: "Admin roles", Options: []uiform.Option{
							{Value: "admin", Label: "Admin"},
							{Value: "superadmin", Label: "Super Admin"},
						}},
						{Label: "Member roles", Options: []uiform.Option{
							{Value: "editor", Label: "Editor"},
							{Value: "viewer", Label: "Viewer"},
						}},
					},
				}),
				// src:end
			),
			example("form-upload-single", "File upload (single)",
				// src:form-upload-single
				uiform.FileUpload(uiform.FileUploadProps{
					ID:     "ex-upload",
					Name:   "file",
					Accept: "image/*,application/pdf",
					Prompt: "Drop an image or PDF, or click to browse",
				}),
				// src:end
			),
			example("form-upload-multi", "File upload (multiple)",
				// src:form-upload-multi
				uiform.FileUpload(uiform.FileUploadProps{
					ID:       "ex-upload-multi",
					Name:     "files",
					Multiple: true,
				}),
				// src:end
			),
		),

		// ── Labels ──────────────────────────────────────
		section("labels", "Labels",
			example("labels-default", "Default",
				// src:labels-default
				core.Label(core.LabelProps{For: "ex-input"}, g.Text("Email address")),
				// src:end
			),
			example("labels-error", "Error state",
				// src:labels-error
				core.Label(core.LabelProps{For: "ex-input-err", Error: "Required"}, g.Text("Password")),
				// src:end
			),
		),

		// ── Pagination ──────────────────────────────────
		section("pagination", "Pagination",
			example("pagination-five", "Five-page example (page 3 active)",
				// src:pagination-five
				pagination.Root(
					pagination.Content(
						pagination.Item(pagination.Previous("/users?page=2", false)),
						pagination.Item(pagination.Link("/users?page=1", false, g.Text("1"))),
						pagination.Item(pagination.Link("/users?page=2", false, g.Text("2"))),
						pagination.Item(pagination.Link("/users?page=3", true, g.Text("3"))),
						pagination.Item(pagination.Link("/users?page=4", false, g.Text("4"))),
						pagination.Item(pagination.Link("/users?page=5", false, g.Text("5"))),
						pagination.Item(pagination.Next("/users?page=4", false)),
					),
				),
				// src:end
			),
		),

		// ── Pager ───────────────────────────────────────
		section("pager", "Pager",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("pager-math", "Pagination math",
					// src:pager-math
					h.Div(
						h.Class("font-mono text-xs text-muted-foreground space-y-1"),
						g.Text("p := pager.New(r, 20, 100)"),
						h.Br(),
						g.Text("p.SetTotal(243) // → TotalPages: 13"),
						h.Br(),
						g.Text("p.Offset()      // → SQL OFFSET"),
						h.Br(),
						g.Text("p.Limit()       // → SQL LIMIT (alias PerPage)"),
						h.Br(),
						g.Text("p.HasPages()    // → true when > 1 page"),
						h.Br(),
						g.Text("p.PageRange(2)  // → [1,-1,3,4,5,6,7,-1,13]"),
					),
					// src:end
				),
				example("pager-ui", "PageRange driving the Pagination UI",
					// src:pager-ui
					h.Div(h.Class("overflow-x-auto"), pagerShowcase()),
					// src:end
				),
			),
		),

		// ── Progress ────────────────────────────────────
		section("progress", "Progress",
			example("progress-default", "Default 60%",
				// src:progress-default
				feedback.Progress(feedback.ProgressProps{Value: 60, Label: "Loading…", ShowValue: true}),
				// src:end
			),
			example("progress-success", "Success 100%",
				// src:progress-success
				feedback.Progress(feedback.ProgressProps{Variant: feedback.ProgressSuccess, Value: 100, ShowValue: true}),
				// src:end
			),
			example("progress-danger", "Danger 25%",
				// src:progress-danger
				feedback.Progress(feedback.ProgressProps{Variant: feedback.ProgressDanger, Value: 25, ShowValue: true}),
				// src:end
			),
			example("progress-small", "Small",
				// src:progress-small
				feedback.Progress(feedback.ProgressProps{Size: feedback.ProgressSm, Value: 40}),
				// src:end
			),
		),

		// ── Separators ──────────────────────────────────
		section("separators", "Separators",
			example("separators-plain", "Horizontal (plain)",
				// src:separators-plain
				core.Separator(core.SeparatorProps{}),
				// src:end
			),
			example("separators-label", "Horizontal with label",
				// src:separators-label
				core.Separator(core.SeparatorProps{Label: "OR"}),
				// src:end
			),
			example("separators-dashed", "Dashed",
				// src:separators-dashed
				core.Separator(core.SeparatorProps{Decoration: core.DecorationDashed}),
				// src:end
			),
		),

		// ── Skeleton ────────────────────────────────────
		section("skeleton", "Skeleton",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("skeleton-avatar", "User card (avatar + text)",
					// src:skeleton-avatar
					h.Div(
						h.Class("flex items-center gap-4"),
						core.Skeleton("size-12 rounded-full shrink-0"),
						h.Div(h.Class("flex-1 space-y-2"),
							core.Skeleton("h-4 w-3/4"),
							core.Skeleton("h-3 w-1/2"),
						),
					),
					// src:end
				),
				example("skeleton-content", "Content block (title + body)",
					// src:skeleton-content
					h.Div(
						h.Class("space-y-3"),
						core.Skeleton("h-5 w-2/3"),
						core.Skeleton("h-3 w-full"),
						core.Skeleton("h-3 w-full"),
						core.Skeleton("h-3 w-4/5"),
					),
					// src:end
				),
			),
		),

		// ── Tables ──────────────────────────────────────
		section("tables", "Tables",
			example("tables-full", "Table with header/body/actions",
				// src:tables-full
				data.Table.Root(
					data.Table.Header(
						data.Table.Row(data.RowProps{},
							data.Table.Head(g.Text("Name")),
							data.Table.Head(g.Text("Role")),
							data.Table.Head(g.Text("Status")),
							data.Table.Head(g.Text("")),
						),
					),
					data.Table.Body(data.BodyProps{},
						data.Table.Row(data.RowProps{},
							data.Table.Cell(g.Text("Alice Johnson")),
							data.Table.Cell(g.Text("Admin")),
							data.Table.Cell(core.Badge(core.BadgeProps{Children: []g.Node{g.Text("Active")}})),
							data.Table.Cell(
								h.Div(h.Class("flex justify-end gap-2"),
									core.Button(core.ButtonProps{Label: "Edit", Variant: core.VariantGhost, Size: core.SizeSm}),
									core.Button(core.ButtonProps{Label: "Delete", Variant: core.VariantDestructiveGhost, Size: core.SizeSm}),
								),
							),
						),
						data.Table.Row(data.RowProps{},
							data.Table.Cell(g.Text("Bob Smith")),
							data.Table.Cell(g.Text("Editor")),
							data.Table.Cell(core.Badge(core.BadgeProps{Variant: core.BadgeSecondary, Children: []g.Node{g.Text("Inactive")}})),
							data.Table.Cell(
								h.Div(h.Class("flex justify-end gap-2"),
									core.Button(core.ButtonProps{Label: "Edit", Variant: core.VariantGhost, Size: core.SizeSm}),
									core.Button(core.ButtonProps{Label: "Delete", Variant: core.VariantDestructiveGhost, Size: core.SizeSm}),
								),
							),
						),
					),
					data.Table.Caption(g.Text("A list of team members.")),
				),
				// src:end
			),
		),

		// ── Theme ────────────────────────────────────────
		section("theme", "Theme",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("theme-selector", "Theme selector button",
					// src:theme-selector
					theme.ThemeSelector(theme.ThemeSelectorProps{}),
					// src:end
					capController,
				),
				example("theme-csp", "CSP hashes (for Content-Security-Policy)",
					// src:theme-csp
					h.Div(
						h.Class("font-mono text-xs text-muted-foreground space-y-2"),
						h.P(g.Text("InitScript hash:")),
						h.Code(h.Class("block text-xs break-all"), g.Text(theme.InitScriptCSPHash)),
					),
					// src:end
				),
			),
		),

		// ── Tabs ────────────────────────────────────────
		section("tabs", "Tabs",
			example("tabs-three", "Three-tab panel",
				// src:tabs-three
				tabs.Root("account-tabs", "account",
					tabs.List(
						tabs.Trigger("account-tabs", "account", true, g.Text("Account")),
						tabs.Trigger("account-tabs", "password", false, g.Text("Password")),
						tabs.Trigger("account-tabs", "settings", false, g.Text("Settings")),
					),
					tabs.Content("account-tabs", "account", true,
						data.Card.Root(
							data.Card.Header(data.Card.Title(g.Text("Account"))),
							data.Card.Content(h.P(g.Text("Manage your account settings here."))),
						),
					),
					tabs.Content("account-tabs", "password", false,
						data.Card.Root(
							data.Card.Header(data.Card.Title(g.Text("Password"))),
							data.Card.Content(h.P(g.Text("Change your password here."))),
						),
					),
					tabs.Content("account-tabs", "settings", false,
						data.Card.Root(
							data.Card.Header(data.Card.Title(g.Text("Settings"))),
							data.Card.Content(h.P(g.Text("Manage your preferences here."))),
						),
					),
				),
				// src:end
				capController,
			),
		),

		// ── Toast ───────────────────────────────────────
		section("toast", "Toast",
			example("toast-variants", "Variants",
				// src:toast-variants
				h.Div(
					h.Class("flex flex-col gap-2"),
					feedback.Toast(feedback.ToastProps{Title: "Event created", Description: "Your event has been created.", Dismissible: true}),
					feedback.Toast(feedback.ToastProps{Title: "Success", Description: "Changes saved.", Variant: feedback.ToastSuccess, Dismissible: true}),
					feedback.Toast(feedback.ToastProps{Title: "Error", Description: "Something went wrong.", Variant: feedback.ToastError, Dismissible: true}),
					feedback.Toast(feedback.ToastProps{Title: "Warning", Description: "This action is irreversible.", Variant: feedback.ToastWarning, Dismissible: true}),
					feedback.Toast(feedback.ToastProps{Title: "Info", Description: "New updates are available.", Variant: feedback.ToastInfo, Dismissible: true}),
				),
				// src:end
			),
			example("toast-interactive", "Interactive — click to trigger (auto-dismisses after 5s)",
				// src:toast-interactive
				h.Div(
					h.Class("flex flex-wrap gap-2"),
					toastTriggerButton("toast-tmpl-default", "Default"),
					toastTriggerButton("toast-tmpl-success", "Success"),
					toastTriggerButton("toast-tmpl-error", "Error"),
					toastTriggerButton("toast-tmpl-warning", "Warning"),
					toastTriggerButton("toast-tmpl-info", "Info"),
					toastTemplate("toast-tmpl-default", feedback.ToastDefault, "Event created", "Your event has been created."),
					toastTemplate("toast-tmpl-success", feedback.ToastSuccess, "Success", "Changes saved successfully."),
					toastTemplate("toast-tmpl-error", feedback.ToastError, "Error", "Something went wrong."),
					toastTemplate("toast-tmpl-warning", feedback.ToastWarning, "Warning", "This action is irreversible."),
					toastTemplate("toast-tmpl-info", feedback.ToastInfo, "Info", "New updates are available."),
				),
				// src:end
				capController,
			),
		),

		// ── Popover ─────────────────────────────────────
		section("popover", "Popover",
			example("popover-default", "Default (left-aligned)",
				// src:popover-default
				core.Popover.Root(core.PopoverRootProps{},
					core.Popover.Trigger(core.PopoverTriggerProps{Class: popoverBtnClass},
						g.Text("Open popover"),
					),
					core.Popover.Content(core.PopoverContentProps{},
						h.P(h.Class("p-4"),
							h.Span(h.Class("block text-sm font-medium mb-1"), g.Text("Popover title")),
							h.Span(h.Class("text-sm text-muted-foreground"), g.Text("This is a generic floating panel. It closes when you click outside.")),
						),
					),
				),
				// src:end
			),
			example("popover-right", "Right-aligned",
				// src:popover-right
				core.Popover.Root(core.PopoverRootProps{},
					core.Popover.Trigger(core.PopoverTriggerProps{Class: popoverBtnClass},
						g.Text("Right-aligned"),
					),
					core.Popover.Content(core.PopoverContentProps{Align: "right"},
						h.P(h.Class("p-4 text-sm text-muted-foreground"), g.Text("Panel anchored to the right edge of the trigger.")),
					),
				),
				// src:end
			),
			example("popover-narrow", "Custom width",
				// src:popover-narrow
				core.Popover.Root(core.PopoverRootProps{},
					core.Popover.Trigger(core.PopoverTriggerProps{Class: popoverBtnClass},
						g.Text("Narrow popover"),
					),
					core.Popover.Content(core.PopoverContentProps{Width: "w-48"},
						h.P(h.Class("p-4 text-sm text-muted-foreground"), g.Text("w-48 panel.")),
					),
				),
				// src:end
			),
		),

		// ── Tooltip ─────────────────────────────────────
		section("tooltip", "Tooltip",
			example("tooltip-hover", "Hover or focus for tooltip",
				// src:tooltip-hover
				tooltip.Root(
					tooltip.Trigger(
						core.Button(core.ButtonProps{
							Label:   "Focus me",
							Variant: core.VariantOutline,
							Extra:   tooltip.TriggerAttrs("example-tooltip"),
						}),
					),
					tooltip.Content("example-tooltip", g.Text("This is a tooltip")),
				),
				// src:end
			),
			example("tooltip-click", "Click-activated (touch-friendly)",
				// src:tooltip-click
				tooltip.ClickRoot(
					tooltip.ClickTrigger(
						g.Attr("aria-describedby", "click-tooltip"),
						core.Icon(iconrender.PropsFor(icons.ChevronDown, core.IconProps{
							Size:  "size-5",
							Label: "Help",
						})),
					),
					tooltip.ClickContent("click-tooltip", g.Text("Click or tap to reveal this tooltip")),
				),
				// src:end
			),
		),
		// Fixed container for triggered toasts; appended to by data-toast-trigger handler.
		h.Div(h.ID("toast-container"), h.Class("fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm")),
	)
}

// popoverBtnClass applies outline-button styling to a <summary> trigger so it
// looks like a button without nesting an invalid <button> inside <summary>.
const popoverBtnClass = "gap-2 rounded-md border bg-background text-foreground shadow-xs hover:bg-accent hover:text-accent-foreground h-9 px-4 py-2 text-sm font-medium transition-all focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] outline-none"

// section renders an anchored <section> with a heading and divider.
func section(id, title string, content ...g.Node) g.Node {
	return h.Section(
		h.ID(id),
		h.Div(
			h.Class("flex items-center justify-between mb-4 scroll-mt-6"),
			h.H2(
				h.Class("text-lg font-semibold"),
				h.A(h.Href("#"+id), h.Class("hover:underline"), g.Text(title)),
			),
			h.A(
				h.Href("#top"),
				h.Class("text-xs text-muted-foreground hover:text-foreground hover:underline"),
				g.Text("↑"),
			),
		),
		h.Div(h.Class("space-y-4"), g.Group(content)),
		h.Hr(h.Class("mt-8 border-border")),
	)
}

// cap marks which progressive-enhancement tier an example demonstrates.
type cap int

const (
	capHTMX       cap = iota + 1 // Tier 2: HTMX-enhanced endpoint
	capController                // Tier 3: requires a JS controller
)

// example renders a named example box with a label, optional tier dots, optional source snippet, and the component.
func example(key, name string, node g.Node, caps ...cap) g.Node {
	var dots []g.Node
	for _, c := range caps {
		switch c {
		case capHTMX:
			dots = append(dots, h.Span(h.Class("flex size-3"),
				h.Span(h.Class("absolute inline-flex h-full w-full animate-ping rounded-full bg-amber-500 opacity-75")),
				h.Span(h.Class("relative inline-flex size-3 rounded-full bg-amber-600")),
			))
		case capController:
			dots = append(dots, h.Span(h.Class("flex size-3"),
				h.Span(h.Class("absolute inline-flex h-full w-full animate-ping rounded-full bg-sky-400 opacity-75")),
				h.Span(h.Class("relative inline-flex size-3 rounded-full bg-sky-500")),
			))
		}
	}
	var badges g.Node
	if len(dots) > 0 {
		badges = h.Span(h.Class("absolute -top-1.5 -right-1.5 z-10 flex gap-1"), g.Group(dots))
	}
	var codeBlock g.Node
	if code, ok := snippets[key]; ok {
		codeBlock = h.Details(
			h.Class("mb-3"),
			h.Summary(h.Class("text-xs font-mono text-muted-foreground cursor-pointer select-none"), g.Text("View source")),
			h.Pre(h.Class("mt-2 overflow-x-auto rounded-md bg-muted p-3 text-xs"),
				h.Code(g.Text(code)),
			),
		)
	}
	return data.Card.Root(
		h.Div(
			h.Class("relative p-4"),
			g.If(badges != nil, badges),
			h.P(h.Class("mb-3 text-xs font-mono text-muted-foreground"), g.Text(name)),
			g.If(codeBlock != nil, codeBlock),
			node,
		),
	)
}

// tocItem renders a single table-of-contents anchor link.
func tocItem(id, label string) g.Node {
	return h.Li(
		h.Class("break-inside-avoid"),
		h.A(h.Href("#"+id), h.Class("text-muted-foreground hover:text-foreground hover:underline"), g.Text(label)),
	)
}

// toastTriggerButton renders a button that clones a <template> toast into #toast-container.
func toastTriggerButton(templateID, label string) g.Node {
	return core.Button(core.ButtonProps{
		Label:   label,
		Variant: core.VariantOutline,
		Size:    core.SizeSm,
		Extra:   []g.Node{g.Attr("data-toast-trigger", templateID)},
	})
}

// toastTemplate renders a hidden <template> containing a toast for JS cloning.
// The toast carries data-controller="toast" and a 5-second auto-dismiss duration.
func toastTemplate(id string, variant feedback.ToastVariant, title, desc string) g.Node {
	return g.El("template", h.ID(id),
		feedback.Toast(feedback.ToastProps{
			Title:       title,
			Description: desc,
			Variant:     variant,
			Dismissible: true,
			Extra: []g.Node{
				g.Attr("data-controller", "toast"),
				g.Attr("data-toast-duration", "5000"),
			},
		}),
	)
}

// pagerShowcase builds a live pagination example using a hardcoded Pager state.
func pagerShowcase() g.Node {
	// Simulate page 5 of 10 with per_page=10, total=100
	p := &pager.Pager{Page: 5, PerPage: 10, TotalItems: 100, TotalPages: 10}
	pages := p.PageRange(2) // [1, -1, 3, 4, 5, 6, 7, -1, 10]

	var items []g.Node
	items = append(items, pagination.Item(pagination.Previous(
		fmt.Sprintf("/items?page=%d", p.PrevPage()), p.IsFirst(),
	)))
	for _, pg := range pages {
		if pg == -1 {
			items = append(items, pagination.Item(pagination.Ellipsis()))
		} else {
			items = append(items, pagination.Item(pagination.Link(
				fmt.Sprintf("/items?page=%d", pg),
				pg == p.Page,
				g.Text(fmt.Sprintf("%d", pg)),
			)))
		}
	}
	items = append(items, pagination.Item(pagination.Next(
		fmt.Sprintf("/items?page=%d", p.NextPage()), p.IsLast(),
	)))
	return pagination.Root(pagination.Content(items...))
}
