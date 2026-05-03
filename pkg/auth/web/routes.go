package authweb

import (
	"cmp"
	"context"
	"errors"

	auth "github.com/go-sum/foundry/pkg/auth"
	authn "github.com/go-sum/foundry/pkg/web/authn"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/google/uuid"
)

// RouteSpec pairs a URL pattern with its router name. Both fields are required;
// a zero-value RouteSpec is replaced by the default during module construction.
type RouteSpec struct {
	Pattern string
	Name    string
}

// RouteConfig holds the URL pattern and route name for every auth endpoint.
// Pass a zero-value RouteConfig (or omit the field entirely) to use
// DefaultRouteConfig. Populate individual fields to override specific endpoints.
type RouteConfig struct {
	Signin       RouteSpec
	SigninPost   RouteSpec
	Signup       RouteSpec
	SignupPost   RouteSpec
	Verify       RouteSpec
	VerifyPost   RouteSpec
	VerifyResend RouteSpec
	Signout      RouteSpec

	PasskeyBeginAuth  RouteSpec
	PasskeyFinishAuth RouteSpec
	PasskeyList       RouteSpec
	PasskeyBeginReg   RouteSpec
	PasskeyFinishReg  RouteSpec
	PasskeyShow       RouteSpec
	PasskeyRenameForm RouteSpec
	PasskeyRename     RouteSpec
	PasskeyDelete     RouteSpec

	EmailChangeShow RouteSpec
	EmailChangePost RouteSpec

	AdminUsers         RouteSpec
	AdminUserShow      RouteSpec
	AdminUserEdit      RouteSpec
	AdminUserUpdate    RouteSpec
	AdminUserDelete    RouteSpec
	AdminBootstrap     RouteSpec
	AdminBootstrapPost RouteSpec
}

// DefaultRouteConfig returns the conventional auth route patterns.
func DefaultRouteConfig() RouteConfig { return applyRouteDefaults(RouteConfig{}) }

// applyRouteDefaults fills any zero-value RouteSpec with its default.
// cmp.Or returns the first non-zero comparable value, so a fully-zero
// RouteConfig transparently adopts all defaults, and a partial override
// keeps only the fields that were set.
func applyRouteDefaults(r RouteConfig) RouteConfig {
	return RouteConfig{
		Signin:       cmp.Or(r.Signin, RouteSpec{"/auth/signin", "auth.signin"}),
		SigninPost:   cmp.Or(r.SigninPost, RouteSpec{"/auth/signin", "auth.signin.post"}),
		Signup:       cmp.Or(r.Signup, RouteSpec{"/auth/signup", "auth.signup"}),
		SignupPost:   cmp.Or(r.SignupPost, RouteSpec{"/auth/signup", "auth.signup.post"}),
		Verify:       cmp.Or(r.Verify, RouteSpec{"/auth/verify", "auth.verify"}),
		VerifyPost:   cmp.Or(r.VerifyPost, RouteSpec{"/auth/verify", "auth.verify.post"}),
		VerifyResend: cmp.Or(r.VerifyResend, RouteSpec{"/auth/verify/resend", "auth.verify.resend"}),
		Signout:      cmp.Or(r.Signout, RouteSpec{"/auth/signout", "auth.signout"}),

		PasskeyBeginAuth:  cmp.Or(r.PasskeyBeginAuth, RouteSpec{"/auth/passkey/authenticate/begin", "auth.passkey.begin_auth"}),
		PasskeyFinishAuth: cmp.Or(r.PasskeyFinishAuth, RouteSpec{"/auth/passkey/authenticate/finish", "auth.passkey.finish_auth"}),
		PasskeyList:       cmp.Or(r.PasskeyList, RouteSpec{"/account/passkeys", "auth.passkey.list"}),
		PasskeyBeginReg:   cmp.Or(r.PasskeyBeginReg, RouteSpec{"/account/passkeys/register/begin", "auth.passkey.begin_reg"}),
		PasskeyFinishReg:  cmp.Or(r.PasskeyFinishReg, RouteSpec{"/account/passkeys/register/finish", "auth.passkey.finish_reg"}),
		PasskeyShow:       cmp.Or(r.PasskeyShow, RouteSpec{"/account/passkeys/{id}", "auth.passkey.show"}),
		PasskeyRenameForm: cmp.Or(r.PasskeyRenameForm, RouteSpec{"/account/passkeys/{id}/edit", "auth.passkey.rename.form"}),
		PasskeyRename:     cmp.Or(r.PasskeyRename, RouteSpec{"/account/passkeys/{id}", "auth.passkey.rename"}),
		PasskeyDelete:     cmp.Or(r.PasskeyDelete, RouteSpec{"/account/passkeys/{id}", "auth.passkey.delete"}),

		EmailChangeShow: cmp.Or(r.EmailChangeShow, RouteSpec{"/account/email-change", "auth.email_change"}),
		EmailChangePost: cmp.Or(r.EmailChangePost, RouteSpec{"/account/email-change", "auth.email_change.post"}),

		AdminUsers:         cmp.Or(r.AdminUsers, RouteSpec{"/admin/users", "auth.admin.users"}),
		AdminUserShow:      cmp.Or(r.AdminUserShow, RouteSpec{"/admin/users/{id}", "auth.admin.user.show"}),
		AdminUserEdit:      cmp.Or(r.AdminUserEdit, RouteSpec{"/admin/users/{id}/edit", "auth.admin.user.edit"}),
		AdminUserUpdate:    cmp.Or(r.AdminUserUpdate, RouteSpec{"/admin/users/{id}", "auth.admin.user.update"}),
		AdminUserDelete:    cmp.Or(r.AdminUserDelete, RouteSpec{"/admin/users/{id}", "auth.admin.user.delete"}),
		AdminBootstrap:     cmp.Or(r.AdminBootstrap, RouteSpec{"/admin/elevate", "auth.admin.bootstrap"}),
		AdminBootstrapPost: cmp.Or(r.AdminBootstrapPost, RouteSpec{"/admin/elevate", "auth.admin.bootstrap.post"}),
	}
}

