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

	"github.com/dskvich/chatgpt-telegram-bot/pkg/auth"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/chatgpt"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/converter"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/database"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/openai"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/repository"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/service"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/telegram"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/telegram/command"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/tools"
)

type Config struct {
	OpenAIToken               string  `env:"OPEN_AI_TOKEN,required"`
	TelegramBotToken          string  `env:"TELEGRAM_BOT_TOKEN,required"`
	TelegramAuthorizedUserIDs []int64 `env:"TELEGRAM_AUTHORIZED_USER_IDS" envSeparator:" "`
	PgURL                     string  `env:"DATABASE_URL"`
	PgHost                    string  `env:"DB_HOST" envDefault:"localhost:65432"`
}

func main() {
	slog.SetDefault(logger.New(slog.LevelDebug))

	if err := runMain(); err != nil {

		slog.Error("shutting down due to error", logger.Err(err))
		return
	}
	slog.Info("shutdown complete")
}

func runMain() error {
	svcGroup, err := setupServices()
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

	return svcGroup.Run(ctx)
}

func setupServices() (service.Group, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parsing env config: %v", err)
	}

	var svc service.Service
	var svcGroup service.Group

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

	// Initialize tools
	tools := []openai.ToolFunction{
		tools.NewGetChatSettings(settingsRepository),
		tools.NewSetSystemPrompt(settingsRepository),
		tools.NewSetModel(settingsRepository),
	}

	//TODO rename textGptClient to openAiClient
	openAIClient, err := openai.NewClient(cfg.OpenAIToken, chatRepository, settingsRepository, tools)
	if err != nil {
		return nil, fmt.Errorf("creating open ai client: %v", err)
	}
	imageGptClient := chatgpt.NewImageClient(cfg.OpenAIToken)
	audioGptClient := chatgpt.NewAudioClient(cfg.OpenAIToken)

	oggToMp3Converter := converter.OggTomp3{}
	speechToTextConverter := converter.NewSpeechToText(audioGptClient)

	commands := []telegram.Command{
		// non ai commands
		command.NewInfo(telegramClient),
		command.NewClearChatHistory(chatRepository, telegramClient),
		command.NewSetChatTTL(telegramClient),

		// features
		command.NewGpt(openAIClient, chatRepository, telegramClient),
		command.NewVoice(telegramClient, &oggToMp3Converter, speechToTextConverter, openAIClient, imageGptClient, promptRepository, telegramClient),
		command.NewDraw(imageGptClient, promptRepository, telegramClient),
		command.NewVision(telegramClient, openAIClient, telegramClient),

		// callbacks
		command.NewDrawCallback(imageGptClient, promptRepository, telegramClient),
		command.NewSetChatTTLCallback(chatRepository, telegramClient),
	}

	commandHandler := telegram.NewCommandHandler(commands)

	if svc, err = service.NewTelegramListener(telegramClient, authenticator, commandHandler); err == nil {
		svcGroup = append(svcGroup, svc)
	} else {
		return nil, err
	}

	return svcGroup, nil
}
