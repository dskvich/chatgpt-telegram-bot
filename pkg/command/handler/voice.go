package handler

import (
	"fmt"
	"strings"

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
	GenerateImage(prompt string) ([]byte, error)
}

type voice struct {
	downloader  FileDownloader
	converter   AudioConverter
	transcriber SpeechTranscriber
	generator   GptResponseGenerator
	outCh       chan<- domain.Message
}

func NewVoice(
	downloader FileDownloader,
	converter AudioConverter,
	transcriber SpeechTranscriber,
	generator GptResponseGenerator,
	outCh chan<- domain.Message,
) *voice {
	return &voice{
		downloader:  downloader,
		converter:   converter,
		transcriber: transcriber,
		generator:   generator,
		outCh:       outCh,
	}
}

func (v *voice) CanHandle(update *tgbotapi.Update) bool {
	return update.Message != nil && update.Message.Voice != nil
}

func (v *voice) Handle(update *tgbotapi.Update) {
	filePath, err := v.downloader.DownloadFile(update.Message.Voice.FileID)
	if err != nil {
		v.outCh <- &domain.TextMessage{
			ChatID:           update.Message.Chat.ID,
			ReplyToMessageID: update.Message.MessageID,
			Content:          fmt.Sprintf("Failed to download audio file: %v", err),
		}
		return
	}

	mp3FilePath, err := v.converter.ConvertToMP3(filePath)
	if err != nil {
		v.outCh <- &domain.TextMessage{
			ChatID:           update.Message.Chat.ID,
			ReplyToMessageID: update.Message.MessageID,
			Content:          fmt.Sprintf("Failed to convert audio file: %v", err),
		}
		return
	}

	text, err := v.transcriber.SpeechToText(mp3FilePath)
	if err != nil {
		v.outCh <- &domain.TextMessage{
			ChatID:           update.Message.Chat.ID,
			ReplyToMessageID: update.Message.MessageID,
			Content:          fmt.Sprintf("Failed to transcribe audio file: %v", err),
		}
		return
	}

	v.outCh <- &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          fmt.Sprintf("Вы сказали: %s", text),
	}

	if strings.Contains(strings.ToLower(text), "рисуй") {
		imgBytes, err := v.generator.GenerateImage(text)
		if err != nil {
			v.outCh <- &domain.TextMessage{
				ChatID:           update.Message.Chat.ID,
				ReplyToMessageID: update.Message.MessageID,
				Content:          fmt.Sprintf("Failed to generate image using Dall-E: %v", err),
			}
			return
		}

		v.outCh <- &domain.ImageMessage{
			ChatID:           update.Message.Chat.ID,
			ReplyToMessageID: update.Message.MessageID,
			Content:          imgBytes,
		}
		return
	}

	response, err := v.generator.GenerateChatResponse(update.Message.Chat.ID, text)
	if err != nil {
		response = fmt.Sprintf("Failed to get response from ChatGPT: %v", err)
	}

	v.outCh <- &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          response,
	}
}
