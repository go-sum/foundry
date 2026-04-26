package authui

import (
	"github.com/go-sum/auth"
	"github.com/go-sum/web"
	"github.com/go-sum/web/htmx"
	"github.com/go-sum/web/render"
)

type adminRenderer struct {
	cfg Config
}

// NewAdminRenderer returns an auth.AdminRenderer that builds views with
// componentry and delegates full-page layout to cfg.Page.
func NewAdminRenderer(cfg Config) auth.AdminRenderer {
	return &adminRenderer{cfg: cfg}
}

// UsersListPage renders the admin user list.
func (r *adminRenderer) UsersListPage(c *web.Context, data auth.UsersListPageData) (web.Response, error) {
	content := usersListView(data)
	if htmx.IsHTMX(c) && !htmx.IsBoosted(c) {
		return render.Fragment(content)
	}
	return r.cfg.Page(c, "Admin: Users", content)
}

// UserEditPage renders the admin user edit form.
func (r *adminRenderer) UserEditPage(c *web.Context, data auth.UserEditPageData) (web.Response, error) {
	content := userEditView(c, data)
	if htmx.IsHTMX(c) && !htmx.IsBoosted(c) {
		return render.Fragment(content)
	}
	return r.cfg.Page(c, "Edit User", content)
}

// UserRowFragment renders a single user table row for HTMX swapping.
func (r *adminRenderer) UserRowFragment(c *web.Context, user auth.User) (web.Response, error) {
	return render.Fragment(userRow(user))
}

// BootstrapPage renders the admin bootstrap page.
func (r *adminRenderer) BootstrapPage(c *web.Context, data auth.BootstrapPageData) (web.Response, error) {
	content := bootstrapView(c, data)
	if htmx.IsHTMX(c) && !htmx.IsBoosted(c) {
		return render.Fragment(content)
	}
	return r.cfg.Page(c, "Admin Bootstrap", content)
}
