package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v9"
	"github.com/go-chi/chi/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/api/handler"
	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/auth"
	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/chatgpt"
	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/command"
	handler2 "github.com/sushkevichd/chatgpt-telegram-bot/pkg/command/handler"
	converter2 "github.com/sushkevichd/chatgpt-telegram-bot/pkg/converter"
	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/digitalocean"
	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/logger"
	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/repository"
	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/service"
	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/service/httpserver"
	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/service/telegram"
	telegrambot "github.com/sushkevichd/chatgpt-telegram-bot/pkg/telegram"
)

type Config struct {
	GptToken                  string  `env:"GPT_TOKEN,required"`
	TelegramBotToken          string  `env:"TELEGRAM_BOT_TOKEN,required"`
	TelegramAuthorizedUserIDs []int64 `env:"TELEGRAM_AUTHORIZED_USER_IDS" envSeparator:" "`
	DigitalOceanAccessToken   string  `env:"DIGITALOCEAN_ACCESS_TOKEN,required"`
	Port                      string  `env:"PORT" envDefault:"8080"`
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

	bot, err := telegrambot.NewBot(cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("creating telegram bot: %v", err)
	}
	authenticator := auth.NewAuthenticator(cfg.TelegramAuthorizedUserIDs)

	chatRepository := repository.NewChatRepository()
	gptClient := chatgpt.NewClient(cfg.GptToken, chatRepository)
	doClient := digitalocean.NewClient(cfg.DigitalOceanAccessToken)

	oggToMp3Converter := converter2.OggTomp3{}
	speechToTextConverter := converter2.NewSpeechToText(gptClient)

	messagesCh := make(chan domain.Message)

	defaultHandler := handler2.NewGpt(gptClient, messagesCh)
	handlers := []command.Handler{
		handler2.NewChat(chatRepository, messagesCh),
		handler2.NewVoice(bot, &oggToMp3Converter, speechToTextConverter, gptClient, messagesCh),
		handler2.NewBalance(doClient, messagesCh),
		handler2.NewUsage(gptClient, messagesCh),
		handler2.NewDraw(gptClient, messagesCh),
		handler2.NewDrawCallback(gptClient, messagesCh),
		handler2.NewGpt(gptClient, messagesCh),
	}
	dispatcher := command.NewDispatcher(handlers, defaultHandler)

	if svc, err = telegram.NewService(bot, authenticator, dispatcher, messagesCh); err == nil {
		svcGroup = append(svcGroup, svc)
	} else {
		return nil, err
	}

	router := chi.NewRouter()
	router.Get("/api/gpt/generate", handler.NewGpt(gptClient).GenerateResponse)

	if svc, err = httpserver.NewService(fmt.Sprintf(":%s", cfg.Port), router); err == nil {
		svcGroup = append(svcGroup, svc)
	} else {
		return nil, err
	}

	return svcGroup, nil
}
