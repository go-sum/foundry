package authweb

import (
	"fmt"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/componentry/form"
	"github.com/go-sum/foundry/pkg/componentry/ui/core"
	"github.com/go-sum/foundry/pkg/componentry/ui/data"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/render"
	"github.com/go-sum/foundry/pkg/web/secure"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// usersListView builds the admin users list page content.
func usersListView(d UsersListPageData) g.Node {
	rows := make([]g.Node, len(d.Users))
	for i, u := range d.Users {
		rows[i] = userRow(u)
	}

	return h.Div(
		h.Class("space-y-6"),
		h.Div(
			h.Class("flex flex-col gap-1"),
			h.H1(h.Class("text-3xl font-bold tracking-tight"), g.Text("Users")),
			h.P(h.Class("text-muted-foreground"), g.Textf("%d total", d.Total)),
		),
		data.Table.Root(
			data.Table.Header(
				data.Table.Row(data.RowProps{},
					data.Table.Head(g.Text("Email")),
					data.Table.Head(g.Text("Display Name")),
					data.Table.Head(g.Text("Role")),
					data.Table.Head(g.Text("Actions")),
				),
			),
			data.Table.Body(data.BodyProps{ID: "users-body"}, rows...),
		),
	)
}

// userRow renders a single <tr> for a user. The row ID enables HTMX swapping.
func userRow(u auth.User) g.Node {
	editURL := fmt.Sprintf("/admin/users/%s/edit", u.ID)
	deleteURL := fmt.Sprintf("/admin/users/%s", u.ID)
	return data.Table.Row(data.RowProps{
		Extra: []g.Node{h.ID("user-" + u.ID.String())},
	},
		data.Table.Cell(g.Text(u.Email)),
		data.Table.Cell(g.Text(u.DisplayName)),
		data.Table.Cell(g.Text(string(u.Role))),
		data.Table.Cell(
			h.Div(h.Class("flex items-center gap-2"),
				core.Button(core.ButtonProps{
					Variant: core.VariantOutline,
					Size:    core.SizeSm,
					Label:   "Edit",
					Href:    editURL,
					Extra: []g.Node{
						render.HXGet(editURL),
						render.HXTarget("#user-" + u.ID.String()),
						render.HXSwap("outerHTML"),
					},
				}),
				core.Button(core.ButtonProps{
					Variant: core.VariantDestructiveGhost,
					Size:    core.SizeSm,
					Label:   "Delete",
					Extra: []g.Node{
						g.Attr("hx-delete", deleteURL),
						render.HXTarget("#user-" + u.ID.String()),
						render.HXSwap("outerHTML"),
						g.Attr("hx-confirm", "Are you sure you want to delete this user?"),
					},
				}),
			),
		),
	)
}

// userEditView builds the admin user edit form.
func userEditView(c *web.Context, d UserEditPageData) g.Node {
	updateURL := fmt.Sprintf("/admin/users/%s", d.User.ID)

	return data.Card.Root(
		data.Card.Header(
			data.Card.Title(g.Text("Edit User")),
			data.Card.Description(g.Text(d.User.Email)),
		),
		data.Card.Content(
			h.Form(
				h.Method("POST"),
				h.Action(updateURL),
				render.CSRFField(render.CSRFProps{
					Token:     secure.CSRFToken(c),
					FieldName: secure.CSRFFieldName(c),
				}),
				h.Input(h.Type("hidden"), h.Name("_method"), h.Value("PATCH")),
				h.Div(h.Class("grid gap-4"),
					form.Field(form.FieldProps{
						ID:     "email",
						Label:  "Email",
						Errors: fieldErrors(d.Errors, "email"),
						Control: form.Input(form.InputProps{
							ID:       "email",
							Name:     "email",
							Type:     form.TypeEmail,
							Value:    d.User.Email,
							HasError: hasFieldError(d.Errors, "email"),
						}),
					}),
					form.Field(form.FieldProps{
						ID:     "display_name",
						Label:  "Display name",
						Errors: fieldErrors(d.Errors, "display_name"),
						Control: form.Input(form.InputProps{
							ID:       "display_name",
							Name:     "display_name",
							Type:     form.TypeText,
							Value:    d.User.DisplayName,
							HasError: hasFieldError(d.Errors, "display_name"),
						}),
					}),
					form.Field(form.FieldProps{
						ID:     "role",
						Label:  "Role",
						Errors: fieldErrors(d.Errors, "role"),
						Control: form.Select(form.SelectProps{
							ID:       "role",
							Name:     "role",
							Selected: string(d.User.Role),
							HasError: hasFieldError(d.Errors, "role"),
							Options: []form.Option{
								{Value: string(auth.RoleUser), Label: "User"},
								{Value: string(auth.RoleAdmin), Label: "Admin"},
							},
						}),
					}),
					h.Div(h.Class("flex gap-2"),
						core.Button(core.ButtonProps{
							Type:  "submit",
							Label: "Save changes",
						}),
						core.Button(core.ButtonProps{
							Variant: core.VariantOutline,
							Label:   "Cancel",
							Href:    "/admin/users",
						}),
					),
				),
			),
		),
	)
}

// bootstrapView builds the admin bootstrap page.
func bootstrapView(c *web.Context, d BootstrapPageData) g.Node {
	if d.HasAdmin {
		return data.Card.Root(
			data.Card.Header(
				data.Card.Title(g.Text("Admin Bootstrap")),
			),
			data.Card.Content(
				h.P(h.Class("text-muted-foreground"),
					g.Text("An admin account already exists. No further bootstrap is needed."),
				),
			),
			data.Card.Footer(
				core.Button(core.ButtonProps{
					Variant: core.VariantOutline,
					Label:   "Go to admin",
					Href:    "/admin/users",
				}),
			),
		)
	}

	return data.Card.Root(
		data.Card.Header(
			data.Card.Title(g.Text("Admin Bootstrap")),
			data.Card.Description(g.Text("No admin account exists yet. Elevate your account to admin.")),
		),
		data.Card.Content(
			h.Form(
				h.Method("POST"),
				h.Action("/admin/elevate"),
				render.CSRFField(render.CSRFProps{
					Token:     secure.CSRFToken(c),
					FieldName: secure.CSRFFieldName(c),
				}),
				core.Button(core.ButtonProps{
					Type:      "submit",
					Label:     "Elevate to admin",
					FullWidth: true,
				}),
			),
		),
	)
}
