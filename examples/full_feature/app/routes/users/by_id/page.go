package by_id

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/full_feature/app/routes/users"
)

func Page(r *http.Request) goldr.Page {
	id := r.PathValue("id")
	contact, ok := users.ContactByID(id)
	metadata := goldr.PageMetadata{
		Title:       "Unknown contact - Goldr Example",
		Description: "No contact exists for this route id.",
	}
	if ok {
		metadata.Title = contact.Name + " - Goldr Example"
		metadata.Description = "Contact details for " + contact.Name + "."
	}
	return goldr.RenderPage(PageView(id, contact.Name, contact.Status, ok), metadata)
}
