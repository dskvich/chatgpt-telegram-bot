package chatgpt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type ChatRepository interface {
	AddMessage(chatID int64, msg openai.ChatCompletionMessage)
	GetMessages(chatID int64) []openai.ChatCompletionMessage
	RemoveMessages(chatID int64)
}

type client struct {
	api        *openai.Client
	token      string
	hc         *http.Client
	modelCosts map[string]domain.GptModelCost
	repo       ChatRepository
}

func NewClient(token string, repo ChatRepository) *client {
	return &client{
		token:      token,
		api:        openai.NewClient(token),
		hc:         &http.Client{},
		modelCosts: domain.DefaultGptModelCosts(),
		repo:       repo,
	}
}

func (c *client) GetUsage() (string, error) {
	now := time.Now()
	url := "https://api.openai.com/v1/usage?date=" + now.Format("2006-01-02")

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching usage data: %v", err)
	}

	var usageData gptUsageData
	if err := json.NewDecoder(resp.Body).Decode(&usageData); err != nil {
		return "", fmt.Errorf("decoding usage data: %v", err)
	}

	modelTotalCost := make(map[string]float64)

	for model, cost := range c.calculateWhisperCost(usageData.WhisperApiData) {
		modelTotalCost[model] += cost
	}

	for model, cost := range c.calculateModelCost(usageData.Data) {
		modelTotalCost[model] += cost
	}

	for model, cost := range c.calculateDalleCost(usageData.DalleApiData) {
		modelTotalCost[model] += cost
	}

	return generateUsageMessage(modelTotalCost), nil
}

type gptDataType struct {
	AggregationTimestamp  int    `json:"aggregation_timestamp"`
	NRequests             int    `json:"n_requests"`
	Operation             string `json:"operation"`
	SnapshotId            string `json:"snapshot_id"`
	NContext              int    `json:"n_context"`
	NContextTokensTotal   int    `json:"n_context_tokens_total"`
	NGenerated            int    `json:"n_generated"`
	NGeneratedTokensTotal int    `json:"n_generated_tokens_total"`
}

type gptWhisperApiDataType struct {
	Timestamp   int    `json:"timestamp"`
	ModelId     string `json:"model_id"`
	NumSeconds  int    `json:"num_seconds"`
	NumRequests int    `json:"num_requests"`
}

type gptDalleApiDataType struct {
	Timestamp   int    `json:"timestamp"`
	NumImages   int    `json:"num_images"`
	NumRequests int    `json:"num_requests"`
	ImageSize   string `json:"image_size"`
	Operation   string `json:"operation"`
}

type gptUsageData struct {
	Object          string                  `json:"object"`
	Data            []gptDataType           `json:"data"`
	FtData          []interface{}           `json:"ft_data"`
	DalleApiData    []gptDalleApiDataType   `json:"dalle_api_data"`
	WhisperApiData  []gptWhisperApiDataType `json:"whisper_api_data"`
	CurrentUsageUsd float64                 `json:"current_usage_usd"`
}

func generateUsageMessage(totalCost map[string]float64) string {
	var message string
	var total float64

	message = "OpenAI API Usage for Today:\n\n"

	for model, cost := range totalCost {
		modelUsage := fmt.Sprintf("%s: $%.2f\n", model, cost)
		message += modelUsage
		total += cost
	}

	message += fmt.Sprintf("\nTotal: $%.2f", total)

	return message
}

func (c *client) calculateWhisperCost(whisperData []gptWhisperApiDataType) map[string]float64 {
	totalCost := make(map[string]float64)

	for _, data := range whisperData {
		model := data.ModelId
		cost, ok := c.modelCosts[model]
		if ok {
			contextCost := float64(data.NumSeconds) * cost.Context
			totalCost[model] += contextCost
		}
	}

	return totalCost
}

func (c *client) calculateModelCost(usageData []gptDataType) map[string]float64 {
	totalCost := make(map[string]float64)

	for _, data := range usageData {
		model := data.SnapshotId
		contextTokens := data.NContextTokensTotal
		generatedTokens := data.NGeneratedTokensTotal

		cost, ok := c.modelCosts[model]
		if ok {
			contextCost := float64(contextTokens) / 1000.0 * cost.Context
			generatedCost := float64(generatedTokens) / 1000.0 * cost.Generated
			totalCost[model] += contextCost + generatedCost
		}
	}

	return totalCost
}

func (c *client) calculateDalleCost(usageData []gptDalleApiDataType) map[string]float64 {
	totalCost := make(map[string]float64)

	for _, data := range usageData {
		model := fmt.Sprintf("dalle-%s", data.ImageSize)

		cost, ok := c.modelCosts[model]
		if ok {
			totalCost[model] += float64(data.NumImages) * cost.Image
		}
	}

	return totalCost
}

func (c *client) Transcribe(filePath string) (string, error) {
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

func (c *client) GenerateImage(prompt string) ([]byte, error) {
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

// GenerateSingleResponse function for HTTP API
func (c *client) GenerateSingleResponse(ctx context.Context, prompt string) (string, error) {
	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	}

	resp, err := c.api.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    openai.GPT4,
			Messages: []openai.ChatCompletionMessage{message},
		},
	)
	if err != nil {
		return "", fmt.Errorf("creating completion: %v", err)
	}

	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no completion response")
}

func (c *client) GenerateChatResponse(chatID int64, prompt string) (string, error) {
	skipFlags := []string{"full answer", "in details", "полный ответ", "подробно"}

	prompt = addSuffixIfNeeded(prompt, " [Короткий ответ]", skipFlags)

	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	}

	messages := c.repo.GetMessages(chatID)
	messages = append(messages, message)

	resp, err := c.api.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT4,
			Messages: messages,
		},
	)
	if err != nil {
		return "", fmt.Errorf("creating completion: %v", err)
	}

	c.repo.AddMessage(chatID, message)

	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no completion response")
}

func addSuffixIfNeeded(prompt, suffix string, skipFlags []string) string {
	prompt = strings.ToLower(prompt)

	for _, flag := range skipFlags {
		if strings.Contains(prompt, flag) {
			return prompt
		}
	}

	return prompt + suffix
}
