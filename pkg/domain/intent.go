package domain

type Intent string

const (
	IntentGenerateImage Intent = "generate_image"
	IntentGenerateText  Intent = "generate_text"
)
