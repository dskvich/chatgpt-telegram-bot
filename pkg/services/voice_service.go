package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/keyword"
)

const (
	voiceTempFilePerm = 0644
	voiceTempDir      = "tmp/voices"
)

type AudioConverter interface {
	ConvertToMP3(inputPath string) (string, error)
}

type OpenAITranscriber interface {
	TranscribeAudio(ctx context.Context, audioFilePath string) (string, error)
}

type VoiceFileDownloader interface {
	DownloadFile(ctx context.Context, fileID string) ([]byte, error)
}

type voiceService struct {
	converter    AudioConverter
	transcriber  OpenAITranscriber
	downloader   VoiceFileDownloader
	imageService *imageService
	textService  *textService
	responseCh   chan<- domain.Response
}

func NewVoiceService(
	converter AudioConverter,
	transcriber OpenAITranscriber,
	downloader VoiceFileDownloader,
	imageService *imageService,
	textService *textService,
	responseCh chan<- domain.Response,
) *voiceService {
	return &voiceService{
		converter:    converter,
		transcriber:  transcriber,
		downloader:   downloader,
		imageService: imageService,
		textService:  textService,
		responseCh:   responseCh,
	}
}

func (v *voiceService) GenerateFromVoice(ctx context.Context, chatID int64, voiceFileID string) {
	voiceData, err := v.downloader.DownloadFile(ctx, voiceFileID)
	if err != nil {
		v.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("downloading voice file: %w", err)}
		return
	}

	if err := os.MkdirAll(voiceTempDir, os.ModePerm); err != nil {
		v.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("creating voice temp directory: %w", err)}
		return
	}

	voiceFilePath := filepath.Join(voiceTempDir, fmt.Sprintf("voice-%d.ogg", time.Now().UnixNano()))
	if err := os.WriteFile(voiceFilePath, voiceData, voiceTempFilePerm); err != nil {
		v.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("saving voice file: %w", err)}
		return
	}

	mp3Path, err := v.converter.ConvertToMP3(voiceFilePath)
	if err != nil {
		v.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("converting voice file to MP3: %w", err)}
		return
	}

	prompt, err := v.transcriber.TranscribeAudio(ctx, mp3Path)
	if err != nil {
		v.responseCh <- domain.Response{ChatID: chatID, Err: fmt.Errorf("transcribing audio file: %w", err)}
		return
	}

	if keyword.IsImageRequest(prompt) {
		v.imageService.GenerateImage(ctx, chatID, prompt)
		return
	}

	v.textService.GenerateFromText(ctx, chatID, prompt)
}
