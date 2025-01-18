package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"reflect"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ChatRepository interface {
	SaveSession(chatID int64, session domain.ChatSession)
	GetSession(chatID int64) (domain.ChatSession, bool)
}

type SettingsRepository interface {
	GetAll(ctx context.Context, chatID int64) (map[string]string, error)
}

type ChatStyleRepository interface {
	GetActiveStyle(ctx context.Context, chatID int64) (*domain.ChatStyle, error)
}

type ToolFunction interface {
	Name() string
	Description() string
	Parameters() jsonschema.Definition
	Function() any
}

type client struct {
	token         string
	hc            *http.Client
	chatRepo      ChatRepository
	settingsRepo  SettingsRepository
	chatStyleRepo ChatStyleRepository
	tools         []ToolFunction
}

func NewClient(
	token string,
	chatRepo ChatRepository,
	settingsRepo SettingsRepository,
	chatStyleRepo ChatStyleRepository,
	tools []ToolFunction,
) (*client, error) {
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}
	return &client{
		token:         token,
		hc:            &http.Client{},
		chatRepo:      chatRepo,
		settingsRepo:  settingsRepo,
		chatStyleRepo: chatStyleRepo,
		tools:         tools,
	}, nil
}

func (c *client) CreateChatCompletion(chatID int64, text, base64image string) (string, error) {
	slog.Info("CreateChatCompletion", "chatID", chatID, "textLength", len(text), "hasImage", base64image != "")

	var content any
	if base64image != "" {
		content = []domain.Content{
			{Type: "image_url", ImageURL: &domain.ImageURL{URL: "data:image/jpeg;base64," + base64image}},
		}
		if text != "" {
			content = append([]domain.Content{{Type: "text", Text: text}}, content.([]domain.Content)...)
		}
	} else {
		content = text
	}

	session, err := c.getSession(chatID)
	if session == nil {
		return "", fmt.Errorf("getting session: %v", err)
	}

	session.Messages = append(session.Messages, domain.ChatMessage{Role: chatMessageRoleUser, Content: content})

	slog.Info("Requesting chat completion", "chatID", chatID, "model", session.ModelName, "messageCount", session.Messages)

	response, err := c.processChatCompletion(session)
	if err != nil {
		return "", fmt.Errorf("processing chat completion: %w", err)
	}

	if response.Content != nil {
		c.saveSession(chatID, session)
		return fmt.Sprint(response.Content), nil
	}

	if err := c.handleToolCalls(chatID, session, response.ToolCalls); err != nil {
		return "", err
	}

	if finalResponse, err := c.processChatCompletion(session); err == nil && finalResponse.Content != nil {
		c.saveSession(chatID, session)
		return fmt.Sprint(finalResponse.Content), nil
	}

	return "", fmt.Errorf("no completion response from API")
}

func (c *client) getSession(chatID int64) (*domain.ChatSession, error) {
	session, ok := c.chatRepo.GetSession(chatID)
	if ok {
		return &session, nil
	}

	slog.Info("Creating new session", "chatID", chatID)
	newSession, err := c.createNewSession(chatID)
	if err != nil {
		return nil, fmt.Errorf("creating new session: %v", err)
	}

	return newSession, nil
}

func (c *client) createNewSession(chatID int64) (*domain.ChatSession, error) {
	ctx := context.Background()
	settings, err := c.settingsRepo.GetAll(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("fetching system settings: %w", err)
	}

	model, found := settings[domain.ModelKey]
	if !found {
		model = domain.DefaultModel
	}

	chatStyle, err := c.chatStyleRepo.GetActiveStyle(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("fetching active chat style: %w", err)
	}

	messages := []domain.ChatMessage{}
	if chatStyle != nil && chatStyle.Description != "" {
		messages = append(messages, domain.ChatMessage{
			Role:    chatMessageRoleDeveloper,
			Content: chatStyle.Description,
		})
	}

	return &domain.ChatSession{
		ModelName: model,
		Messages:  messages,
	}, nil
}

