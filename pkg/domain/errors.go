package domain

import "errors"

const SessionInvalidatedMessage = "История очищена. Начните новый чат."

var ErrSessionInvalidated = errors.New("session invalidated")
