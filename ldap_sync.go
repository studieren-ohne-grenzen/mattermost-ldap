package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/go-ldap/ldap"
	"github.com/mattermost/mattermost-server/model"
	"github.com/studieren-ohne-grenzen/mattermost-ldap/ldapauthenticator"
)

type group struct {
	uid  string
	name string
}

// AuthenticatorWithSync composes ldapauthenticator.Authenticator with mattermost and syncs groups and users
type AuthenticatorWithSync struct {
	authenticator *ldapauthenticator.Authenticator

	userDn           string
	groupMemberQuery string
	groupBaseDn      string

	mattermostClient *model.Client4

	mattermostURL      string
	mattermostUsername string
	mattermostPassword string

	transformer Transformer
}

// NewAuthenticatorWithSync creates a new authenticator with Mattermost syncing functionality
func NewAuthenticatorWithSync(bindDn, bindPassword, queryDn, groupMemberQuery, groupBaseDn string, transformer Transformer) AuthenticatorWithSync {
	var syncAuther AuthenticatorWithSync

	auther := ldapauthenticator.NewAuthenticator(bindDn, bindPassword, queryDn, transformer)
	syncAuther.authenticator = &auther
	syncAuther.userDn = queryDn
	syncAuther.groupMemberQuery = groupMemberQuery
	syncAuther.groupBaseDn = groupBaseDn

	syncAuther.transformer = transformer

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
	auth.mattermostClient = model.NewAPIv4Client(url)
	_, resp := auth.mattermostClient.Login(username, password)

	auth.mattermostURL = url
	auth.mattermostUsername = username
	auth.mattermostPassword = password

	if resp.Error != nil {
		log.Printf("Got error during login: %+v\n", resp.Error)
		return resp.Error
	}

	return nil
}

// Mattermost returns the current valid mattermost connection
func (auth *AuthenticatorWithSync) Mattermost() *model.Client4 {
	if _, resp := auth.mattermostClient.GetPing(); resp.Error != nil {
		// Ping was not successful, retry to connect
		if err := auth.ReconnectMattermost(10); err != nil {
			// Reconnect was not successful, there is something more troublesome going on...
			panic(err)
		}
	}

	return auth.mattermostClient
}

// ReconnectMattermost tries to reconnect to mattermost within maxReconnectCount times
func (auth *AuthenticatorWithSync) ReconnectMattermost(maxReconnectCount uint) error {
	if err := auth.ConnectMattermost(auth.mattermostURL, auth.mattermostUsername, auth.mattermostPassword); err != nil {
		log.Printf("Could not connect to mattermost: %+v\n", err)
		// login was not successful
		if maxReconnectCount >= 0 {
			// but we have some more tries to go
			log.Println("Retrying to connect to mattermost")
			return auth.ReconnectMattermost(maxReconnectCount - 1)
		}

		return errors.New("Could not reconnect to mattermost")
	}

	return nil // sucessful connection
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

	mattermostUser, mmErr := auth.Mattermost().GetUserByEmail(user.(userData).Email, "")
	if mmErr.Error != nil {
		log.Printf("Could not retrieve user from mattermost: %+v\n", mmErr.Error)
		return
	}

	auth.checkMattermostUser(user.(userData).ID, mattermostUser.Username, user.(userData).Name, mattermostUser.Email)

	groups := auth.fetchGroupsForUser(uid)
	mattermostGroups, mmErr := auth.Mattermost().GetTeamsForUser(mattermostUser.Id, "")
	if mmErr.Error != nil {
		log.Printf("Could not retrieve groups for user %s from mattermost: %+v\n", mattermostUser.Username, mmErr.Error)
		return
	}

	var mattermostTeamNames []string
	for _, team := range mattermostGroups {
		mattermostTeamNames = append(mattermostTeamNames, team.Name)
	}

	log.Printf("Comparing [ldap: %+v] vs. [mattermost: %+v]", groups, mattermostTeamNames)

	// check which groups we need to add
	for _, group := range groups {
		found := false
		for index, mmGroup := range mattermostGroups {
			if auth.normalizeGroupName(group.uid) == mmGroup.Name {
				// user already in group, delete entry from mattermost array. No need to consider it further
				found = true
				mattermostGroups = append(mattermostGroups[:index], mattermostGroups[index+1:]...)
				break
			}
		}

		if !found {
			auth.checkGroupForMattermostUser(group, mattermostUser.Email)
		}
	}

	for _, group := range mattermostGroups {
		// all these remaining groups could not be matched against a ldap group. remove the user!
		if _, mmErr := auth.Mattermost().RemoveTeamMember(group.Id, mattermostUser.Id); mmErr.Error != nil {
			log.Printf("Could not remove user %s from team %s:%+v\n", mattermostUser.Username, group.Name, mmErr.Error)
		}
	}

}
