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
	"github.com/dskvich/chatgpt-telegram-bot/pkg/tools"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/auth"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/database"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/openai"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/repository"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/services"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/telegram"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/workers"
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

	telegramClient, err := telegram.NewClient(cfg.TelegramBotToken)
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

	chatRepository := repository.NewChatRepository()
	promptRepository := repository.NewPromptsRepository(db)
	settingsRepository := repository.NewSettingsRepository(db)

	// Price per 1M tokens (Input/Output)
	supportedTextModels := []string{
		"gpt-4o-mini",   // $0.15/$0.60
		"gpt-3.5-turbo", // $0.50/$1.50
		"o3-mini",       // $1.10/$4.40
		//"gpt-4o",        // $2.50/$10.00
		//"gpt-4-turbo",   // $10.00/$30.00
	}

	toolFunctions := []services.ToolFunction{
		tools.NewSetModel(settingsRepository, supportedTextModels),
	}

	toolService, err := services.NewToolService(toolFunctions)
	if err != nil {
		return nil, fmt.Errorf("creating tool service: %w", err)
	}

	responseCh := make(chan domain.Response)

	imageService := services.NewImageService(
		openAIClient,
		promptRepository,
		responseCh,
	)

	chatService := services.NewChatService(
		chatRepository,
		settingsRepository,
		toolService,
		supportedTextModels,
		responseCh,
	)

	textService := services.NewTextService(
		openAIClient,
		toolService,
		imageService,
		chatService,
		telegramClient,
		responseCh,
	)

	voiceService := services.NewVoiceService(
		&converter.VoiceToMP3{},
		openAIClient,
		telegramClient,
		imageService,
		textService,
		responseCh,
	)

	handler := telegram.NewHandler(
		imageService,
		textService,
		voiceService,
		chatService,
	)

	if worker, err = workers.
		NewTelegramUpdateListener(
			telegramClient,
			authenticator,
			handler,
			responseCh,
		); err == nil {
		workerGroup = append(workerGroup, worker)
	} else {
		return nil, err
	}

	return workerGroup, nil
}
