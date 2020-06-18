package oauthenticator

// AuthenticatorBackend interface to provide to a new OAuth server
type AuthenticatorBackend interface {
	// Authenticate authenticates the user and returns the unique user identifier
	Authenticate(username, password string) (string, error)

	// GetUserById fetches the user object from the backend without
	GetUserByID(id string) (interface{}, error)
}
