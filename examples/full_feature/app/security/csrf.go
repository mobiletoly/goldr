package security

import "github.com/mobiletoly/goldr/csrf"

// CSRF is the example application's CSRF guard.
//
// Real applications should load this secret from configuration.
var CSRF = mustCSRF()

func mustCSRF() *csrf.Guard {
	guard, err := csrf.New(csrf.Config{
		Secret: []byte("goldr-full-feature-example-csrf-secret"),
	})
	if err != nil {
		panic(err)
	}
	return guard
}
