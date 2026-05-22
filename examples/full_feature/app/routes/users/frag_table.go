package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

const (
	statusFilterQuery    = "status"
	statusFilterActive   = "active"
	statusFilterInactive = "inactive"
)

func FragTable(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(FragTableView(filteredContacts(r.URL.Query().Get(statusFilterQuery))))
}

func filteredContacts(filter string) []Contact {
	contacts := ListContacts()
	switch filter {
	case statusFilterActive:
		return contactsWithStatus(contacts, statusActive)
	case statusFilterInactive:
		return contactsWithStatus(contacts, statusInactive)
	default:
		return contacts
	}
}

func contactsWithStatus(contacts []Contact, status string) []Contact {
	var filtered []Contact
	for _, contact := range contacts {
		if contact.Status == status {
			filtered = append(filtered, contact)
		}
	}
	return filtered
}
