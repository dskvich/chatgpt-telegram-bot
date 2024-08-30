package domain

type TextMessage struct {
	ChatID           int64
	ReplyToMessageID int
	Text             string
}

type ImageMessage struct {
	ChatID           int64
	ReplyToMessageID int
	Bytes            []byte
}

type TTLMessage struct {
	ChatID           int64
	ReplyToMessageID int
}

type CallbackMessage struct {
	CallbackQueryID string
}
