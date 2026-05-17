package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func FragTable(_ *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(FragTableView(ListContacts()))
}
