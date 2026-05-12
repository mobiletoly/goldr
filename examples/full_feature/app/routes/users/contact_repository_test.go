package users

import "testing"

func TestContactRepositoryFindsContacts(t *testing.T) {
	contact, ok := ContactByID("42")
	if !ok {
		t.Fatal("ContactByID() ok = false, want true")
	}
	if contact.Name != "Ada Lovelace" {
		t.Fatalf("contact name = %q, want Ada Lovelace", contact.Name)
	}

	if _, ok := ContactByID("missing"); ok {
		t.Fatal("ContactByID(missing) ok = true, want false")
	}
}

func TestContactRepositoryListReturnsCopy(t *testing.T) {
	repository := newContactRepository([]Contact{
		{ID: "42", Name: "Ada Lovelace", Status: "Active"},
	})
	list := repository.List()
	if len(list) != 1 {
		t.Fatalf("len(List()) = %d, want 1", len(list))
	}

	list[0].Name = "Changed"
	again := repository.List()
	if again[0].Name != "Ada Lovelace" {
		t.Fatalf("List() returned mutable backing data")
	}
}

func TestContactRepositoryAddsContacts(t *testing.T) {
	repository := newContactRepository([]Contact{
		{ID: "42", Name: "Ada Lovelace", Status: "Active"},
	})

	contact := repository.Add("Hedy Lamarr", "Inactive")

	if contact.ID != "43" {
		t.Fatalf("contact ID = %q, want 43", contact.ID)
	}
	if contact.Name != "Hedy Lamarr" {
		t.Fatalf("contact name = %q, want Hedy Lamarr", contact.Name)
	}
	if contact.Status != "Inactive" {
		t.Fatalf("contact status = %q, want Inactive", contact.Status)
	}
	if _, ok := repository.ByID("43"); !ok {
		t.Fatal("ByID(43) ok = false, want true")
	}
}
