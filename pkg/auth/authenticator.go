package auth

import "log/slog"

type authenticator struct {
	authorizedUserIDs []int64
}

func NewAuthenticator(authorizedUserIDs []int64) *authenticator {
	slog.Info("telegram authorized user IDs", "user_ids", authorizedUserIDs)

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
