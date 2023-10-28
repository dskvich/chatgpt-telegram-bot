package converter

import (
	"fmt"
	"os"
	"os/exec"
	"path"
)

type OggTomp3 struct{}

func (o *OggTomp3) ConvertToMP3(inputPath string) (string, error) {
	var (
		outputPath string
		err        error
	)

	if path.Ext(inputPath) == ".ogg" || path.Ext(inputPath) == ".oga" {
		outputPath, err = convertAudioToMp3(inputPath)
		defer os.Remove(inputPath)
		if err != nil {
			return "", fmt.Errorf("converting file: %v", err)
		}
	}

	return outputPath, err
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
