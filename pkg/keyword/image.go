package keyword

import "strings"

var imageKeywords = []string{"рисуй", "draw"}

func IsImageRequest(text string) bool {
	text = strings.ToLower(text)
	for _, kw := range imageKeywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}
