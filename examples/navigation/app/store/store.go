package store

type Office struct {
	ID   string
	Name string
}

type Team struct {
	ID       string
	OfficeID string
	Name     string
}

type Customer struct {
	ID     string
	TeamID string
	Name   string
	Risk   string
}

type Store struct {
	Offices   map[string]Office
	Teams     map[string]Team
	Customers map[string]Customer
}

var Default = Store{
	Offices: map[string]Office{
		"sea": {ID: "sea", Name: "Seattle"},
	},
	Teams: map[string]Team{
		"hq-team":       {ID: "hq-team", Name: "HQ Team"},
		"regional-team": {ID: "regional-team", OfficeID: "sea", Name: "Regional Team"},
	},
	Customers: map[string]Customer{
		"contoso":   {ID: "contoso", TeamID: "hq-team", Name: "Contoso Retail", Risk: "high"},
		"northwind": {ID: "northwind", TeamID: "regional-team", Name: "Northwind Supply", Risk: "medium"},
	},
}

func (store Store) Office(id string) Office {
	return store.Offices[id]
}

func (store Store) Team(id string) Team {
	return store.Teams[id]
}

func (store Store) Customer(id string) Customer {
	return store.Customers[id]
}

func (store Store) TeamCustomers(teamID string) []Customer {
	var customers []Customer
	for _, customer := range store.Customers {
		if customer.TeamID == teamID {
			customers = append(customers, customer)
		}
	}
	return customers
}
