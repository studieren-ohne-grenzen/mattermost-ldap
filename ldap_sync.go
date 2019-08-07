package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/go-ldap/ldap"
	"github.com/mattermost/mattermost-server/model"
	lauth "github.com/zonradkuse/go-ldap-authenticator"
)

type group struct {
	uid  string
	name string
}

// AuthenticatorWithSync composes lauth.Authenticator with mattermost and syncs groups and users
type AuthenticatorWithSync struct {
	authenticator *lauth.Authenticator

	userDn           string
	groupMemberQuery string
	groupBaseDn      string

	mattermost *model.Client4
}

// NewAuthenticatorWithSync creates a new authenticator with Mattermost syncing functionality
func NewAuthenticatorWithSync(bindDn, bindPassword, queryDn, groupMemberQuery, groupBaseDn string, transformer Transformer) AuthenticatorWithSync {
	var syncAuther AuthenticatorWithSync

	auther := lauth.NewAuthenticator(bindDn, bindPassword, queryDn, transformer)
	syncAuther.authenticator = &auther
	syncAuther.userDn = queryDn
	syncAuther.groupMemberQuery = groupMemberQuery
	syncAuther.groupBaseDn = groupBaseDn

	return syncAuther
}

// Connect to bindUrl LDAP server
func (auth *AuthenticatorWithSync) Connect(bindURL string) error {
	return auth.authenticator.Connect(bindURL)
}

// Close the LDAP connection
func (auth *AuthenticatorWithSync) Close() {
	auth.authenticator.Close()
}

// ConnectMattermost connects to the given mattermost instance
func (auth *AuthenticatorWithSync) ConnectMattermost(url, username, password string) error {
	auth.mattermost = model.NewAPIv4Client(url)
	_, resp := auth.mattermost.Login(username, password)

	if resp.Error != nil {
		log.Printf("Got error during login: %+v\n", resp.Error)
		return errors.New("Login failed")
	}

	return nil
}

// GetUserByID from LDAP
func (auth AuthenticatorWithSync) GetUserByID(id string) (interface{}, error) {
	return auth.authenticator.GetUserByID(id)
}

// Authenticate user with password at LDAP
func (auth AuthenticatorWithSync) Authenticate(username, password string) (string, error) {
	uid, err := auth.authenticator.Authenticate(username, password)
	if err != nil {
		return "", err
	}

	auth.syncMattermostForUser(uid)
	return uid, nil
}

func (auth *AuthenticatorWithSync) fetchGroupsForUser(uid string) []group {
	conn := auth.authenticator.Connection()

	filter := fmt.Sprintf(auth.groupMemberQuery, uid, auth.userDn)
	searchRequest := ldap.NewSearchRequest(
		auth.groupBaseDn, // The base dn to search
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter, // The filter to apply
		// TODO make more generic
		[]string{"dn", "cn", "ou"}, // A list attributes to retrieve
		nil,
	)

	res, err := conn.Search(searchRequest)
	if err != nil {
		log.Printf("ERROR: %+v\n", err)
		return []group{}
	}

	entries := res.Entries
	var groups []group
	for _, entry := range entries {
		// TODO make more generic
		group := group{uid: entry.GetAttributeValue("ou"), name: entry.GetAttributeValue("cn")}
		groups = append(groups, group)
	}

	return groups
}

func (auth *AuthenticatorWithSync) syncMattermostForUser(uid string) {
	user, err := auth.GetUserByID(uid)
	if err != nil {
		log.Printf("ERROR: %+v\n", err)
		return
	}

	if strings.Index(user.(userData).Username, uid) < 0 {
		log.Printf("ERROR: Invalid state. Got uid %s but userData %+v\n", uid, user)
		return
	}

	auth.checkMattermostUser(user.(userData).ID, user.(userData).Username, user.(userData).Name, user.(userData).Email)

	groups := auth.fetchGroupsForUser(uid)
	for _, group := range groups {
		auth.checkGroupForMattermostUser(group, user.(userData).Email)
	}
}
