package main

import (
	"crypto/sha256"
	"encoding/binary"

	ldap "github.com/zonradkuse/go-ldap-authenticator"
)

// Transformer transforms a single LDAP entry into a user entity
type Transformer struct {
	UsernamePrefix string

	MailAttrName string
	CNAttrName   string
	UIDAttrName  string

	AdditionalSelectors []string
}

// Selectors used by the transformer
func (transformer Transformer) Selectors() []string {
	return append(transformer.AdditionalSelectors, transformer.MailAttrName, transformer.CNAttrName, transformer.UIDAttrName)
}

// Transform performs the actual tranformation
func (transformer Transformer) Transform(entry *ldap.Entry) interface{} {
	user := newUserData()

	for _, attr := range entry.Attributes {
		if attr.Name == transformer.MailAttrName {
			user.Email = attr.Values[0]
		}

		if attr.Name == transformer.CNAttrName {
			user.Name = attr.Values[0]
		}

		if attr.Name == transformer.UIDAttrName {
			// create a int64 hash sum to generate a user id from uid
			// this is technically important in order to be compatible to mattermost
			uid := attr.Values[0]
			h := sha256.New()
			h.Write([]byte(uid))
			user.ID = int64(binary.BigEndian.Uint64(h.Sum(nil)))

			// generate user name from uid
			user.Username = transformer.UsernamePrefix + attr.Values[0]
		}
	}

	return user
}