// Routes returns the declarative route tree for the auth module.
// The caller registers the returned nodes via router.Register(rt, authweb.Routes(m)...).
func Routes(m *Module) []router.Node {
	r := m.routes
	nodes := []router.Node{
		router.GET(r.Signin.Pattern, r.Signin.Name, m.AuthHandler.ShowSignin),
		router.POST(r.SigninPost.Pattern, r.SigninPost.Name, m.AuthHandler.BeginSignin),
		router.GET(r.Signup.Pattern, r.Signup.Name, m.AuthHandler.ShowSignup),
		router.POST(r.SignupPost.Pattern, r.SignupPost.Name, m.AuthHandler.BeginSignup),
		router.GET(r.Verify.Pattern, r.Verify.Name, m.AuthHandler.ShowVerify),
		router.POST(r.VerifyPost.Pattern, r.VerifyPost.Name, m.AuthHandler.Verify),
		router.POST(r.VerifyResend.Pattern, r.VerifyResend.Name, m.AuthHandler.Resend),
		router.POST(r.Signout.Pattern, r.Signout.Name, m.AuthHandler.Signout),
	}

	if m.PasskeyHandler != nil {
		nodes = append(nodes,
			// public passkey auth JSON endpoints
			router.POST(r.PasskeyBeginAuth.Pattern, r.PasskeyBeginAuth.Name, m.PasskeyHandler.BeginAuthentication),
			router.POST(r.PasskeyFinishAuth.Pattern, r.PasskeyFinishAuth.Name, m.PasskeyHandler.FinishAuthentication),
			// authenticated passkey management
			router.Layout(
				router.Use(authn.RequireAuth(m.signinPath)),
				router.GET(r.PasskeyList.Pattern, r.PasskeyList.Name, m.PasskeyHandler.List),
				router.POST(r.PasskeyBeginReg.Pattern, r.PasskeyBeginReg.Name, m.PasskeyHandler.BeginRegistration),
				router.POST(r.PasskeyFinishReg.Pattern, r.PasskeyFinishReg.Name, m.PasskeyHandler.FinishRegistration),
				router.GET(r.PasskeyShow.Pattern, r.PasskeyShow.Name, m.PasskeyHandler.Show),
				router.GET(r.PasskeyRenameForm.Pattern, r.PasskeyRenameForm.Name, m.PasskeyHandler.RenameForm),
				router.PATCH(r.PasskeyRename.Pattern, r.PasskeyRename.Name, m.PasskeyHandler.Rename),
				router.DELETE(r.PasskeyDelete.Pattern, r.PasskeyDelete.Name, m.PasskeyHandler.Delete),
			),
		)
	}

	nodes = append(nodes,
		// email change (auth required)
		router.Layout(
			router.Use(authn.RequireAuth(m.signinPath)),
			router.GET(r.EmailChangeShow.Pattern, r.EmailChangeShow.Name, m.AuthHandler.ShowEmailChange),
			router.POST(r.EmailChangePost.Pattern, r.EmailChangePost.Name, m.AuthHandler.BeginEmailChange),
		),
		// admin user management (admin role required)
		router.Layout(
			router.Use(authn.RequireAuth(m.signinPath), authn.LoadUserRole(userRoleReaderFn(m.userReader)), authn.RequireAdmin()),
			router.GET(r.AdminUsers.Pattern, r.AdminUsers.Name, m.AdminHandler.List),
			router.GET(r.AdminUserShow.Pattern, r.AdminUserShow.Name, m.AdminHandler.Show),
			router.GET(r.AdminUserEdit.Pattern, r.AdminUserEdit.Name, m.AdminHandler.EditForm),
			router.PATCH(r.AdminUserUpdate.Pattern, r.AdminUserUpdate.Name, m.AdminHandler.Update),
			router.DELETE(r.AdminUserDelete.Pattern, r.AdminUserDelete.Name, m.AdminHandler.Delete),
		),
		// admin bootstrap (auth required, not admin role — creates first admin)
		router.Layout(
			router.Use(authn.RequireAuth(m.signinPath)),
			router.GET(r.AdminBootstrap.Pattern, r.AdminBootstrap.Name, m.AdminHandler.ShowBootstrap),
			router.POST(r.AdminBootstrapPost.Pattern, r.AdminBootstrapPost.Name, m.AdminHandler.Bootstrap),
		),
	)

	return nodes
}

func userRoleReaderFn(users auth.UserReader) authn.RoleReaderFunc {
	return func(ctx context.Context, id uuid.UUID) (string, error) {
		user, err := users.GetUserByID(ctx, id)
		if errors.Is(err, auth.ErrUserNotFound) {
			return "", authn.ErrRoleNotFound
		}
		if err != nil {
			return "", err
		}
		return string(user.Role), nil
	}
}
