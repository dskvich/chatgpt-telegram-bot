package domain

import "strings"

type CommandText string

// ExtractAfterKeywords extracts the text after any of the provided keywords.
func (c CommandText) ExtractAfterKeywords(keywords []string) string {
	lowerText := strings.ToLower(string(c))
	for _, keyword := range keywords {
		if idx := strings.Index(lowerText, keyword); idx != -1 {
			return strings.TrimSpace(lowerText[idx+len(keyword):])
		}
	}
	return string(c)
}

// ContainsAny checks if any of the provided keywords are present in the text.
func (c CommandText) ContainsAny(keywords []string) bool {
	lowerText := strings.ToLower(string(c))
	for _, keyword := range keywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}
	return false
}
