package routes

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/react_island/app/security"
	"github.com/mobiletoly/goldr/examples/react_island/app/urls"
)

const maxSaveBody = 4 << 10

type project struct {
	Name   string `json:"name"`
	Pinned bool   `json:"pinned"`
}

var projectStore = struct {
	sync.RWMutex
	project project
}{project: project{Name: "Goldr island", Pinned: false}}

var Route = goldr.RouteDef{
	Page: Page,
	Actions: goldr.Actions{
		goldr.HTTPAction(http.MethodPost, "/save", PostSave),
	},
}

func Page(_ *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(PageView(currentProject(), urls.Save.Path(), urls.About.Path()), goldr.PageMetadata{
		Title:       "React Island - Goldr Example",
		Description: "A bounded React editor inside a Goldr page.",
	})
}

func PostSave(w http.ResponseWriter, r *http.Request) {
	if err := security.CSRF.Validate(r, ""); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSaveBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var submitted project
	if err := decoder.Decode(&submitted); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	submitted.Name = strings.TrimSpace(submitted.Name)
	if submitted.Name == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"errors": map[string]string{"name": "Enter a project name."},
		})
		return
	}
	projectStore.Lock()
	projectStore.project = submitted
	projectStore.Unlock()
	writeJSON(w, http.StatusOK, map[string]project{"project": submitted})
}

func currentProject() project {
	projectStore.RLock()
	defer projectStore.RUnlock()
	return projectStore.project
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
