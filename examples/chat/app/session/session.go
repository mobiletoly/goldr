package session

import (
	"net/http"
	"net/url"
	"strings"
)

const nameCookie = "goldr_chat_name"

func Name(r *http.Request) string {
	cookie, err := r.Cookie(nameCookie)
	if err != nil {
		return ""
	}
	name, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(name)
}

func SetName(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     nameCookie,
		Value:    url.QueryEscape(name),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearName(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     nameCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
