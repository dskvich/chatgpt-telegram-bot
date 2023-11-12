package chatgpt

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type imageClient struct {
	api   *openai.Client
	token string
}

func NewImageClient(token string) *imageClient {
	return &imageClient{
		token: token,
		api:   openai.NewClient(token),
	}
}

func (c *imageClient) GenerateImage(prompt string) ([]byte, error) {
	req := openai.ImageRequest{
		Prompt:         prompt,
		Size:           openai.CreateImageSize512x512,
		ResponseFormat: openai.CreateImageResponseFormatB64JSON,
		N:              1,
	}

	resp, err := c.api.CreateImage(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("creating image: %v", err)
	}

	imgBytes, err := base64.StdEncoding.DecodeString(resp.Data[0].B64JSON)
	if err != nil {
		return nil, fmt.Errorf("base64 decoding: %v", err)
	}

	return imgBytes, nil
}
