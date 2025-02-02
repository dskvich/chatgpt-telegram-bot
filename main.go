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
	"github.com/dskvich/chatgpt-telegram-bot/pkg/services"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/telegram/handler"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/workers"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/auth"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/database"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/openai"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/repository"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/telegram"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/tools"
)

const DefaultChatTTL = 15 * time.Minute

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

	imageGenerationStyles := map[string]string{
		"vivid":   "Яркий",
		"natural": "Естественный",
	}

	telegramClient, err := telegram.NewClient(cfg.TelegramBotToken, imageGenerationStyles)
	if err != nil {
		return nil, fmt.Errorf("creating telegram client: %w", err)
	}
	authenticator := auth.NewAuthenticator(cfg.TelegramAuthorizedUserIDs)

	db, err := database.NewPostgres(cfg.PgURL, cfg.PgHost)
	if err != nil {
		return nil, fmt.Errorf("creating db: %w", err)
	}

	openAIClient, err := openai.NewClient(cfg.OpenAIToken) //, chatRepository, settingsRepository, chatStyleRepository, tools)
	if err != nil {
		return nil, fmt.Errorf("creating open ai client: %w", err)
	}

	chatRepository := repository.NewChatRepository(DefaultChatTTL)
	promptRepository := repository.NewImagePromptRepository(db)
	settingsRepository := repository.NewSettingsRepository(db)
	chatStyleRepository := repository.NewChatStyleRepository(db)

	// Price per 1M tokens (Input/Output)
	supportedModels := []string{
		"gpt-4o-mini",   // $0.15/$0.60
		"gpt-3.5-turbo", // $0.50/$1.50
		"o3-mini",       // $1.10/$4.40
		//"gpt-4o",        // $2.50/$10.00
		//"gpt-4-turbo",   // $10.00/$30.00
	}

	toolFunctions := []services.ToolFunction{
		tools.NewGetChatSettings(settingsRepository),
		tools.NewSetModel(settingsRepository, supportedModels),
		tools.NewUpdateActiveChatStyle(chatStyleRepository),
		tools.NewCreateChatStyleFromActive(chatStyleRepository),
		tools.NewActivateChatStyle(chatStyleRepository),
		tools.NewDeleteChatStyle(chatStyleRepository),
	}

	toolService, err := services.NewToolService(toolFunctions)
	if err != nil {
		return nil, fmt.Errorf("creating tool service: %w", err)
	}

	const (
		ttlShort  = 15 * time.Minute
		ttlMedium = time.Hour
		ttlLong   = 8 * time.Hour
		ttlNone   = 0
	)

	chatTTLOptions := map[string]time.Duration{
		"ttl_15m":      ttlShort,
		"ttl_1h":       ttlMedium,
		"ttl_8h":       ttlLong,
		"ttl_disabled": ttlNone,
	}

	imageKeywords := []string{"рисуй", "draw"}
	intentDetector := services.NewIntentDetector(imageKeywords)

	imageService := services.NewImageService(openAIClient, promptRepository, settingsRepository)
	chatService := services.NewChatService(
		openAIClient,
		chatRepository,
		settingsRepository,
		chatStyleRepository,
		toolService,
		intentDetector,
		chatTTLOptions,
		&converter.VoiceToMP3{},
		imageService,
	)

	handlers := []workers.Handler{
		// non ai commands
		handler.NewShowWelcomeMessage(telegramClient),
		handler.NewClearChatMessage(chatService, telegramClient),
		handler.NewSetTTLMessage(telegramClient),
		handler.NewSetTTLCallback(chatService, telegramClient, chatTTLOptions),
		handler.NewShowChatSettingsMessage(chatService, telegramClient),
		handler.NewSetImageStyleMessage(telegramClient),
		handler.NewSetImageStyleCallback(imageService, telegramClient, imageGenerationStyles),
		handler.NewShowChatStylesMessage(chatService, telegramClient),

		// ai commands
		handler.NewGenerateImageCallback(imageService, telegramClient),
		handler.NewGenerateResponseFromVoiceMessage(chatService, telegramClient),
		handler.NewGenerateResponseFromImageMessage(chatService, telegramClient),
		handler.NewGenerateResponseMessage(chatService, telegramClient),
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
