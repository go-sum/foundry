// Package showcase renders a living reference of every visual component in
// pkg/componentry/. It produces a pure g.Node with no HTTP or internal/ imports
// and follows the componentry module's documented tiered import DAG.
package showcase

import (
	"fmt"

	"github.com/go-sum/componentry/email"
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
	"github.com/go-sum/componentry/patterns/font"
	"github.com/go-sum/componentry/patterns/head"
	"github.com/go-sum/componentry/patterns/pager"
	"github.com/go-sum/componentry/ui/core"
	"github.com/go-sum/componentry/ui/data"
	"github.com/go-sum/componentry/ui/feedback"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

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
					tocItem("email", "Email Components"),
					tocItem("font-loading", "Font Loading"),
					tocItem("form-fields", "Form Fields"),
					tocItem("head-builder", "Head Builder"),
					tocItem("htmx-patterns", "HTMX Patterns"),
					tocItem("labels", "Labels"),
					tocItem("pagination", "Pagination"),
					tocItem("pager", "Pager"),
					tocItem("popover", "Popover"),
					tocItem("progressive-tiers", "Progressive Tiers"),
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

		section("progressive-tiers", "Progressive Tiers",
			h.Div(h.Class("grid gap-4 md:grid-cols-3"),
				example("Tier 1 — Native-first", h.P(g.Text("Server-rendered HTML with native platform behaviour first: dialog, details, forms, links, and CSS state."))),
				example("Tier 2 — HTMX patterns", h.P(g.Text("HTML-over-the-wire behaviours expressed with typed hx-* helpers so async UX stays local to the component markup."))),
				example("Tier 3 — Islands / custom elements", h.P(g.Text("Opt-in client islands for the few cases where native HTML plus HTMX is not enough; keep them narrow and predictable."))),
			),
		),

		// ── Accordion ───────────────────────────────────
		section("accordion", "Accordion",
			example("Three items", accordion.Root(
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
			)),
		),

		// ── Alerts ──────────────────────────────────────
		section("alerts", "Alerts",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("Default (dismissible)", feedback.Alert.Root(
					feedback.AlertProps{Variant: feedback.AlertDefault, Dismissible: true},
					feedback.Alert.Title(g.Text("Note")),
					feedback.Alert.Description(g.Text("Here is some helpful information.")),
				)),
				example("Destructive (dismissible)", feedback.Alert.Root(
					feedback.AlertProps{Variant: feedback.AlertDestructive, Dismissible: true},
					feedback.Alert.Title(g.Text("Error")),
					feedback.Alert.Description(g.Text("Something went wrong. Please try again.")),
				)),
			),
		),

		// ── Avatars ──────────────────────────────────────
		section("avatars", "Avatars",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("Fallback initials", h.Div(
					h.Class("flex gap-4"),
					core.Avatar.Fallback(g.Text("AB")),
				)),
				example("Lucide icon", core.Icon(iconrender.PropsFor(icons.ChevronDown, core.IconProps{
					Size:  "size-10",
					Label: "User account",
				}))),
			),
		),

		// ── Badges ──────────────────────────────────────
		section("badges", "Badges",
			example("Variants", h.Div(
				h.Class("flex flex-wrap gap-2"),
				core.Badge(core.BadgeProps{Children: []g.Node{g.Text("Default")}}),
				core.Badge(core.BadgeProps{Variant: core.BadgeSecondary, Children: []g.Node{g.Text("Secondary")}}),
				core.Badge(core.BadgeProps{Variant: core.BadgeDestructive, Children: []g.Node{g.Text("Destructive")}}),
				core.Badge(core.BadgeProps{Variant: core.BadgeOutline, Children: []g.Node{g.Text("Outline")}}),
			)),
		),

		// ── Breadcrumb ──────────────────────────────────
		section("breadcrumb", "Breadcrumb",
			example("Three-level path", breadcrumb.Root(
				breadcrumb.List(
					breadcrumb.Item(breadcrumb.Link("/", g.Text("Home"))),
					breadcrumb.Item(breadcrumb.Separator()),
					breadcrumb.Item(breadcrumb.Link("/users", g.Text("Users"))),
					breadcrumb.Item(breadcrumb.Separator()),
					breadcrumb.Item(breadcrumb.Page(g.Text("Alice Johnson"))),
				),
			)),
		),

		// ── Buttons ──────────────────────────────────────
		section("buttons", "Buttons",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("Variants", h.Div(
					h.Class("flex flex-wrap gap-2"),
					core.Button(core.ButtonProps{Label: "Default"}),
					core.Button(core.ButtonProps{Label: "Destructive", Variant: core.VariantDestructive}),
					core.Button(core.ButtonProps{Label: "Destructive Ghost", Variant: core.VariantDestructiveGhost}),
					core.Button(core.ButtonProps{Label: "Outline", Variant: core.VariantOutline}),
					core.Button(core.ButtonProps{Label: "Secondary", Variant: core.VariantSecondary}),
					core.Button(core.ButtonProps{Label: "Ghost", Variant: core.VariantGhost}),
					core.Button(core.ButtonProps{Label: "Link", Variant: core.VariantLink}),
				)),
				example("Sizes", h.Div(
					h.Class("flex flex-wrap items-center gap-2"),
					core.Button(core.ButtonProps{Label: "Large", Size: core.SizeLg}),
					core.Button(core.ButtonProps{Label: "Default"}),
					core.Button(core.ButtonProps{Label: "Small", Size: core.SizeSm}),
				)),
				example("Link (as <a>)", h.Div(
					h.Class("flex gap-2"),
					core.Button(core.ButtonProps{Label: "Go Home", Href: "/", Variant: core.VariantSecondary}),
					core.Button(core.ButtonProps{Label: "Users", Href: "/users", Variant: core.VariantGhost, Size: core.SizeSm}),
				)),
				example("Disabled", h.Div(
					h.Class("flex gap-2"),
					core.Button(core.ButtonProps{Label: "Disabled", Disabled: true}),
					core.Button(core.ButtonProps{Label: "Disabled Outline", Variant: core.VariantOutline, Disabled: true}),
				)),
			),
		),

		// ── Cards ───────────────────────────────────────
		section("cards", "Cards",
			example("Full card anatomy", data.Card.Root(
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
			)),
		),

		// ── Dialog ──────────────────────────────────────
		section("dialog", "Dialog",
			example("Modal dialog with trigger", dialog.Root(
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
			)),
		),

		// ── Dropdown ────────────────────────────────────
		section("dropdown", "Dropdown",
			example("Native summary trigger", dropdown.Root(
				dropdown.Props{},
				dropdown.Trigger(dropdown.TriggerProps{}, g.Text("Options")),
				dropdown.Content(
					dropdown.Label("Account"),
					dropdown.Item(dropdown.ItemProps{Label: "View Profile", Href: "#"}),
					dropdown.Item(dropdown.ItemProps{Label: "Edit Settings", Href: "#"}),
					dropdown.Separator(),
					dropdown.Item(dropdown.ItemProps{Label: "Sign Out", Href: "#"}),
				),
			)),
		),

		// ── Email Components ─────────────────────────────
		section("email", "Email Components",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("H1 heading", email.H1("Welcome to Acme")),
				example("H2 heading", email.H2("Your order is confirmed")),
				example("Paragraph", email.P("Thank you for signing up. Your account is now active and ready to use.")),
				example("CTA Button", email.Button("Get started", "https://example.com/start")),
				example("Inline link", email.A("View your account", "https://example.com/account")),
				example("Horizontal rule", email.HR()),
				example("Preview text (hidden)", email.PreviewText("Order confirmed — click to view your receipt.")),
			),
		),

		// ── Font Loading ─────────────────────────────────
		section("font-loading", "Font Loading",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("Google Fonts — nodes", h.Div(
					h.Class("font-mono text-xs text-muted-foreground space-y-1"),
					g.Text("Renders: preconnect + stylesheet link tags"),
					h.Br(),
					g.Text("CSP StyleSrc: fonts.googleapis.com"),
					h.Br(),
					g.Text("CSP FontSrc: fonts.gstatic.com"),
					g.Group(font.Nodes(font.Google("Inter"))),
				)),
				example("Bunny Fonts (GDPR-friendly) — nodes", h.Div(
					h.Class("font-mono text-xs text-muted-foreground space-y-1"),
					g.Text("Renders: preconnect + stylesheet link tags"),
					h.Br(),
					g.Text("Single origin: fonts.bunny.net"),
				)),
				example("Self-hosted @font-face", h.Div(
					h.Class("font-mono text-xs text-muted-foreground"),
					g.Text("Renders: preload link + @font-face <style> block"),
				)),
				example("CSP source collection", h.Div(
					h.Class("font-mono text-xs text-muted-foreground"),
					g.Text("CollectCSPSources merges all providers and deduplicates within each directive."),
				)),
			),
		),

		// ── Head Builder ─────────────────────────────────
		section("head-builder", "Head Builder",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("Full <head> structure", h.Div(
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
				)),
				example("Open Graph tags", h.Div(
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
				)),
				example("Stylesheet with SRI", h.Div(
					h.Class("font-mono text-xs text-muted-foreground"),
					g.Text(`head.CSS(head.Stylesheet{`),
					h.Br(),
					g.Text(`  Href: "/static/app.css",`),
					h.Br(),
					g.Text(`  Integrity: "sha384-…",`),
					h.Br(),
					g.Text(`})`),
					head.CSS(head.Stylesheet{Href: "/static/app.css"}),
				)),
				example("Script (deferred)", h.Div(
					h.Class("font-mono text-xs text-muted-foreground"),
					g.Text(`head.JS(head.Script{`),
					h.Br(),
					g.Text(`  Src: "/static/app.js",`),
					h.Br(),
					g.Text(`  Defer: true,`),
					h.Br(),
					g.Text(`})`),
					head.JS(head.Script{Src: "/static/app.js", Defer: true}),
				)),
			),
		),

		// ── HTMX Patterns ───────────────────────────────
		section("htmx-patterns", "HTMX Patterns",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("Live search input", uiform.Input(uiform.InputProps{
					ID:          "search-users",
					Name:        "q",
					Placeholder: "Search users...",
					Extra: htmx.Attrs(htmx.AttrsProps{
						Get:       "/users/search",
						Target:    "#search-results",
						Indicator: "#search-indicator",
						Trigger:   "input changed delay:300ms",
						PushURL:   "true",
					}),
				})),
				example("Inline validation", uiform.Input(uiform.InputProps{
					ID:    "validate-email",
					Name:  "email",
					Type:  uiform.TypeEmail,
					Extra: htmx.Attrs(htmx.AttrsProps{Post: "/validate/email", Target: "#email-field", Trigger: "blur"}),
				})),
				example("Paginated table link", core.Button(core.ButtonProps{
					Label:   "Next page",
					Variant: core.VariantGhost,
					Size:    core.SizeSm,
					Extra: htmx.Attrs(htmx.AttrsProps{
						Get:       "/users?page=2",
						Target:    "#users-table",
						Indicator: "#table-indicator",
						PushURL:   "true",
					}),
				})),
				example("Async dialog trigger", core.Button(core.ButtonProps{
					Label: "Load User Dialog",
					Extra: htmx.Attrs(htmx.AttrsProps{
						Get:    "/users/new",
						Target: "#async-user-dialog-body",
					}),
				})),
				example("Dependent select", uiform.Select(uiform.SelectProps{
					ID:   "country",
					Name: "country",
					Options: []uiform.Option{
						{Value: "se", Label: "Sweden"},
						{Value: "us", Label: "United States"},
					},
					Extra: htmx.Attrs(htmx.AttrsProps{
						Get:     "/regions",
						Target:  "#region-field",
						Trigger: "change",
					}),
				})),
				example("OOB toast on success", feedback.Toast(feedback.ToastProps{
					Description: "Saved in the background",
					Variant:     feedback.ToastSuccess,
					Dismissible: true,
				})),
			),
		),

		// ── Form Fields ──────────────────────────────────
		section("form-fields", "Form Fields",
			example("Text input", uiform.Input(uiform.InputProps{
				ID:          "ex-text",
				Name:        "username",
				Placeholder: "e.g. alice",
			})),
			example("Email input (required)", uiform.Input(uiform.InputProps{
				ID:       "ex-email",
				Name:     "email",
				Type:     uiform.TypeEmail,
				Required: true,
			})),
			example("Input with error state", uiform.Input(uiform.InputProps{
				ID:       "ex-error",
				Name:     "password",
				Type:     uiform.TypePassword,
				Value:    "short",
				HasError: true,
			})),
			example("Select", uiform.Select(uiform.SelectProps{
				ID:       "ex-role",
				Name:     "role",
				Selected: "editor",
				Options: []uiform.Option{
					{Value: "admin", Label: "Admin"},
					{Value: "editor", Label: "Editor"},
					{Value: "viewer", Label: "Viewer"},
				},
			})),
			example("Switch (toggle)", h.Label(
				h.Class("flex items-center gap-2 text-sm cursor-pointer"),
				uiform.Switch(uiform.SwitchProps{
					ID:      "ex-switch",
					Name:    "enabled",
					Checked: true,
				}),
				g.Text("Enable feature"),
			)),
			example("Textarea", uiform.Textarea(uiform.TextareaProps{
				ID:          "ex-bio",
				Name:        "bio",
				Placeholder: "Tell us about yourself…",
				Rows:        4,
			})),
			example("FieldSet — radio group", uiform.FieldSet(uiform.FieldSetProps{
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
			)),
			example("Select with opt-groups", uiform.Select(uiform.SelectProps{
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
			})),
			example("File upload (single)", uiform.FileUpload(uiform.FileUploadProps{
				ID:     "ex-upload",
				Name:   "file",
				Accept: "image/*,application/pdf",
				Prompt: "Drop an image or PDF, or click to browse",
			})),
			example("File upload (multiple)", uiform.FileUpload(uiform.FileUploadProps{
				ID:       "ex-upload-multi",
				Name:     "files",
				Multiple: true,
			})),
		),

		// ── Labels ──────────────────────────────────────
		section("labels", "Labels",
			example("Default", core.Label(core.LabelProps{For: "ex-input"}, g.Text("Email address"))),
			example("Error state", core.Label(core.LabelProps{For: "ex-input-err", Error: "Required"}, g.Text("Password"))),
		),

		// ── Pagination ──────────────────────────────────
		section("pagination", "Pagination",
			example("Five-page example (page 3 active)", pagination.Root(
				pagination.Content(
					pagination.Item(pagination.Previous("/users?page=2", false)),
					pagination.Item(pagination.Link("/users?page=1", false, g.Text("1"))),
					pagination.Item(pagination.Link("/users?page=2", false, g.Text("2"))),
					pagination.Item(pagination.Link("/users?page=3", true, g.Text("3"))),
					pagination.Item(pagination.Link("/users?page=4", false, g.Text("4"))),
					pagination.Item(pagination.Link("/users?page=5", false, g.Text("5"))),
					pagination.Item(pagination.Next("/users?page=4", false)),
				),
			)),
		),

		// ── Pager ───────────────────────────────────────
		section("pager", "Pager",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("Pagination math", h.Div(
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
				)),
				example("PageRange driving the Pagination UI", pagerShowcase()),
			),
		),

		// ── Progress ────────────────────────────────────
		section("progress", "Progress",
			example("Default 60%", feedback.Progress(feedback.ProgressProps{Value: 60, Label: "Loading…", ShowValue: true})),
			example("Success 100%", feedback.Progress(feedback.ProgressProps{Variant: feedback.ProgressSuccess, Value: 100, ShowValue: true})),
			example("Danger 25%", feedback.Progress(feedback.ProgressProps{Variant: feedback.ProgressDanger, Value: 25, ShowValue: true})),
			example("Small", feedback.Progress(feedback.ProgressProps{Size: feedback.ProgressSm, Value: 40})),
		),

		// ── Separators ──────────────────────────────────
		section("separators", "Separators",
			example("Horizontal (plain)", core.Separator(core.SeparatorProps{})),
			example("Horizontal with label", core.Separator(core.SeparatorProps{Label: "OR"})),
			example("Dashed", core.Separator(core.SeparatorProps{Decoration: core.DecorationDashed})),
		),

		// ── Skeleton ────────────────────────────────────
		section("skeleton", "Skeleton",
			example("Loading placeholders", h.Div(
				h.Class("space-y-2"),
				core.Skeleton(h.Class("h-4 w-[250px]")),
				core.Skeleton(h.Class("h-4 w-[200px]")),
				core.Skeleton(h.Class("h-4 w-[150px]")),
			)),
		),

		// ── Tables ──────────────────────────────────────
		section("tables", "Tables",
			example("Table with header/body/actions", data.Table.Root(
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
			)),
		),

		// ── Theme ────────────────────────────────────────
		section("theme", "Theme",
			h.Div(h.Class("grid gap-4 md:grid-cols-2"),
				example("Theme selector button", theme.ThemeSelector(theme.ThemeSelectorProps{})),
				example("CSP hashes (for Content-Security-Policy)", h.Div(
					h.Class("font-mono text-xs text-muted-foreground space-y-2"),
					h.P(g.Text("ThemeScript hash:")),
					h.Code(h.Class("block text-xs break-all"), g.Text(theme.ScriptCSPHash)),
					h.P(h.Class("mt-2"), g.Text("SelectorScript hash:")),
					h.Code(h.Class("block text-xs break-all"), g.Text(theme.SelectorScriptCSPHash)),
				)),
				example("ThemeScript (place in <head>)", h.Div(
					h.Class("font-mono text-xs text-muted-foreground"),
					g.Text("theme.ThemeScript() — inline <script> that reads"),
					h.Br(),
					g.Text("localStorage.themePreference and applies .dark"),
					h.Br(),
					g.Text("to <html> before first paint (prevents FOUC)."),
				)),
			),
		),

		// ── Tabs ────────────────────────────────────────
		section("tabs", "Tabs",
			example("Three-tab panel", tabs.Root("account-tabs", "account",
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
			)),
		),

		// ── Toast ───────────────────────────────────────
		section("toast", "Toast",
			example("Variants", h.Div(
				h.Class("flex flex-col gap-2"),
				feedback.Toast(feedback.ToastProps{Title: "Event created", Description: "Your event has been created.", Dismissible: true}),
				feedback.Toast(feedback.ToastProps{Title: "Success", Description: "Changes saved.", Variant: feedback.ToastSuccess, Dismissible: true}),
				feedback.Toast(feedback.ToastProps{Title: "Error", Description: "Something went wrong.", Variant: feedback.ToastError, Dismissible: true}),
				feedback.Toast(feedback.ToastProps{Title: "Warning", Description: "This action is irreversible.", Variant: feedback.ToastWarning, Dismissible: true}),
				feedback.Toast(feedback.ToastProps{Title: "Info", Description: "New updates are available.", Variant: feedback.ToastInfo, Dismissible: true}),
			)),
			example("Interactive — click to trigger (auto-dismisses after 5s)", h.Div(
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
			)),
		),

		// ── Popover ─────────────────────────────────────
		section("popover", "Popover",
			example("Default (left-aligned)", core.Popover.Root(core.PopoverRootProps{},
				core.Popover.Trigger(core.PopoverTriggerProps{Class: popoverBtnClass},
					g.Text("Open popover"),
				),
				core.Popover.Content(core.PopoverContentProps{},
					h.P(h.Class("p-4"),
						h.Span(h.Class("block text-sm font-medium mb-1"), g.Text("Popover title")),
						h.Span(h.Class("text-sm text-muted-foreground"), g.Text("This is a generic floating panel. It closes when you click outside.")),
					),
				),
			)),
			example("Right-aligned", core.Popover.Root(core.PopoverRootProps{},
				core.Popover.Trigger(core.PopoverTriggerProps{Class: popoverBtnClass},
					g.Text("Right-aligned"),
				),
				core.Popover.Content(core.PopoverContentProps{Align: "right"},
					h.P(h.Class("p-4 text-sm text-muted-foreground"), g.Text("Panel anchored to the right edge of the trigger.")),
				),
			)),
			example("Custom width", core.Popover.Root(core.PopoverRootProps{},
				core.Popover.Trigger(core.PopoverTriggerProps{Class: popoverBtnClass},
					g.Text("Narrow popover"),
				),
				core.Popover.Content(core.PopoverContentProps{Width: "w-48"},
					h.P(h.Class("p-4 text-sm text-muted-foreground"), g.Text("w-48 panel.")),
				),
			)),
		),

		// ── Tooltip ─────────────────────────────────────
		section("tooltip", "Tooltip",
			example("Hover or focus for tooltip", tooltip.Root(
				tooltip.Trigger(
					core.Button(core.ButtonProps{
						Label:   "Focus me",
						Variant: core.VariantOutline,
						Extra:   tooltip.TriggerAttrs("example-tooltip"),
					}),
				),
				tooltip.Content("example-tooltip", g.Text("This is a tooltip")),
			)),
			example("Click-activated (touch-friendly)", tooltip.ClickRoot(
				tooltip.ClickTrigger(
					g.Attr("aria-describedby", "click-tooltip"),
					core.Icon(iconrender.PropsFor(icons.ChevronDown, core.IconProps{
						Size:  "size-5",
						Label: "Help",
					})),
				),
				tooltip.ClickContent("click-tooltip", g.Text("Click or tap to reveal this tooltip")),
			)),
		),
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

// example renders a named example box with a label and the component.
func example(name string, node g.Node) g.Node {
	return data.Card.Root(
		h.Div(
			h.Class("p-4"),
			h.P(h.Class("mb-3 text-xs font-mono text-muted-foreground"), g.Text(name)),
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
func toastTemplate(id string, variant feedback.ToastVariant, title, desc string) g.Node {
	return g.El("template", h.ID(id),
		feedback.Toast(feedback.ToastProps{
			Title:       title,
			Description: desc,
			Variant:     variant,
			Dismissible: true,
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
