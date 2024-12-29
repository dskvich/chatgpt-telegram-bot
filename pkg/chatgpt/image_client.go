package chatgpt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
)

type SettingsRepository interface {
	GetAll(ctx context.Context, chatID int64) (map[string]string, error)
}

type imageClient struct {
	token        string
	hc           *http.Client
	settingsRepo SettingsRepository
}

func NewImageClient(token string, settingsRepo SettingsRepository) *imageClient {
	return &imageClient{
		token:        token,
		hc:           &http.Client{},
		settingsRepo: settingsRepo,
	}
}

func (c *imageClient) GenerateImage(chatID int64, prompt string) ([]byte, error) {
	settings, err := c.settingsRepo.GetAll(context.TODO(), chatID)
	if err != nil {
		slog.Error("fetching system settings", "chatID", chatID, logger.Err(err))
	}

	imageStyle, found := settings[domain.ImageStyleKey]
	if !found {
		imageStyle = domain.ImageStyleDefault
	}

	// Prepare the request.
	chatRequest := imagesGenerationsRequest{
		Model:          "dall-e-3",
		Prompt:         prompt,
		N:              1,
		Size:           "1024x1024",
		ResponseFormat: "b64_json",
		Style:          imageStyle,
	}

	fmt.Println(imageStyle)
	return nil, nil

	// Send request to the API.
	url := "https://api.openai.com/v1/images/generations"
	resp, err := c.sendRequest(url, chatRequest)
	if err != nil {
		return nil, fmt.Errorf("sending request to %s: %v", url, err)
	}
	defer resp.Body.Close()

	// Process the response.
	var imageResponse imagesGenerationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&imageResponse); err != nil {
		return nil, fmt.Errorf("decoding response data: %v", err)
	}

	if len(imageResponse.Data) > 0 {
		return imageResponse.Data[0].B64Json, nil
	}

	return nil, fmt.Errorf("no response from API")
}

func (c *imageClient) sendRequest(url string, chatRequest imagesGenerationsRequest) (*http.Response, error) {
	body, err := json.Marshal(chatRequest)
	if err != nil {
		return nil, fmt.Errorf("marshaling chat request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing HTTP request: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}
