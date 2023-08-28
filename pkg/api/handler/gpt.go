package handler

import (
	"context"
	"net/http"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/api/response"
)

type GptProvider interface {
	GenerateSingleResponse(ctx context.Context, prompt string) (string, error)
}

type gpt struct {
	provider GptProvider
	writer   response.JSONResponseWriter
}

func NewGpt(provider GptProvider) *gpt {
	return &gpt{
		provider: provider,
		writer:   response.JSONResponseWriter{},
	}
}

func (g *gpt) GenerateResponse(w http.ResponseWriter, r *http.Request) {
	prompt := r.URL.Query().Get("prompt")
	if prompt == "" {
		g.writer.WriteErrorResponse(w, http.StatusBadRequest, "Prompt parameter is missing or empty.")
		return
	}

	resp, err := g.provider.GenerateSingleResponse(r.Context(), prompt)
	if err != nil {
		g.writer.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	g.writer.WriteSuccessResponse(w, map[string]string{
		"response": resp,
	})
}
