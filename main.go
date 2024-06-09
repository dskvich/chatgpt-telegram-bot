package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v9"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/auth"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/chatgpt"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/converter"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/database"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/digitalocean"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/repository"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/service"
	telegramservice "github.com/dskvich/chatgpt-telegram-bot/pkg/service/telegram"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/telegram"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/telegram/command"
)

type Config struct {
	OpenAIToken               string  `env:"OPEN_AI_TOKEN,required"`
	TelegramBotToken          string  `env:"TELEGRAM_BOT_TOKEN,required"`
	TelegramAuthorizedUserIDs []int64 `env:"TELEGRAM_AUTHORIZED_USER_IDS" envSeparator:" "`
	DigitalOceanAccessToken   string  `env:"DIGITALOCEAN_ACCESS_TOKEN,required"`
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

	bot, err := telegram.NewBot(cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("creating telegram bot: %v", err)
	}
	authenticator := auth.NewAuthenticator(cfg.TelegramAuthorizedUserIDs)

	db, err := database.NewPostgres(cfg.PgURL, cfg.PgHost)
	if err != nil {
		return nil, fmt.Errorf("creating db: %v", err)
	}

	chatRepository := repository.NewChatRepository()
	promptRepository := repository.NewPromptRepository(db)
	settingsRepository := repository.NewSettingsRepository(db)

	textGptClient := chatgpt.NewTextClient(cfg.OpenAIToken, chatRepository, settingsRepository)
	imageGptClient := chatgpt.NewImageClient(cfg.OpenAIToken)
	audioGptClient := chatgpt.NewAudioClient(cfg.OpenAIToken)
	visionGptClient := chatgpt.NewVisionClient(cfg.OpenAIToken, chatRepository)

	doClient := digitalocean.NewClient(cfg.DigitalOceanAccessToken)

	oggToMp3Converter := converter.OggTomp3{}
	speechToTextConverter := converter.NewSpeechToText(audioGptClient)

	messagesCh := make(chan domain.Message)
	commands := []telegram.Command{
		// non ai commands
		command.NewInfo(messagesCh),
		command.NewCleanChat(chatRepository, messagesCh),
		command.NewBalance(doClient, messagesCh),
		command.NewSettings(settingsRepository, messagesCh),

		// awaitings
		command.NewSettingsAwaiting(chatRepository, settingsRepository, messagesCh),

		// features
		command.NewGpt(textGptClient, chatRepository, messagesCh),
		command.NewVoice(bot, &oggToMp3Converter, speechToTextConverter, textGptClient, imageGptClient, promptRepository, messagesCh),
		command.NewDraw(imageGptClient, promptRepository, messagesCh),
		command.NewVision(bot, visionGptClient, messagesCh),

		// callbacks
		command.NewDrawCallback(imageGptClient, promptRepository, messagesCh),
		command.NewSettingsCallback(chatRepository, messagesCh),
	}

	commandDispatcher := telegram.NewCommandDispatcher(commands)

	if svc, err = telegramservice.NewService(bot, authenticator, commandDispatcher, messagesCh); err == nil {
		svcGroup = append(svcGroup, svc)
	} else {
		return nil, err
	}

	return svcGroup, nil
}
