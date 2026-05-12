package users

import (
	"bytes"
	"net/http"

	"github.com/mobiletoly/goldr/hx"
)

func renderSavePreview(w http.ResponseWriter, r *http.Request) {
	component := FragTableView(ListContacts())
	var body bytes.Buffer
	if err := component.Render(r.Context(), &body); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	hx.Trigger(w, "user:saved")
	hx.Retarget(w, "#users-table")
	hx.Reswap(w, "outerHTML")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = body.WriteTo(w)
}
