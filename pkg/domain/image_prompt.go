package domain

type ImagePrompt struct {
	ChatID   int64
	Prompt   string
	FromUser string
	ImageID  int64
}
