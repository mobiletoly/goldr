package users

type contactForm struct {
	Name        string
	Status      string
	NameError   string
	StatusError string
}

func (form contactForm) hasErrors() bool {
	return form.NameError != "" || form.StatusError != ""
}

func validateContactForm(form contactForm) contactForm {
	if form.Name == "" {
		form.NameError = "Name is required."
	}
	switch form.Status {
	case statusActive, statusInactive:
	default:
		form.StatusError = "Choose a valid status."
	}
	return form
}