func (c *client) processChatCompletion(session *domain.ChatSession) (*domain.ChatMessage, error) {
	req := c.buildChatCompletionRequest(session.ModelName, session.Messages)
	resp, err := c.sendChatCompletionRequest(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %v", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	response := &resp.Choices[0].Message
	session.Messages = append(session.Messages, *response)

	if response.Role != chatMessageRoleAssistant {
		return nil, fmt.Errorf("unexpected role: received %v, expected %v", response.Role, chatMessageRoleAssistant)
	}
	return response, nil
}

func (c *client) buildChatCompletionRequest(model string, messages []domain.ChatMessage) *chatCompletionsRequest {
	tools := make([]tool, 0, len(c.tools))
	for _, t := range c.tools {
		tools = append(tools, tool{
			Type: toolTypeFunction,
			Function: &function{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}

	return &chatCompletionsRequest{
		Model:     model,
		Messages:  messages,
		MaxTokens: 4096,
		Tools:     tools,
	}
}

func (c *client) saveSession(chatID int64, session *domain.ChatSession) {
	c.chatRepo.SaveSession(chatID, *session)
}

func (c *client) handleToolCalls(chatID int64, session *domain.ChatSession, toolCalls []domain.ToolCall) error {
	slog.Info("Handling tool calls", "chatID", chatID, "toolCallCount", len(toolCalls))

	for _, toolCall := range toolCalls {
		toolResponse, err := c.callTool(chatID, toolCall)
		if err != nil {
			return fmt.Errorf("calling tool %s: %w", toolCall.Function.Name, err)
		}

		session.Messages = append(session.Messages, domain.ChatMessage{
			ToolCallID: toolCall.ID,
			Role:       chatMessageRoleTool,
			Name:       toolCall.Function.Name,
			Content:    toolResponse,
		})
		slog.Info("Tool call succeeded", "chatID", chatID, "tool", toolCall.Function.Name)
	}
	return nil
}

func (c *client) callTool(chatID int64, toolCall domain.ToolCall) (string, error) {
	// Find the tool in the tools slice
	var toolFunction ToolFunction
	for _, tool := range c.tools {
		if tool.Name() == toolCall.Function.Name {
			toolFunction = tool
			break
		}
	}

	if toolFunction == nil {
		slog.Error("Tool function not found", "chatID", chatID, "tool", toolCall.Function.Name)
		return "", fmt.Errorf("no function available for tool %s", toolCall.Function.Name)
	}

	fn := reflect.ValueOf(toolFunction.Function())
	fnType := fn.Type()

	if fnType.Kind() != reflect.Func {
		slog.Error("Tool is not a valid function", "chatID", chatID, "tool", toolCall.Function.Name)
		return "", fmt.Errorf("tool function %s is not a function", toolCall.Function.Name)
	}

	// Start building arguments with chatID as the first parameter
	args := []reflect.Value{reflect.ValueOf(chatID)}

	// Decode the arguments JSON string into a map
	var argumentMap map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &argumentMap); err != nil {
		slog.Error("Failed to decode tool function arguments", "chatID", chatID, "tool", toolCall.Function.Name, "error", err)
		return "", fmt.Errorf("failed to decode arguments: %v", err)
	}

	// Prepare log data for expected parameters
	expectedParameters := make(map[string]jsonschema.DataType)
	for paramName, paramDef := range toolFunction.Parameters().Properties {
		expectedParameters[paramName] = paramDef.Type
	}

	// Log attempt to call with expected arguments
	slog.Info("Attempting to call tool function with arguments",
		"chatID", chatID,
		"tool", toolCall.Function.Name,
		"expectedArguments", expectedParameters,
		"providedArguments", argumentMap,
	)

	// Match arguments by name using the parameters definition
	for _, requiredParam := range toolFunction.Parameters().Required {
		value, exists := argumentMap[requiredParam]
		if !exists {
			slog.Error("Missing required argument", "chatID", chatID, "tool", toolCall.Function.Name, "parameter", requiredParam)
			return "", fmt.Errorf("missing required argument: %s", requiredParam)
		}

		// Ensure the value type is correct (assuming all parameters are strings here)
		val := reflect.ValueOf(value)
		if val.Kind() != reflect.String {
			slog.Error("Argument type mismatch", "chatID", chatID, "tool", toolCall.Function.Name, "parameter", requiredParam, "expectedType", "string", "providedType", val.Kind().String())
			return "", fmt.Errorf("argument type mismatch for parameter %s: expected string, got %s", requiredParam, val.Kind().String())
		}

		args = append(args, val)
	}

	// Validate argument count
	if len(args) != fnType.NumIn() {
		slog.Error("Argument count mismatch", "chatID", chatID, "tool", toolCall.Function.Name, "expected", fnType.NumIn(), "provided", len(args))
		return "", fmt.Errorf("argument count mismatch: expected %d, got %d", fnType.NumIn(), len(args))
	}

	// Call the function dynamically
	results := fn.Call(args)

	// Check and return the results
	if len(results) != 2 {
		return "", fmt.Errorf("unexpected number of return values from function %s", toolCall.Function.Name)
	}

	// Handling the result string
	result, ok := results[0].Interface().(string)
	if !ok {
		return "", fmt.Errorf("unexpected type for return value from function %s", toolCall.Function.Name)
	}

	// Handling the error return value
	var err error
	if results[1].IsValid() && !results[1].IsNil() {
		err, ok = results[1].Interface().(error)
		if !ok {
			return "", fmt.Errorf("unexpected type for error return value from function %s", toolCall.Function.Name)
		}
	}

	return result, err
}

func (c *client) sendChatCompletionRequest(request *chatCompletionsRequest) (*chatCompletionsResponse, error) {
	url := "https://api.openai.com/v1/chat/completions"
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResponse chatCompletionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResponse); err != nil {
		return nil, fmt.Errorf("decoding response data: %v", err)
	}

	return &chatResponse, nil
}

func (c *client) TranscribeAudio(audioFilePath string) (string, error) {
	const apiURL = "https://api.openai.com/v1/audio/transcriptions"
	const model = "whisper-1"

	slog.Info("Transcribing audio", "audioFilePath", audioFilePath, "model", model)

	// Create multipart form data
	requestBody, contentType, err := createMultipartForm(audioFilePath, model)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, requestBody)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	var responseBody struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return "", fmt.Errorf("decoding response data: %v", err)
	}

	slog.Info("Transcription successful", "text", responseBody.Text)

	return responseBody.Text, nil
}

