package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v9"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/converter"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/database"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/openai"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/repository"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/telegram/handlers"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/telegram/matchers"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/telegram/middleware"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/workers"
	"github.com/go-telegram/bot"
)

type Config struct {
	OpenAIToken                           string        `env:"OPEN_AI_TOKEN,required"`
	TelegramBotToken                      string        `env:"TELEGRAM_BOT_TOKEN,required"`
	TelegramAuthorizedUserIDs             []int64       `env:"TELEGRAM_AUTHORIZED_USER_IDS" envSeparator:" "`
	TelegramUpdateListenerPoolSize        int           `env:"TELEGRAM_UPDATE_LISTENER_POOL_SIZE" envDefault:"10"`
	TelegramUpdateListenerPollingInterval time.Duration `env:"TELEGRAM_UPDATE_LISTENER_POLL_INTERVAL" envDefault:"100ms"`
	PgURL                                 string        `env:"DATABASE_URL"`
	PgHost                                string        `env:"DB_HOST" envDefault:"localhost:65432"`
}

func main() {
	slog.SetDefault(slog.New(logger.NewHandler(os.Stderr, logger.DefaultOptions)))

	if err := runMain(); err != nil {
		slog.Error("shutting down due to error", logger.Err(err))
		os.Exit(1)
	}
	slog.Info("shutdown complete")
}

func runMain() error {
	workerGroup, err := setupWorkers()
	if err != nil {
		return err
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGHUP)
		select {
		case s := <-sigCh:
			slog.Info("shutting down due to signal", "signal", s.String())
			cancelFn()
		case <-ctx.Done():
		}
	}()

	return workerGroup.Start(ctx)
}

func setupWorkers() (workers.Group, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parsing env config: %w", err)
	}

	var worker workers.Worker
	var workerGroup workers.Group

	db, err := database.NewPostgres(cfg.PgURL, cfg.PgHost)
	if err != nil {
		return nil, fmt.Errorf("creating db: %w", err)
	}

	openAIClient, err := openai.NewClient(cfg.OpenAIToken)
	if err != nil {
		return nil, fmt.Errorf("creating open ai client: %w", err)
	}

	chatRepository := repository.NewChatRepository()
	stateRepository := repository.NewStateRepository()
	promptRepository := repository.NewPromptsRepository(db)
	settingsRepository := repository.NewSettingsRepository(db)

	// Price per 1M tokens (Input/Output)
	// https://platform.openai.com/docs/pricing
	supportedTextModels := []string{
		"gpt-4o-mini",   // $0.15/$0.60
		"gpt-3.5-turbo", // $0.50/$1.50
		"o3-mini",       // $1.10/$4.40
		// "gpt-4o",        // $2.50/$10.00
		// "gpt-4-turbo",   // $10.00/$30.00
	}

	supportedTTLOptions := []time.Duration{
		15 * time.Minute,
		time.Hour,
		8 * time.Hour,
		24 * time.Hour,
		7 * 24 * time.Hour,
	}

	opts := []bot.Option{
		bot.WithMiddlewares(
			middleware.RequestID,
			middleware.Auth(cfg.TelegramAuthorizedUserIDs),
			middleware.Typing,
			middleware.VoiceToText(&converter.VoiceToMP3{}, openAIClient),
		),

		bot.WithDefaultHandler(handlers.GenerateContent(settingsRepository, chatRepository, promptRepository, openAIClient)),
		bot.WithMessageTextHandler("/start", bot.MatchTypePrefix, handlers.Start()),
		bot.WithMessageTextHandler("/new", bot.MatchTypePrefix, handlers.ClearChat(chatRepository)),
		bot.WithMessageTextHandler("/text_models", bot.MatchTypePrefix, handlers.ShowTextModels(supportedTextModels)),
		bot.WithMessageTextHandler("/image_models", bot.MatchTypePrefix, handlers.ShowImageModels()),
		bot.WithMessageTextHandler("/system_prompt", bot.MatchTypePrefix, handlers.ShowSystemPrompt(settingsRepository)),
		bot.WithMessageTextHandler("/ttl", bot.MatchTypePrefix, handlers.ShowTTL(supportedTTLOptions)),

		bot.WithCallbackQueryDataHandler(domain.SetTTLCallbackPrefix, bot.MatchTypePrefix, handlers.SetTTL(settingsRepository, supportedTTLOptions)),
		bot.WithCallbackQueryDataHandler(domain.SetTextModelCallbackPrefix, bot.MatchTypePrefix, handlers.SetTextModel(settingsRepository, chatRepository, supportedTextModels)),
		bot.WithCallbackQueryDataHandler(domain.SetSystemPromptCallbackPrefix, bot.MatchTypePrefix, handlers.RequestSystemPrompt(stateRepository)),
		bot.WithCallbackQueryDataHandler(domain.GenImageCallbackPrefix, bot.MatchTypePrefix, handlers.RegenerateImage(promptRepository, openAIClient)),
	}

	b, err := bot.New(cfg.TelegramBotToken, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating telegram bot: %w", err)
	}

	b.RegisterHandlerMatchFunc(matchers.IsEditingSystemPrompt(stateRepository), handlers.SetSystemPrompt(settingsRepository, chatRepository, stateRepository))

	if worker, err = workers.NewTelegramBot(b); err == nil {
		workerGroup = append(workerGroup, worker)
	} else {
		return nil, err
	}

	return workerGroup, nil
}
