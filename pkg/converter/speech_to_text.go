package converter

import (
	"fmt"
)

type SpeechTranscriber interface {
	Transcribe(filePath string) (string, error)
}

type speechToText struct {
	transcriber SpeechTranscriber
}

func NewSpeechToText(
	transcriber SpeechTranscriber,
) *speechToText {
	return &speechToText{
		transcriber: transcriber,
	}
}

func (s *speechToText) SpeechToText(filePath string) (string, error) {
	text, err := s.transcriber.Transcribe(filePath)
	if err != nil {
		return "", fmt.Errorf("transcribing audio file: %v", err)
	}

	return text, nil
}
