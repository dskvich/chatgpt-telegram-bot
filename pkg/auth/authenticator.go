package auth

type authenticator struct {
	authorizedUserIDs []int64
}

func NewAuthenticator(authorizedUserIDs []int64) *authenticator {
	return &authenticator{
		authorizedUserIDs: authorizedUserIDs,
	}
}

func (a *authenticator) IsAuthorized(userID int64) bool {
	for _, id := range a.authorizedUserIDs {
		if userID == id {
			return true
		}
	}
	return false
}
