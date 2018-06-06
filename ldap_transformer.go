package main

import (
	"github.com/studieren-ohne-grenzen/mattermost-ldap-sync"
	ldap "github.com/zonradkuse/go-ldap-authenticator"

	"crypto/sha256"
	"encoding/binary"
)

type LDAPTransformer struct{}

func (this LDAPTransformer) Transform(entry *ldap.Entry) interface{} {
	user := ldapsync.NewUserData()

	for _, attr := range entry.Attributes {
		if attr.Name == "mail" {
			user.Email = attr.Values[0]
		}
		if attr.Name == "createTimestamp" {
			// WARNING, this is not unique!!
			// re := regexp.MustCompile("[0-9]+")
			// numbers := re.FindAllString(attr.Values[0], -1)
			// id, err := strconv.ParseInt(strings.Join(numbers, ""), 10, 64)
			// user.Id = id

			// if err != nil {
			//	panic(err)
			//}
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
