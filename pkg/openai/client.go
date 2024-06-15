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
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
)

type ChatRepository interface {
	SaveSession(chatID int64, session domain.ChatSession)
	GetSession(chatID int64) (domain.ChatSession, bool)
}

type SettingsRepository interface {
	GetByKey(ctx context.Context, chatID int64, key string) (string, error)
}

type toolFunctionMap map[string]any

type ToolFunction interface {
	Name() string
	Description() string
	Parameters() jsonschema.Definition
	Function() any
}

type client struct {
	token              string
	hc                 *http.Client
	chatRepo           ChatRepository
	settingsRepo       SettingsRepository
	tools              []ToolFunction
	availableFunctions toolFunctionMap
}

func NewClient(
	token string,
	chatRepo ChatRepository,
	settingsRepo SettingsRepository,
	tools []ToolFunction,
) (*client, error) {
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}
	return &client{
		token:              token,
		hc:                 &http.Client{},
		chatRepo:           chatRepo,
		settingsRepo:       settingsRepo,
		tools:              tools,
		availableFunctions: createAvailableFunctions(tools),
	}, nil
}

func createAvailableFunctions(tools []ToolFunction) toolFunctionMap {
	m := make(toolFunctionMap)
	for _, t := range tools {
		m[t.Name()] = t.Function()
	}
	return m
}

func (c *client) CreateChatCompletion(chatID int64, prompt string) (string, error) {
	session := c.getSession(chatID)

	userMessage := domain.ChatMessage{Role: chatMessageRoleUser, Content: prompt}
	session.Messages = append(session.Messages, userMessage)

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

func (c *client) getSession(chatID int64) *domain.ChatSession {
	session, ok := c.chatRepo.GetSession(chatID)
	if ok {
		return &session
	}
	return c.createNewSession(chatID)
}

func (c *client) createNewSession(chatID int64) *domain.ChatSession {
	systemPrompt, err := c.settingsRepo.GetByKey(context.TODO(), chatID, domain.SystemPromptKey)
	if err != nil {
		slog.Error("fetching system prompt", "chatID", chatID, logger.Err(err))
	}

	return &domain.ChatSession{
		ModelName: "gpt-4o",
		Messages: []domain.ChatMessage{
			{Role: chatMessageRoleSystem, Content: systemPrompt},
		},
	}
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
	for _, toolCall := range toolCalls {
		toolResponse, err := c.callTool(chatID, toolCall)
		if err != nil {
			return fmt.Errorf("calling tool %s: %w", toolCall.Function.Name, err)
		}

		toolMessage := domain.ChatMessage{
			ToolCallID: toolCall.ID,
			Role:       chatMessageRoleTool,
			Name:       toolCall.Function.Name,
			Content:    toolResponse,
		}
		session.Messages = append(session.Messages, toolMessage)
	}
	return nil
}

func (c *client) callTool(chatID int64, toolCall domain.ToolCall) (string, error) {
	functionToCall, exists := c.availableFunctions[toolCall.Function.Name]
	if !exists {
		return "", fmt.Errorf("no function available for tool %s", toolCall.Function.Name)
	}

	fn := reflect.ValueOf(functionToCall)
	fnType := fn.Type()

	if fnType.Kind() != reflect.Func {
		return "", fmt.Errorf("tool function %s is not a function", toolCall.Function.Name)
	}

	// The first argument is always chatID
	args := []reflect.Value{reflect.ValueOf(chatID)}

	// If the function has two parameters, add the second one from toolCall.Function.Arguments
	if fnType.NumIn() == 2 {
		// Assuming the second parameter is of type string and is passed in toolCall.Function.Arguments
		args = append(args, reflect.ValueOf(toolCall.Function.Arguments))
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
