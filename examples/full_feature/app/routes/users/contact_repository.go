package users

import (
	"strconv"
	"sync"
)

type Contact struct {
	ID             string
	Name           string
	Status         string
	AvatarFilename string
}

type contactRepository struct {
	mu       sync.Mutex
	contacts []Contact
}

var contacts = newContactRepository([]Contact{
	{ID: "42", Name: "Ada Lovelace", Status: "Active"},
	{ID: "7", Name: "Grace Hopper", Status: "Active"},
	{ID: "11", Name: "Katherine Johnson", Status: "Inactive"},
})

func newContactRepository(values []Contact) *contactRepository {
	copied := make([]Contact, len(values))
	copy(copied, values)
	return &contactRepository{contacts: copied}
}

func ListContacts() []Contact {
	return contacts.List()
}

func ContactByID(id string) (Contact, bool) {
	return contacts.ByID(id)
}

func AddContact(name, status, avatarFilename string) Contact {
	return contacts.Add(name, status, avatarFilename)
}

func resetContactsForTest() {
	contacts = newContactRepository([]Contact{
		{ID: "42", Name: "Ada Lovelace", Status: "Active"},
		{ID: "7", Name: "Grace Hopper", Status: "Active"},
		{ID: "11", Name: "Katherine Johnson", Status: "Active"},
	})
}

func (repository *contactRepository) List() []Contact {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	copied := make([]Contact, len(repository.contacts))
	copy(copied, repository.contacts)
	return copied
}

func (repository *contactRepository) ByID(id string) (Contact, bool) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	for _, contact := range repository.contacts {
		if contact.ID == id {
			return contact, true
		}
	}
	return Contact{}, false
}

func (repository *contactRepository) Add(name, status, avatarFilename string) Contact {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	contact := Contact{
		ID:             strconv.Itoa(repository.nextID()),
		Name:           name,
		Status:         status,
		AvatarFilename: avatarFilename,
	}
	repository.contacts = append(repository.contacts, contact)
	return contact
}

func (repository *contactRepository) nextID() int {
	next := 1
	for _, contact := range repository.contacts {
		id, err := strconv.Atoi(contact.ID)
		if err != nil {
			continue
		}
		if id >= next {
			next = id + 1
		}
	}
	return next
}
