package security

import "net/http"

const (
	// DemoAuthCookie is the example app's cookie for protected-page demos.
	DemoAuthCookie = "goldr_demo_role"
	// RoleAdmin can view the protected admin page.
	RoleAdmin = "admin"
	// RoleMember is authenticated but forbidden from the admin page.
	RoleMember = "member"
)

// DemoRole returns the example app role stored on the request.
func DemoRole(r *http.Request) string {
	if r == nil {
		return ""
	}
	cookie, err := r.Cookie(DemoAuthCookie)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// SetDemoRole stores the example app role.
func SetDemoRole(w http.ResponseWriter, role string) {
	http.SetCookie(w, &http.Cookie{
		Name:     DemoAuthCookie,
		Value:    role,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearDemoRole clears the example app role.
func ClearDemoRole(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     DemoAuthCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