// createMultipartForm creates a multipart form with the file and model fields
func createMultipartForm(filePath string, model string) (*bytes.Buffer, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add the file field
	fileWriter, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		return nil, "", fmt.Errorf("error creating form file: %w", err)
	}
	if _, err := io.Copy(fileWriter, file); err != nil {
		return nil, "", fmt.Errorf("error copying file: %w", err)
	}

	// Add the model field
	if err := writer.WriteField("model", model); err != nil {
		return nil, "", fmt.Errorf("error writing model field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("error closing writer: %w", err)
	}

	return &requestBody, writer.FormDataContentType(), nil
}

func (c *client) GenerateImage(chatID int64, prompt string) ([]byte, error) {
	const apiURL = "https://api.openai.com/v1/images/generations"
	const model = "dall-e-3"

	slog.Info("Generating image", "chatID", chatID, "prompt", prompt, "model", model)

	settings, err := c.settingsRepo.GetAll(context.TODO(), chatID)
	if err != nil {
		slog.Error("fetching system settings", "chatID", chatID, logger.Err(err))
	}

	imageStyle, found := settings[domain.ImageStyleKey]
	if !found {
		imageStyle = domain.ImageStyleDefault
	}

	var requestBody = struct {
		Model          string `json:"model"`
		Prompt         string `json:"prompt"`
		N              int    `json:"n"`
		Size           string `json:"size"`
		ResponseFormat string `json:"response_format"`
		Style          string `json:"style"`
	}{
		Model:          model,
		Prompt:         prompt,
		N:              1,
		Size:           "1024x1024",
		ResponseFormat: "b64_json",
		Style:          imageStyle,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling chat request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	var responseBody struct {
		Created int `json:"created"`
		Data    []struct {
			B64Json []byte `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return nil, fmt.Errorf("decoding response data: %v", err)
	}

	if len(responseBody.Data) > 0 {
		slog.Info("Image generation successful", "chatID", chatID, slog.Int("image_count", len(responseBody.Data)))
		return responseBody.Data[0].B64Json, nil
	}

	return nil, fmt.Errorf("no response from API")
}
