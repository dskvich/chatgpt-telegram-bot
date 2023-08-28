package handler

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type FileDownloader interface {
	DownloadFile(fileID string) (filePath string, err error)
}

type AudioConverter interface {
	ConvertToMP3(inputPath string) (outputPath string, err error)
}

type SpeechTranscriber interface {
	SpeechToText(filePath string) (text string, err error)
}

type GptResponseGenerator interface {
	GenerateChatResponse(chatID int64, prompt string) (string, error)
}

type Voice struct {
	downloader  FileDownloader
	converter   AudioConverter
	transcriber SpeechTranscriber
	generator   GptResponseGenerator
}

func (v *Voice) CanHandle(update *tgbotapi.Update) bool {
	return update.Message != nil && update.Message.Voice != nil
}

func (v *Voice) Handle(update *tgbotapi.Update) domain.Message {
	filePath, err := v.downloader.DownloadFile(update.Message.Voice.FileID)
	if err != nil {
		return &domain.TextMessage{
			ChatID:           update.Message.Chat.ID,
			ReplyToMessageID: update.Message.MessageID,
			Content:          fmt.Sprintf("Failed to download audio file: %v", err),
		}
	}

	mp3FilePath, err := v.converter.ConvertToMP3(filePath)
	if err != nil {
		return &domain.TextMessage{
			ChatID:           update.Message.Chat.ID,
			ReplyToMessageID: update.Message.MessageID,
			Content:          fmt.Sprintf("Failed to convert audio file: %v", err),
		}
	}

	text, err := v.transcriber.SpeechToText(mp3FilePath)
	if err != nil {
		return &domain.TextMessage{
			ChatID:           update.Message.Chat.ID,
			ReplyToMessageID: update.Message.MessageID,
			Content:          fmt.Sprintf("Failed to transcribe audio file: %v", err),
		}
	}

	response, err := v.generator.GenerateChatResponse(update.Message.Chat.ID, text)
	if err != nil {
		response = fmt.Sprintf("Failed to get response from ChatGPT: %v", err)
	}
	return &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          response,
	}
}
