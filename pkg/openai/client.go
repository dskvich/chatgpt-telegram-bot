package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"

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
			{Type: "image_url", ImageUrl: &domain.ImageUrl{Url: "data:image/jpeg;base64," + base64image}},
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
		return "", fmt.Errorf("processing chat completion: %v", err)
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
		return nil, fmt.Errorf("fetching system settings: %v", err)
	}

	model, found := settings[domain.ModelKey]
	if !found {
		model = domain.DefaultModel
	}

	chatStyle, err := c.chatStyleRepo.GetActiveStyle(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("fetching active chat style: %v", err)
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
