package audio

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TTSService provides text-to-speech functionality
type TTSService struct {
	audioDir string
}

const ttsRequestTimeout = 10 * time.Second

// NewTTSService creates a new TTS service
func NewTTSService(audioDir string) *TTSService {
	return &TTSService{
		audioDir: audioDir,
	}
}

// GenerateAudioFile converts text to speech and saves as MP3
// Returns the filename (not full path) on success
func (s *TTSService) GenerateAudioFile(text string) (string, error) {
	// Sanitize text for filename
	sanitized := strings.ToLower(strings.TrimSpace(text))
	sanitized = strings.ReplaceAll(sanitized, " ", "_")

	// Create filename
	filename := fmt.Sprintf("word_%s.mp3", sanitized)
	filepath := filepath.Join(s.audioDir, filename)

	// Check if file already exists
	if _, err := os.Stat(filepath); err == nil {
		// File already exists, return existing filename
		return filename, nil
	}

	// Generate audio using Google Translate TTS (free, no API key needed)
	if err := s.generateUsingGoogleTTS(text, filepath); err != nil {
		return "", fmt.Errorf("failed to generate audio: %w", err)
	}

	return filename, nil
}

// GenerateAudioFileWithPrefix converts text to speech and saves as MP3 with a custom filename prefix
// Returns the filename (not full path) on success
func (s *TTSService) GenerateAudioFileWithPrefix(text, prefix string) (string, error) {
	// Sanitize prefix for filename
	sanitized := strings.ToLower(strings.TrimSpace(prefix))
	sanitized = strings.ReplaceAll(sanitized, " ", "_")

	// Create filename with custom prefix
	filename := fmt.Sprintf("%s.mp3", sanitized)
	filepath := filepath.Join(s.audioDir, filename)

	// Check if file already exists
	if _, err := os.Stat(filepath); err == nil {
		// File already exists, return existing filename
		return filename, nil
	}

	// Generate audio using Google Translate TTS (free, no API key needed)
	if err := s.generateUsingGoogleTTS(text, filepath); err != nil {
		return "", fmt.Errorf("failed to generate audio: %w", err)
	}

	return filename, nil
}

// generateUsingGoogleTTS uses Google Translate's text-to-speech API
// This is a simple, free option that doesn't require API keys
func (s *TTSService) generateUsingGoogleTTS(text, outputPath string) error {
	// Google Translate TTS endpoint
	baseURL := "https://translate.google.com/translate_tts"

	// Build URL with parameters
	params := url.Values{}
	params.Set("ie", "UTF-8")
	params.Set("q", text)
	params.Set("tl", "en")
	params.Set("client", "tw-ob")
	params.Set("textlen", fmt.Sprintf("%d", len(text)))

	fullURL := baseURL + "?" + params.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), ttsRequestTimeout)
	defer cancel()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent (required by Google)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Make request
	client := &http.Client{Timeout: ttsRequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch audio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy audio data to file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write audio file: %w", err)
	}

	return nil
}

// BatchGenerateAudio generates audio files for multiple words
func (s *TTSService) BatchGenerateAudio(words []string) (map[string]string, error) {
	results := make(map[string]string)

	for _, word := range words {
		filename, err := s.GenerateAudioFile(word)
		if err != nil {
			return results, fmt.Errorf("failed to generate audio for '%s': %w", word, err)
		}
		results[word] = filename
	}

	return results, nil
}

// DeleteAudioFile removes an audio file
func (s *TTSService) DeleteAudioFile(filename string) error {
	filepath := filepath.Join(s.audioDir, filename)

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil // Already deleted
	}

	return os.Remove(filepath)
}

// GetAllAudioFiles returns a list of all MP3 files in the audio directory
func (s *TTSService) GetAllAudioFiles() ([]string, error) {
	files, err := os.ReadDir(s.audioDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio directory: %w", err)
	}

	var audioFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".mp3" {
			audioFiles = append(audioFiles, file.Name())
		}
	}

	return audioFiles, nil
}
