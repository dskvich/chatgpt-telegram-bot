package handler

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type AudioConverter interface {
	ConvertToMP3(inputPath string) (outputPath string, err error)
}

type VoicePromptSaver interface {
	SavePrompt(ctx context.Context, p *domain.Prompt) error
}

type completeVoiceMessage struct {
	converter    AudioConverter
	openAiClient OpenAiClient
	saver        VoicePromptSaver
	client       TelegramClient
}

func NewCompleteVoiceMessage(
	converter AudioConverter,
	openAiClient OpenAiClient,
	saver VoicePromptSaver,
	client TelegramClient,
) *completeVoiceMessage {
	return &completeVoiceMessage{
		converter:    converter,
		openAiClient: openAiClient,
		saver:        saver,
		client:       client,
	}
}

func (_ *completeVoiceMessage) CanHandle(u *tgbotapi.Update) bool {
	return u.Message != nil && u.Message.Voice != nil
}

func (c *completeVoiceMessage) Handle(u *tgbotapi.Update) {
	chatID := u.Message.Chat.ID
	messageID := u.Message.MessageID

	filePath, err := c.client.DownloadFile(u.Message.Voice.FileID)
	if err != nil {
		c.client.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   fmt.Sprintf("Failed to download audio file: %c", err),
		})
		return
	}

	mp3FilePath, err := c.converter.ConvertToMP3(filePath)
	if err != nil {
		c.client.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   fmt.Sprintf("Failed to convert audio file: %c", err),
		})
		return
	}

	prompt, err := c.openAiClient.TranscribeAudio(mp3FilePath)
	if err != nil {
		c.client.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   fmt.Sprintf("Failed to transcribe audio file: %c", err),
		})
		return
	}

	c.client.SendTextMessage(domain.TextMessage{
		ChatID: chatID,
		Text:   fmt.Sprintf("🎤 %s", prompt),
	})

	if err := c.saver.SavePrompt(context.Background(), &domain.Prompt{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      prompt,
		FromUser:  fmt.Sprintf("%s %s", u.Message.From.FirstName, u.Message.From.LastName),
	}); err != nil {
		c.client.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   fmt.Sprintf("Failed to save prompt: %c", err),
		})
	}

	commandText := domain.CommandText(prompt)
	if commandText.ContainsAny(domain.DrawKeywords) {
		prompt = commandText.ExtractAfterKeywords(domain.DrawKeywords)

		imgBytes, err := c.openAiClient.GenerateImage(chatID, prompt)
		if err != nil {
			c.client.SendTextMessage(domain.TextMessage{
				ChatID: chatID,
				Text:   fmt.Sprintf("Failed to generate image using Dall-E: %c", err),
			})
			return
		}

		c.client.SendImageMessage(domain.ImageMessage{
			ChatID: chatID,
			Bytes:  imgBytes,
		})
		return
	}

	response, err := c.openAiClient.CreateChatCompletion(u.Message.Chat.ID, prompt, "")
	if err != nil {
		response = fmt.Sprintf("Failed to get chat completion: %c", err)
	}

	c.client.SendTextMessage(domain.TextMessage{
		ChatID: chatID,
		Text:   response,
	})
}
