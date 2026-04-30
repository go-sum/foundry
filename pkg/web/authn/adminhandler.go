package authn

import (
	"strconv"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"
	"github.com/google/uuid"
)

// AdminRenderer produces full-page and partial HTML responses for admin views.
// The host application implements this interface to control layout and styling.
type AdminRenderer interface {
	UsersListPage(c *web.Context, data UsersListPageData) (web.Response, error)
	UserEditPage(c *web.Context, data UserEditPageData) (web.Response, error)
	UserRowFragment(c *web.Context, user auth.User) (web.Response, error)
	BootstrapPage(c *web.Context, data BootstrapPageData) (web.Response, error)
}

// UsersListPageData carries state for rendering the admin users list view.
type UsersListPageData struct {
	Users   []auth.User
	Page    int
	PerPage int
	Total   int64
}

// UserEditPageData carries state for rendering the admin user edit form.
type UserEditPageData struct {
	User   auth.User
	Errors map[string]string
}

// BootstrapPageData carries state for rendering the admin bootstrap view.
type BootstrapPageData struct {
	HasAdmin bool
}

// AdminHandler handles HTTP requests for admin user management.
type AdminHandler struct {
	svc       *auth.AdminService
	router    *router.Router
	validator validate.Validator
	renderer  AdminRenderer
}

// List returns a paginated list of users.
func (h *AdminHandler) List(c *web.Context) (web.Response, error) {
	page, _ := strconv.Atoi(c.URL().Query().Get("page"))
	perPage, _ := strconv.Atoi(c.URL().Query().Get("per_page"))
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	users, err := h.svc.ListUsers(c.Context(), page, perPage)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	total, err := h.svc.CountUsers(c.Context())
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	return h.renderer.UsersListPage(c, UsersListPageData{
		Users:   users,
		Page:    page,
		PerPage: perPage,
		Total:   total,
	})
}

// Show returns a user row fragment for a single user.
func (h *AdminHandler) Show(c *web.Context) (web.Response, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return web.Response{}, web.ErrBadRequest("Invalid user ID")
	}

	user, err := h.svc.GetUser(c.Context(), id)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	return h.renderer.UserRowFragment(c, user)
}

// EditForm renders the user edit form.
func (h *AdminHandler) EditForm(c *web.Context) (web.Response, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return web.Response{}, web.ErrBadRequest("Invalid user ID")
	}

	user, err := h.svc.GetUser(c.Context(), id)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	return h.renderer.UserEditPage(c, UserEditPageData{User: user})
}

// Update applies changes to a user record and returns the updated user row fragment.
func (h *AdminHandler) Update(c *web.Context) (web.Response, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return web.Response{}, web.ErrBadRequest("Invalid user ID")
	}

	var input auth.UpdateUserInput
	if err := validate.Bind(h.validator, c.Request, &input); err != nil {
		user, getErr := h.svc.GetUser(c.Context(), id)
		if getErr != nil {
			return web.Response{}, mapServiceError(getErr)
		}
		return h.renderer.UserEditPage(c, UserEditPageData{User: user})
	}

	user, err := h.svc.UpdateUser(c.Context(), id, input)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	return h.renderer.UserRowFragment(c, user)
}

// Delete removes a user record.
func (h *AdminHandler) Delete(c *web.Context) (web.Response, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return web.Response{}, web.ErrBadRequest("Invalid user ID")
	}

	if err := h.svc.DeleteUser(c.Context(), id); err != nil {
		return web.Response{}, mapServiceError(err)
	}

	return web.Respond(200), nil
}

// ShowBootstrap renders the admin bootstrap page showing whether an admin exists.
func (h *AdminHandler) ShowBootstrap(c *web.Context) (web.Response, error) {
	hasAdmin, err := h.svc.HasAdmin(c.Context())
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	return h.renderer.BootstrapPage(c, BootstrapPageData{HasAdmin: hasAdmin})
}

// Bootstrap elevates the authenticated user to admin when no admin exists.
func (h *AdminHandler) Bootstrap(c *web.Context) (web.Response, error) {
	uid := UserID(c)
	if uid == "" {
		return web.Response{}, web.ErrUnauthorized("Not authenticated")
	}
	userID, err := uuid.Parse(uid)
	if err != nil {
		return web.Response{}, web.ErrBadRequest("Invalid session")
	}

	_, err = h.svc.ElevateToAdmin(c.Context(), userID)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	adminURL := h.router.MustReverse(RouteAdminUsers, nil)
	return web.SeeOther(adminURL), nil
}
