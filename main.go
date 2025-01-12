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
	"github.com/dskvich/chatgpt-telegram-bot/pkg/telegram/handler"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/workers"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/auth"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/chatgpt"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/converter"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/database"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/openai"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/repository"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/telegram"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/tools"
)

type Config struct {
	OpenAIToken                           string        `env:"OPEN_AI_TOKEN,required"`
	TelegramBotToken                      string        `env:"TELEGRAM_BOT_TOKEN,required"`
	TelegramAuthorizedUserIDs             []int64       `env:"TELEGRAM_AUTHORIZED_USER_IDS" envSeparator:" "`
	TelegramUpdateListenerPoolSize        int           `env:"TELEGRAM_UPDATE_LISTENER_POOL_SIZE" envDefault:"10"`
	TelegramUpdateListenerPollingInterval time.Duration `env:"TELEGRAM_UPDATE_LISTENER_POLLING_INTERVAL" envDefault:"100ms"`
	PgURL                                 string        `env:"DATABASE_URL"`
	PgHost                                string        `env:"DB_HOST" envDefault:"localhost:65432"`
}

func main() {
	slog.SetDefault(logger.New(slog.LevelDebug))

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
		return nil, fmt.Errorf("parsing env config: %v", err)
	}

	var worker workers.Worker
	var workerGroup workers.Group

	telegramClient, err := telegram.NewClient(cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("creating telegram client: %v", err)
	}
	authenticator := auth.NewAuthenticator(cfg.TelegramAuthorizedUserIDs)

	db, err := database.NewPostgres(cfg.PgURL, cfg.PgHost)
	if err != nil {
		return nil, fmt.Errorf("creating db: %v", err)
	}

	chatRepository := repository.NewChatRepository(15 * time.Minute)
	promptRepository := repository.NewPromptRepository(db)
	settingsRepository := repository.NewSettingsRepository(db)
	chatStyleRepository := repository.NewChatStyleRepository(db)

	// Initialize tools
	tools := []openai.ToolFunction{
		tools.NewGetChatSettings(settingsRepository),
		tools.NewSetModel(settingsRepository),
		tools.NewUpdateActiveChatStyle(chatStyleRepository),
		tools.NewCreateChatStyleFromActive(chatStyleRepository),
		tools.NewActivateChatStyle(chatStyleRepository),
	}

	//TODO rename textGptClient to openAiClient
	openAIClient, err := openai.NewClient(cfg.OpenAIToken, chatRepository, settingsRepository, chatStyleRepository, tools)
	if err != nil {
		return nil, fmt.Errorf("creating open ai client: %v", err)
	}
	imageGptClient := chatgpt.NewImageClient(cfg.OpenAIToken, settingsRepository)
	audioGptClient := chatgpt.NewAudioClient(cfg.OpenAIToken)

	oggToMp3Converter := converter.OggTomp3{}
	speechToTextConverter := converter.NewSpeechToText(audioGptClient)

	handlers := []any{
		// non ai commands
		handler.NewShowInfo(telegramClient),
		handler.NewClearChat(chatRepository, telegramClient),
		handler.NewSetTTL(telegramClient, chatRepository),
		handler.NewShowSettings(telegramClient, settingsRepository),
		handler.NewSetImageStyle(telegramClient, settingsRepository),
		handler.NewShowChatStyles(telegramClient, chatStyleRepository),

		// features
		handler.NewCompleteChat(openAIClient, telegramClient),
		handler.NewProcessVoice(telegramClient, &oggToMp3Converter, speechToTextConverter, openAIClient, imageGptClient, promptRepository, telegramClient),
		handler.NewDrawImage(imageGptClient, promptRepository, telegramClient),
		handler.NewCompleteImage(telegramClient, openAIClient, telegramClient),
	}

	if worker, err = workers.NewTelegramUpdateListener(
		telegramClient,
		authenticator,
		handlers,
		cfg.TelegramUpdateListenerPoolSize,
		cfg.TelegramUpdateListenerPollingInterval,
	); err == nil {
		workerGroup = append(workerGroup, worker)
	} else {
		return nil, err
	}

	return workerGroup, nil
}
