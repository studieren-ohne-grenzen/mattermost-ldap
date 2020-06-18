package ldapauthenticator

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"

	"github.com/go-ldap/ldap"
)

// Entry is a synonyme to go-ldap/ldap Entry
type Entry = ldap.Entry

// Authenticator holds the connection to the LDAP server as well as a given transformer to process retrieved entries.
type Authenticator struct {
	bindURL      string
	bindDN       string
	bindPassword string
	queryDN      string
	selectors    []string

	conn *ldap.Conn

	transformer Transformer
}

// NewAuthenticator creates a new Authenticator
func NewAuthenticator(bindDN, bindPassword, queryDN string, transformer Transformer) Authenticator {
	var authenticator Authenticator

	authenticator.bindDN = bindDN
	authenticator.bindPassword = bindPassword
	authenticator.queryDN = queryDN
	authenticator.transformer = transformer

	authenticator.selectors = transformer.Selectors()

	return authenticator
}

// Connection returns the current ldap connection
func (auth *Authenticator) Connection() *ldap.Conn {
	if auth.conn.IsClosing() {
		if err := auth.Connect(auth.bindURL); err != nil {
			// could not reconnect automatically.
			log.Fatal(err)
		}
	}

	return auth.conn
}

// Connect to bindURL ldap server and upgrade to TLS
func (auth *Authenticator) Connect(bindURL string) error {
	l, err := ldap.DialURL(bindURL)
	if err != nil {
		return err
	}

	// Upgrade connection to TLS
	err = l.StartTLS(&tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return err
	}

	auth.conn = l
	auth.bindURL = bindURL

	return nil
}

// Close the ldap connection
func (auth *Authenticator) Close() {
	auth.Connection().Close()
}

// Authenticate a user with username and passwort with given ldap server and return its uid
func (auth Authenticator) Authenticate(username, password string) (string, error) {
	entry, err := auth.searchForUser(username)
	if err != nil {
		return "", err
	}

	// Bind as the user to verify their password
	userdn := entry.DN
	err = auth.Connection().Bind(userdn, password)
	if err != nil {
		return "", err
	}

	defer auth.bindReadUser()

	return entry.GetAttributeValue("uid"), nil
}

// GetUserByID searches for the given user id and returns it if there is such a user.
func (auth Authenticator) GetUserByID(id string) (interface{}, error) {
	entry, err := auth.searchForUser(id)
	if err != nil {
		return nil, err
	}

	return auth.transformer.Transform(entry), nil
}

func (auth *Authenticator) searchForUser(uid string) (*ldap.Entry, error) {
	if auth.bindURL == "" {
		log.Fatal(errors.New("ran a query without connecting to the server"))
	}

	// The username and password we want to check
	bindusername := auth.bindDN
	bindpassword := auth.bindPassword

	// First bind with a read only user
	err := auth.Connection().Bind(bindusername, bindpassword)
	if err != nil {
		return nil, err
	}

	defer auth.bindReadUser()

	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		auth.queryDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprintf("(&(objectClass=organizationalPerson)(uid=%s))", uid),
		auth.selectors,
		nil)

	sr, err := auth.Connection().Search(searchRequest)
	if err != nil {
		return nil, err
	}

	if len(sr.Entries) != 1 {
		return nil, errors.New("user does not exist or there is technically more than one")
	}

	return sr.Entries[0], nil
}

func (auth *Authenticator) bindReadUser() {
	bindusername := auth.bindDN
	bindpassword := auth.bindPassword

	auth.Connection().Bind(bindusername, bindpassword)
}
