package chatgpt

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sashabaranov/go-openai"
)

type audioClient struct {
	api   *openai.Client
	token string
	hc    *http.Client
}

func NewAudioClient(token string) *audioClient {
	return &audioClient{
		token: token,
		api:   openai.NewClient(token),
	}
}

func (c *audioClient) Transcribe(filePath string) (string, error) {
	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: filePath,
	}
	resp, err := c.api.CreateTranscription(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("creating transcription: %v", err)
	}

	return resp.Text, nil
}
