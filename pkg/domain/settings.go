package domain

const (
	SystemPromptKey = "system_prompt"
	ModelKey        = "model"

	DefaultModel = "gpt-4o-mini"
)

var SupportedModels = []string{
	"gpt-4o-mini",
	"gpt-4o",
	"gpt-4",
	"gpt-4-turbo",
	"gpt-3.5-turbo",
}
