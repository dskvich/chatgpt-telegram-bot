package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

const toolTypeFunction = "function"

type ToolFunction interface {
	Name() string
	Description() string
	Parameters() domain.Definition
	Function() any
}

type toolService struct {
	tools []domain.Tool
}

func NewToolService(toolFunctions []ToolFunction) (*toolService, error) {
	tools := make([]domain.Tool, len(toolFunctions))
	for i, t := range toolFunctions {
		if err := validateFunction(t); err != nil {
			return nil, fmt.Errorf("invalid tool function %q: %w", t.Name(), err)
		}

		tools[i] = domain.Tool{
			Type: toolTypeFunction,
			Function: &domain.Function{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
				Function:    t.Function(),
			},
		}
	}

	return &toolService{tools: tools}, nil
}

func (ts *toolService) Tools() []domain.Tool {
	return ts.tools
}

// InvokeFunction calls a specific tool by name with the provided arguments.
func (ts *toolService) InvokeFunction(ctx context.Context, chatID int64, name, args string) (string, error) {
	slog.DebugContext(ctx, "Invoking function", "name", name, "args", args)

	var tool *domain.Tool
	for i := range ts.tools {
		if ts.tools[i].Function.Name == name {
			tool = &ts.tools[i]
			break
		}
	}
	if tool == nil {
		return "", fmt.Errorf("tool not found: %q", name)
	}

	var parsedArgs map[string]interface{}
	if err := json.Unmarshal([]byte(args), &parsedArgs); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	slog.DebugContext(ctx, "Arguments parsed", "args", parsedArgs)

	function := tool.Function
	if err := validateArguments(function.Parameters, parsedArgs); err != nil {
		return "", fmt.Errorf("invalid arguments for function %q: %w", name, err)
	}

	handler := reflect.ValueOf(function.Function)
	if handler.Kind() != reflect.Func {
		return "", fmt.Errorf("function %q is not callable", name)
	}

	// prepare function args starting with (context, chatID, ...)
	funcArgs := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(chatID),
	}
	for _, param := range function.Parameters.Required {
		funcArgs = append(funcArgs, reflect.ValueOf(parsedArgs[param]))
	}

	results := handler.Call(funcArgs)
	if len(results) != 2 {
		return "", fmt.Errorf("function %q must return (string, error), got %d values", name, len(results))
	}

	result, ok := results[0].Interface().(string)
	if !ok {
		return "", fmt.Errorf("function %q returned non-string result", name)
	}

	var err error
	if results[1].Interface() != nil {
		err, _ = results[1].Interface().(error)
	}

	slog.DebugContext(ctx, "Function executed", "result", result, "err", err)
	return result, err
}

func validateFunction(t ToolFunction) error {
	if t.Name() == "" {
		return errors.New("function name cannot be empty")
	}
	if t.Function() == nil {
		return errors.New("function handler cannot be nil")
	}
	if reflect.TypeOf(t.Function).Kind() != reflect.Func {
		return errors.New("function handler must be callable")
	}
	return nil
}

func validateArguments(schema domain.Definition, args map[string]interface{}) error {
	for paramName, paramDef := range schema.Properties {
		value, ok := args[paramName]
		if !ok {
			return fmt.Errorf("missing required parameter %q", paramName)
		}

		if !isValidType(value, paramDef.Type) {
			return fmt.Errorf("parameter %q has invalid type: expected %q, got %T", paramName, paramDef.Type, value)
		}
	}
	return nil
}

func isValidType(value interface{}, expectedType domain.DataType) bool {
	switch expectedType {
	case domain.String:
		_, ok := value.(string)
		return ok
	case domain.Number:
		_, ok := value.(float64)
		return ok
	case domain.Integer:
		_, ok := value.(int)
		return ok
	case domain.Boolean:
		_, ok := value.(bool)
		return ok
	default:
		return false
	}
}
