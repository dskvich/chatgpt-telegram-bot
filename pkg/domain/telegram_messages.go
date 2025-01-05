package domain

type TextMessage struct {
	ChatID int64
	Text   string
}

type ImageMessage struct {
	ChatID int64
	Bytes  []byte
}

type TTLMessage struct {
	ChatID int64
}

type CallbackMessage struct {
	CallbackQueryID string
}
