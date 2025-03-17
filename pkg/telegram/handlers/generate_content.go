package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/render"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/samber/lo"
)

type generateContentSettingsProvider interface {
	Get(ctx context.Context, chatID int64, topicID int) (*domain.Settings, error)
}

type generateContentChatProvider interface {
	Get(chatID int64, topicID int) (domain.Chat, time.Time, bool)
	Save(chat domain.Chat)
}

type generateContentAIService interface {
	GenerateImage(ctx context.Context, prompt string) ([]byte, error)
	CreateChatCompletion(ctx context.Context, chat *domain.Chat) (domain.ChatMessage, error)
}

type generateContentPromptSaver interface {
	Save(ctx context.Context, prompt string) (int64, error)
}

func GenerateContent(
	settingsProvider generateContentSettingsProvider,
	chatProvider generateContentChatProvider,
	promptSaver generateContentPromptSaver,
	aiService generateContentAIService,
) bot.HandlerFunc {
	const maxTelegramMessageLength = 4096
	const moreButtonText = "Еще"

	isExpired := func(lastUpdate time.Time, ttl time.Duration) bool {
		if ttl <= 0 {
			return false
		}
		return time.Since(lastUpdate) > ttl
	}

	findCutIndex := func(text string, maxLength int) int {
		if i := strings.LastIndex(text[:maxLength], "<pre>"); i > -1 {
			return i
		}
		if i := strings.LastIndex(text[:maxLength], "\n"); i > -1 {
			return i
		}
		return maxLength
	}

	downloadFileToBuffer := func(link string) ([]byte, error) {
		resp, err := http.Get(link)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, err
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		topicID := update.Message.MessageThreadID
		prompt := lo.CoalesceOrEmpty(update.Message.Text, update.Message.Caption)

		isImagePrompt := strings.Contains(strings.ToLower(prompt), "рисуй") ||
			strings.Contains(strings.ToLower(prompt), "draw")

		if isImagePrompt {
			promptID, err := promptSaver.Save(ctx, prompt)
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          chatID,
					MessageThreadID: topicID,
					Text:            fmt.Sprintf("❌ Не удалось сохранить промпт: %s", err),
				})
				return
			}

			imageData, err := aiService.GenerateImage(ctx, prompt)
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          chatID,
					MessageThreadID: topicID,
					Text:            fmt.Sprintf("❌ Не удалось сгенерировать изображение: %s", err),
				})
				return
			}

			kb := &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{{Text: moreButtonText, CallbackData: domain.GenImageCallbackPrefix + fmt.Sprint(promptID)}},
				},
			}

			b.SendPhoto(ctx, &bot.SendPhotoParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Photo: &models.InputFileUpload{
					Data: bytes.NewReader(imageData),
				},
				ReplyMarkup: kb,
			})
			return
		}

		// in case user send photo
		var imageBytes []byte
		if len(update.Message.Photo) > 0 {
			imageFile, err := b.GetFile(ctx, &bot.GetFileParams{
				FileID: update.Message.Photo[0].FileID,
			})
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          update.Message.Chat.ID,
					MessageThreadID: update.Message.MessageThreadID,
					Text:            fmt.Sprintf("❌ Не удалось получить метадату фото файла: %s", err),
				})
				return
			}

			imageFileURL, err := url.Parse(b.FileDownloadLink(imageFile))
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          update.Message.Chat.ID,
					MessageThreadID: update.Message.MessageThreadID,
					Text:            fmt.Sprintf("❌ Не удалось получить ссылку на фото файл: %s", err),
				})
				return
			}

			imageBytes, err = downloadFileToBuffer(imageFileURL.String())
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          update.Message.Chat.ID,
					MessageThreadID: update.Message.MessageThreadID,
					Text:            fmt.Sprintf("❌ Не удалось получить фото файл: %s", err),
				})
				return
			}
		}

		settings, err := settingsProvider.Get(ctx, chatID, topicID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          update.Message.Chat.ID,
				MessageThreadID: update.Message.MessageThreadID,
				Text:            fmt.Sprintf("❌ Не удалось получить настройки: %s", err),
			})
			return
		}

		settings, _ = lo.Coalesce(settings, &domain.Settings{})
		settings.TextModel, _ = lo.Coalesce(settings.TextModel, domain.Gpt4oMiniModel)
		settings.TTL, _ = lo.Coalesce(settings.TTL, 15*time.Minute)

		chat, lastUpdate, ok := chatProvider.Get(chatID, topicID)
		if !ok || isExpired(lastUpdate, settings.TTL) {
			slog.DebugContext(ctx, "Creating a new chat with parameters",
				"textModel", settings.TextModel,
				"ttl", settings.TTL,
				"systemPrompt", settings.SystemPrompt,
			)

			chat = domain.Chat{
				ID:        chatID,
				TopicID:   topicID,
				ModelName: settings.TextModel,
				Messages: lo.If(settings.SystemPrompt != "",
					[]domain.ChatMessage{{Role: "developer", Content: settings.SystemPrompt}}).
					Else(nil),
			}

			text := fmt.Sprintf(`<i>🛠️ Создан новый чат!
Текстовая модель GPT: %s
Период хранения данных: %s
Системная инструкция: %s
</i>`, settings.TextModel, settings.TTL, settings.SystemPrompt)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            text,
				ParseMode:       models.ParseModeHTML,
			})
		}

		// Add user message
		content := any(prompt)

		if len(imageBytes) > 0 {
			imageContent := domain.Content{
				Type: "image_url",
				ImageURL: &domain.ImageURL{
					URL: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(imageBytes),
				},
			}

			if prompt != "" {
				content = []domain.Content{
					{Type: "text", Text: prompt},
					imageContent,
				}
			} else {
				content = []domain.Content{imageContent}
			}
		}

		chat.Messages = append(chat.Messages, domain.ChatMessage{
			Role:    "user",
			Content: content,
		})

		slog.InfoContext(ctx, "Calling AI for chat completion", "model", chat.ModelName, "messagesCount", len(chat.Messages))

		chatResponse, err := aiService.CreateChatCompletion(ctx, &chat)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось сгенерировать ответ: %s", err),
			})
			return
		}

		slog.DebugContext(ctx, "Chat completion received", "content", chatResponse.Content)

		// Add assistant message
		chat.Messages = append(chat.Messages, chatResponse)

		if chatResponse.Content != nil {
			chatProvider.Save(chat)

			htmlText := render.ToHTML(fmt.Sprint(chatResponse.Content))
			for len(htmlText) > 0 {
				if utf8.RuneCountInString(htmlText) <= maxTelegramMessageLength {
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID:          chatID,
						MessageThreadID: topicID,
						Text:            htmlText,
						ParseMode:       models.ParseModeHTML,
					})
					return
				}

				cutIndex := findCutIndex(htmlText, maxTelegramMessageLength)
				_, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:          chatID,
					MessageThreadID: topicID,
					Text:            htmlText[:cutIndex],
					ParseMode:       models.ParseModeHTML,
				})
				if err != nil {
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID:          chatID,
						MessageThreadID: topicID,
						Text:            fmt.Sprintf("❌ Не удалось сгенерировать ответ: %s", err),
					})
				}
				htmlText = htmlText[cutIndex:]
				time.Sleep(time.Second) // tg rate limit
			}
			return
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            fmt.Sprintf("❌ Неожиданный ответ:  %+v", chatResponse),
		})
	}
}
