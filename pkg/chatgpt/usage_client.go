package chatgpt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type usageClient struct {
	token      string
	hc         *http.Client
	modelCosts map[string]domain.GptModelCost
}

func NewUsageClient(token string) *usageClient {
	return &usageClient{
		token:      token,
		hc:         &http.Client{},
		modelCosts: domain.DefaultGptModelCosts(),
	}
}

func (c *usageClient) GetUsage() (string, error) {
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

func (c *usageClient) calculateWhisperCost(whisperData []gptWhisperApiDataType) map[string]float64 {
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

func (c *usageClient) calculateModelCost(usageData []gptDataType) map[string]float64 {
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

func (c *usageClient) calculateDalleCost(usageData []gptDalleApiDataType) map[string]float64 {
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
