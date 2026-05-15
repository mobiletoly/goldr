package users

import "github.com/mobiletoly/goldr/bind"

func validateContactForm(form bind.Form) bind.FieldErrors {
	var errors bind.FieldErrors
	if form.Value("name") == "" {
		errors.Add("name", "Name is required.")
	}
	switch form.Value("status") {
	case statusActive, statusInactive:
	default:
		errors.Add("status", "Choose a valid status.")
	}
	return errors
}
