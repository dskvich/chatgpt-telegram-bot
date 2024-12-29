package domain

const (
	ImageStyleCallbackPrefix = "image_style_"
	ImageStyleKey            = "image_style"
	ImageStyleDefault        = "vivid"
)

var ImageStyles = map[string]string{
	"vivid":   "Яркий",
	"natural": "Естественный",
}

func GetImageStylePrompt() string {
	return "Выберите стиль генерации изображений:"
}
