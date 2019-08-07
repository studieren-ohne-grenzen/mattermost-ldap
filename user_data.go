package main

type userData struct {
	Email    string `json:"email"`
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	State    string `json:"state"`
}

func newUserData() userData {
	var data userData

	data.State = "active"

	return data
}
