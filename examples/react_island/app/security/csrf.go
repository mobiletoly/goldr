package security

import "github.com/mobiletoly/goldr/csrf"

var CSRF = mustCSRF()

func mustCSRF() *csrf.Guard {
	guard, err := csrf.New(csrf.Config{Secret: []byte("goldr-react-island-example-secret")})
	if err != nil {
		panic(err)
	}
	return guard
}
