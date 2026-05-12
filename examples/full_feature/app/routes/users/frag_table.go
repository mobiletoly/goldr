package users

import (
	"net/http"

	"github.com/a-h/templ"
)

func FragTable(_ *http.Request) templ.Component {
	return FragTableView(ListContacts())
}
