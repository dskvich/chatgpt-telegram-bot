package command

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
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

type GptTextResponseGenerator interface {
	CreateChatCompletion(chatID int64, text, base64image string) (string, error)
}

type GptImageResponseGenerator interface {
	GenerateImage(chatID int64, prompt string) ([]byte, error)
}

type VoicePromptSaver interface {
	SavePrompt(ctx context.Context, p *domain.Prompt) error
}

type processVoice struct {
	downloader     FileDownloader
	converter      AudioConverter
	transcriber    SpeechTranscriber
	textGenerator  GptTextResponseGenerator
	imageGenerator GptImageResponseGenerator
	saver          VoicePromptSaver
	client         TelegramClient
}

func NewProcessVoice(
	downloader FileDownloader,
	converter AudioConverter,
	transcriber SpeechTranscriber,
	textGenerator GptTextResponseGenerator,
	imageGenerator GptImageResponseGenerator,
	saver VoicePromptSaver,
	client TelegramClient,
) *processVoice {
	return &processVoice{
		downloader:     downloader,
		converter:      converter,
		transcriber:    transcriber,
		textGenerator:  textGenerator,
		imageGenerator: imageGenerator,
		saver:          saver,
		client:         client,
	}
}

func (p *processVoice) IsCommand(u *tgbotapi.Update) bool {
	return u.Message != nil && u.Message.Voice != nil
}

func (p *processVoice) HandleCommand(u *tgbotapi.Update) {
	chatID := u.Message.Chat.ID
	messageID := u.Message.MessageID

	filePath, err := p.downloader.DownloadFile(u.Message.Voice.FileID)
	if err != nil {
		p.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             fmt.Sprintf("Failed to download audio file: %p", err),
		})
		return
	}

	mp3FilePath, err := p.converter.ConvertToMP3(filePath)
	if err != nil {
		p.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             fmt.Sprintf("Failed to convert audio file: %p", err),
		})
		return
	}

	prompt, err := p.transcriber.SpeechToText(mp3FilePath)
	if err != nil {
		p.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             fmt.Sprintf("Failed to transcribe audio file: %p", err),
		})
		return
	}

	p.client.SendTextMessage(domain.TextMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Text:             fmt.Sprintf("🎤 %s", prompt),
	})

	if err := p.saver.SavePrompt(context.Background(), &domain.Prompt{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      prompt,
		FromUser:  fmt.Sprintf("%s %s", u.Message.From.FirstName, u.Message.From.LastName),
	}); err != nil {
		p.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             fmt.Sprintf("Failed to save prompt: %p", err),
		})
	}

	commandText := domain.CommandText(prompt)
	if commandText.ContainsAny(domain.DrawKeywords) {
		prompt = commandText.ExtractAfterKeywords(domain.DrawKeywords)

		imgBytes, err := p.imageGenerator.GenerateImage(chatID, prompt)
		if err != nil {
			p.client.SendTextMessage(domain.TextMessage{
				ChatID:           chatID,
				ReplyToMessageID: messageID,
				Text:             fmt.Sprintf("Failed to generate image using Dall-E: %p", err),
			})
			return
		}

		p.client.SendImageMessage(domain.ImageMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Bytes:            imgBytes,
		})
		return
	}

	response, err := p.textGenerator.CreateChatCompletion(u.Message.Chat.ID, prompt, "")
	if err != nil {
		response = fmt.Sprintf("Failed to get chat completion: %p", err)
	}

	p.client.SendTextMessage(domain.TextMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Text:             response,
	})
}
