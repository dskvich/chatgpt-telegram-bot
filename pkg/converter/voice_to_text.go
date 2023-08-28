package converter

import (
	"fmt"
	"os"
	"os/exec"
	"path"
)

type SpeechTranscriber interface {
	Transcribe(filePath string) (string, error)
}

type VoiceToText struct {
	transcriber SpeechTranscriber
}

func (v *VoiceToText) Convert(voiceFilePath string) (string, error) {
	if path.Ext(voiceFilePath) == ".ogg" || path.Ext(voiceFilePath) == ".oga" {
		newFilePath, err := convertAudioToMp3(voiceFilePath)
		defer os.Remove(voiceFilePath)
		if err != nil {
			return "", fmt.Errorf("converting file: %v", err)
		}
		voiceFilePath = newFilePath
	}

	text, err := v.transcriber.Transcribe(voiceFilePath)
	if err != nil {
		return "", fmt.Errorf("transcribing audio file: %v", err)
	}

	return text, nil
}

func convertAudioToMp3(filePath string) (string, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", fmt.Errorf("looking for `ffmpeg`: %w", err)
	}

	newFilePath := filePath + ".mp3"

	cmd := exec.Command("ffmpeg", "-i", filePath, newFilePath)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return newFilePath, fmt.Errorf("running `ffmpeg`: %v", err)
	}

	return newFilePath, nil
}
