package domain

type Prompt struct {
	MessageID int
	ChatID    int64
	Text      string
	FromUser  string
}
