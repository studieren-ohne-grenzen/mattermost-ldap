package main

import (
	"github.com/studieren-ohne-grenzen/mattermost-ldap-sync"
	ldap "github.com/zonradkuse/go-ldap-authenticator"

	"crypto/sha256"
	"encoding/binary"
)

type Transformer struct{}

func (this Transformer) Transform(entry *ldap.Entry) interface{} {
	user := ldapsync.NewUserData()

	for _, attr := range entry.Attributes {
		if attr.Name == "mail" {
			user.Email = attr.Values[0]
		}

		if attr.Name == "cn" {
			user.Name = attr.Values[0]
		}

		if attr.Name == "uid" {
			uid := attr.Values[0]
			h := sha256.New()
			h.Write([]byte(uid))
			user.Id = int64(binary.BigEndian.Uint64(h.Sum(nil)))

			user.Username = "sog_" + uid
		}
	}

	return user
}
