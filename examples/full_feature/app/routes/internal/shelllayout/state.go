package shelllayout

import "github.com/mobiletoly/goldr"

var Key = goldr.NewLayoutKey[State]("full-feature.shell")

type State struct {
	ActiveNav Nav
}

type Nav string

const (
	NavUsers     Nav = "users"
	NavSettings  Nav = "settings"
	NavProtected Nav = "protected"
	NavSignIn    Nav = "sign-in"
)
