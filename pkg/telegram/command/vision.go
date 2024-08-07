package command

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ImageGetter interface {
	GetFile(fileID string) (base64image string, err error)
}

type ImageRecognizer interface {
	CreateChatCompletion(chatID int64, text, base64image string) (string, error)
}

type vision struct {
	getter     ImageGetter
	recognizer ImageRecognizer
	outCh      chan<- domain.Message
}

func NewVision(
	getter ImageGetter,
	imageRecognizer ImageRecognizer,
	outCh chan<- domain.Message,
) *vision {
	return &vision{
		getter:     getter,
		recognizer: imageRecognizer,
		outCh:      outCh,
	}
}

func (_ *vision) CanExecute(update *tgbotapi.Update) bool {
	if update.Message == nil {
		return false
	}

	return len(update.Message.Photo) > 0
}

func (v *vision) Execute(update *tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	messageID := update.Message.MessageID
	caption := update.Message.Caption
	photo := (update.Message.Photo)[len(update.Message.Photo)-1]

	base64image, err := v.getter.GetFile(photo.FileID)
	if err != nil {
		v.outCh <- &domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Content:          fmt.Sprintf("Failed to get image: %v", err),
		}
		return
	}

	response, err := v.recognizer.CreateChatCompletion(chatID, caption, base64image)
	if err != nil {
		response = fmt.Sprintf("Failed to recognize image: %v", err)
	}

	v.outCh <- &domain.TextMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Content:          response,
	}
}
