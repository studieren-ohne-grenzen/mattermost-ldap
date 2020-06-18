// Package ldapauthenticator is a very simple LDAP Authenticator written in Go. It uses the example code from https://godoc.org/github.com/go-ldap/ldap
package ldapauthenticator

// Transformer transforms a ldap Entry to a proper datastructure
type Transformer interface {
	// Transform a single LDAP entry into some data type
	Transform(entry *Entry) interface{}

	// Selectors to use by this Transformer
	Selectors() []string
}
