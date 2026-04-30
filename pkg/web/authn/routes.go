package authn

import "github.com/go-sum/foundry/pkg/web/router"

// Route name constants for all auth endpoints.
const (
	RouteSigninShow   = "auth.signin"
	RouteSigninPost   = "auth.signin.post"
	RouteSignupShow   = "auth.signup"
	RouteSignupPost   = "auth.signup.post"
	RouteVerifyShow   = "auth.verify"
	RouteVerifyPost   = "auth.verify.post"
	RouteVerifyResend = "auth.verify.resend"
	RouteSignout      = "auth.signout"

	RouteEmailChangeShow = "auth.email_change"
	RouteEmailChangePost = "auth.email_change.post"

	RoutePasskeyBeginAuth  = "auth.passkey.begin_auth"
	RoutePasskeyFinishAuth = "auth.passkey.finish_auth"
	RoutePasskeyBeginReg   = "auth.passkey.begin_reg"
	RoutePasskeyFinishReg  = "auth.passkey.finish_reg"
	RoutePasskeyList       = "auth.passkey.list"
	RoutePasskeyShow       = "auth.passkey.show"
	RoutePasskeyRename     = "auth.passkey.rename"
	RoutePasskeyDelete     = "auth.passkey.delete"

	RouteAdminUsers         = "auth.admin.users"
	RouteAdminUserShow      = "auth.admin.user.show"
	RouteAdminUserEdit      = "auth.admin.user.edit"
	RouteAdminUserUpdate    = "auth.admin.user.update"
	RouteAdminUserDelete    = "auth.admin.user.delete"
	RouteAdminBootstrap     = "auth.admin.bootstrap"
	RouteAdminBootstrapPost = "auth.admin.bootstrap.post"
)

// Routes returns the declarative route tree for the auth module.
// The caller registers the returned nodes via router.Register(rt, authn.Routes(m)...).
func Routes(m *Module) []router.Node {
	nodes := []router.Node{
		router.Group("/auth",
			router.GET("/signin", RouteSigninShow, m.AuthHandler.ShowSignin),
			router.POST("/signin", RouteSigninPost, m.AuthHandler.BeginSignin),
			router.GET("/signup", RouteSignupShow, m.AuthHandler.ShowSignup),
			router.POST("/signup", RouteSignupPost, m.AuthHandler.BeginSignup),
			router.GET("/verify", RouteVerifyShow, m.AuthHandler.ShowVerify),
			router.POST("/verify", RouteVerifyPost, m.AuthHandler.Verify),
			router.POST("/verify/resend", RouteVerifyResend, m.AuthHandler.Resend),
			router.POST("/signout", RouteSignout, m.AuthHandler.Signout),
		),
	}

	if m.PasskeyHandler != nil {
		// Public passkey authentication endpoints (JSON API).
		nodes = append(nodes,
			router.Group("/auth/passkey",
				router.POST("/authenticate/begin", RoutePasskeyBeginAuth, m.PasskeyHandler.BeginAuthentication),
				router.POST("/authenticate/finish", RoutePasskeyFinishAuth, m.PasskeyHandler.FinishAuthentication),
			),
		)

		// Authenticated passkey management endpoints.
		nodes = append(nodes,
			router.Group("/account/passkeys",
				router.Use(RequireAuth(m.signinPath)),
				router.GET("", RoutePasskeyList, m.PasskeyHandler.List),
				router.POST("/register/begin", RoutePasskeyBeginReg, m.PasskeyHandler.BeginRegistration),
				router.POST("/register/finish", RoutePasskeyFinishReg, m.PasskeyHandler.FinishRegistration),
				router.GET("/{id}", RoutePasskeyShow, m.PasskeyHandler.Show),
				router.GET("/{id}/edit", RoutePasskeyRename+".form", m.PasskeyHandler.RenameForm),
				router.PATCH("/{id}", RoutePasskeyRename, m.PasskeyHandler.Rename),
				router.DELETE("/{id}", RoutePasskeyDelete, m.PasskeyHandler.Delete),
			),
		)
	}

	// Authenticated email change endpoints.
	nodes = append(nodes,
		router.Group("/account",
			router.Use(RequireAuth(m.signinPath)),
			router.GET("/email-change", RouteEmailChangeShow, m.AuthHandler.ShowEmailChange),
			router.POST("/email-change", RouteEmailChangePost, m.AuthHandler.BeginEmailChange),
		),
	)

	// Admin user management (requires admin role).
	nodes = append(nodes,
		router.Group("/admin",
			router.Use(RequireAuth(m.signinPath), LoadUserRole(m.userReader), RequireAdmin()),
			router.GET("/users", RouteAdminUsers, m.AdminHandler.List),
			router.GET("/users/{id}", RouteAdminUserShow, m.AdminHandler.Show),
			router.GET("/users/{id}/edit", RouteAdminUserEdit, m.AdminHandler.EditForm),
			router.PATCH("/users/{id}", RouteAdminUserUpdate, m.AdminHandler.Update),
			router.DELETE("/users/{id}", RouteAdminUserDelete, m.AdminHandler.Delete),
		),
	)

	// Admin bootstrap (requires auth but not admin — creates the first admin).
	nodes = append(nodes,
		router.Group("/admin",
			router.Use(RequireAuth(m.signinPath)),
			router.GET("/elevate", RouteAdminBootstrap, m.AdminHandler.ShowBootstrap),
			router.POST("/elevate", RouteAdminBootstrapPost, m.AdminHandler.Bootstrap),
		),
	)

	return nodes
}
