package command

import (
	"context"
	"fmt"
	"strings"

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
	GenerateImage(prompt string) ([]byte, error)
}

type VoicePromptSaver interface {
	Save(ctx context.Context, p *domain.Prompt) error
}

type voice struct {
	downloader     FileDownloader
	converter      AudioConverter
	transcriber    SpeechTranscriber
	textGenerator  GptTextResponseGenerator
	imageGenerator GptImageResponseGenerator
	saver          VoicePromptSaver
	outCh          chan<- domain.Message
}

func NewVoice(
	downloader FileDownloader,
	converter AudioConverter,
	transcriber SpeechTranscriber,
	textGenerator GptTextResponseGenerator,
	imageGenerator GptImageResponseGenerator,
	saver VoicePromptSaver,
	outCh chan<- domain.Message,
) *voice {
	return &voice{
		downloader:     downloader,
		converter:      converter,
		transcriber:    transcriber,
		textGenerator:  textGenerator,
		imageGenerator: imageGenerator,
		saver:          saver,
		outCh:          outCh,
	}
}

func (v *voice) CanExecute(update *tgbotapi.Update) bool {
	return update.Message != nil && update.Message.Voice != nil
}

func (v *voice) Execute(update *tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	messageID := update.Message.MessageID

	filePath, err := v.downloader.DownloadFile(update.Message.Voice.FileID)
	if err != nil {
		v.outCh <- &domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Content:          fmt.Sprintf("Failed to download audio file: %v", err),
		}
		return
	}

	mp3FilePath, err := v.converter.ConvertToMP3(filePath)
	if err != nil {
		v.outCh <- &domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Content:          fmt.Sprintf("Failed to convert audio file: %v", err),
		}
		return
	}

	prompt, err := v.transcriber.SpeechToText(mp3FilePath)
	if err != nil {
		v.outCh <- &domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Content:          fmt.Sprintf("Failed to transcribe audio file: %v", err),
		}
		return
	}

	v.outCh <- &domain.TextMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Content:          fmt.Sprintf("🎤 %s", prompt),
	}

	if err := v.saver.Save(context.Background(), &domain.Prompt{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      prompt,
		FromUser:  fmt.Sprintf("%s %s", update.Message.From.FirstName, update.Message.From.LastName),
	}); err != nil {
		v.outCh <- &domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Content:          fmt.Sprintf("Failed to save prompt: %v", err),
		}
	}

	if strings.Contains(strings.ToLower(prompt), "рисуй") {
		processedPrompt := removeWordContaining(prompt, "рисуй")

		imgBytes, err := v.imageGenerator.GenerateImage(processedPrompt)
		if err != nil {
			v.outCh <- &domain.TextMessage{
				ChatID:           chatID,
				ReplyToMessageID: messageID,
				Content:          fmt.Sprintf("Failed to generate image using Dall-E: %v", err),
			}
			return
		}

		v.outCh <- &domain.ImageMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Content:          imgBytes,
		}
		return
	}

	response, err := v.textGenerator.CreateChatCompletion(update.Message.Chat.ID, prompt, "")
	if err != nil {
		response = fmt.Sprintf("Failed to get chat completion: %v", err)
	}

	v.outCh <- &domain.TextMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Content:          response,
	}
}

func removeWordContaining(text string, target string) string {
	words := strings.Fields(text)
	var filtered []string

	for _, word := range words {
		if !strings.Contains(strings.ToLower(word), strings.ToLower(target)) {
			filtered = append(filtered, word)
		}
	}

	return strings.Join(filtered, " ")
}
