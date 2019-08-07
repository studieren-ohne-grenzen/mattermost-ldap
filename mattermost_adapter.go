package main

import (
	"github.com/mattermost/mattermost-server/model"

	"log"
	"strconv"
	"strings"
)

func (auth *AuthenticatorWithSync) getAllOAuthUsers() ([]*model.User, error) {
	curPage := 0
	terminate := false
	var result []*model.User

	for !terminate {
		users, resp := auth.Mattermost().GetUsers(curPage, 50, "")

		if resp.Error != nil {
			return nil, resp.Error
		}

		for _, user := range users {
			if user.AuthService == model.USER_AUTH_SERVICE_GITLAB {
				result = append(result, user)
			}
		}

		curPage++
		terminate = len(users) == 0
	}

	return result, nil
}

func (auth *AuthenticatorWithSync) syncOAuthUsersWithBackend([]*model.User) error {
	users, err := auth.getAllOAuthUsers()

	if err != nil {
		return err
	}

	for _, user := range users {
		// strip off username prefix to obtain the actual username to feed into backend
		username := strings.Replace(user.Username, auth.transformer.UsernamePrefix, "", 1)
		log.Printf("Syncing user %s with backend.\n", username)

		auth.syncMattermostForUser(username)
	}
	return nil
}

func (auth *AuthenticatorWithSync) syncAllOAuthUsers() {
	users, err := auth.getAllOAuthUsers()
	if err != nil {
		log.Printf("Error while syncing all OAuth users: %+v", err)
	}

	auth.syncOAuthUsersWithBackend(users)
}

func (auth *AuthenticatorWithSync) checkMattermostUser(id int64, username, name, mail string) {
	user, resp := auth.Mattermost().GetUserByEmail(mail, "")
	if resp.Error != nil && resp.StatusCode != 404 {
		log.Printf("ERROR: %+v", resp.Error)
		return
	}

	created := false
	userID := strconv.FormatInt(id, 10)
	if resp.StatusCode == 404 {
		log.Println("Creating new user.")
		// auth user does not exist
		var newUser model.User
		newUser.AuthService = model.USER_AUTH_SERVICE_GITLAB
		newUser.AuthData = &userID
		newUser.Email = mail
		newUser.FirstName = name
		newUser.Username = username
		newUser.EmailVerified = true

		user, resp = auth.Mattermost().CreateUser(&newUser)
		if resp.Error != nil {
			log.Printf("Could not create user with email %s, got error: %+v.", mail, resp.Error)
			return
		}

		created = true
	}

	// Update user if not just created
	if !created {
		user.Username = username
		user.Email = mail
		user.FirstName = strings.Split(name, " ")[0]
		if len(strings.Split(name, " ")) > 1 {
			user.LastName = strings.Split(name, " ")[1]
		}

		_, resp = auth.Mattermost().UpdateUser(user)
		if resp.Error != nil {
			log.Printf("Could not update existing user, got Error %+v", resp.Error)
			return
		}
	}

}

func (auth *AuthenticatorWithSync) checkGroupForMattermostUser(group group, mail string) {
	group.uid = strings.Replace(group.uid, "_", "-", -1)
	team, resp := auth.Mattermost().GetTeamByName(group.uid, "")
	if resp.Error != nil && resp.StatusCode != 404 {
		log.Printf("ERROR: Could not find team %+v, got error: %+v.", group, resp.Error)
	}

	if resp.StatusCode == 404 {
		newTeam := model.Team{}
		newTeam.Name = auth.normalizeGroupName(group.uid)
		newTeam.DisplayName = group.name
		newTeam.Type = "I"
		team, resp = auth.Mattermost().CreateTeam(&newTeam)
		if resp.Error != nil {
			log.Printf("ERROR: Could not create Team %+v, got error %+v", group, resp.Error)
			return
		}

		log.Printf("Created new Team %s.\n", team.DisplayName)
	}

	user, userResp := auth.Mattermost().GetUserByEmail(mail, "")
	if userResp.Error != nil {
		log.Printf("ERROR: Could not fetch user when adding to team %+v, got error: %+v", group, userResp.Error)
		return
	}

	_, err := auth.Mattermost().AddTeamMember(team.Id, user.Id)
	if err.Error != nil {
		log.Printf("ERROR: Could add user to team %+v, got error: %+v", group, err.Error)
		return
	}

	log.Printf("Added user %s to team %s \n", user.Email, team.DisplayName)
}

func (auth *AuthenticatorWithSync) normalizeGroupName(name string) string {
	return strings.Replace(name, "_", "-", -1)
}
