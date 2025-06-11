package proxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

// AudioProcessor handles audio URL processing and conversion
type AudioProcessor struct {
	httpClient *http.Client
	maxSize    int64
}

// NewAudioProcessor creates a new audio processor with default settings
func NewAudioProcessor() *AudioProcessor {
	return &AudioProcessor{
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Longer timeout for audio files
		},
		maxSize: 25 * 1024 * 1024, // 25MB limit for audio files
	}
}

// AudioData represents processed audio data
type AudioData struct {
	Data   string `json:"data"`   // Base64 encoded audio data
	Format string `json:"format"` // Format: "wav" or "mp3"
}

// ProcessAudioURL downloads audio from a URL and converts it to WAV or MP3 format
func (p *AudioProcessor) ProcessAudioURL(ctx context.Context, audioURL string, headers map[string]string) (*AudioData, error) {
	logger.LogWithStructure(ctx, logger.LevelDebug, "Processing audio URL",
		map[string]interface{}{
			"url":     audioURL,
			"headers": headers,
		},
		nil, // request
		nil, // response
		nil) // error

	// Download the audio file
	audioData, contentType, err := p.downloadAudio(ctx, audioURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to download audio: %w", err)
	}

	// Determine the best output format based on the input
	outputFormat := p.determineOutputFormat(contentType)

	// Convert audio if necessary
	convertedData, err := p.convertAudio(ctx, audioData, contentType, outputFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to convert audio: %w", err)
	}

	// Encode to base64
	base64Data := base64.StdEncoding.EncodeToString(convertedData)

	logger.LogWithStructure(ctx, logger.LevelDebug, "Audio processed successfully",
		map[string]interface{}{
			"original_url":   audioURL,
			"content_type":   contentType,
			"output_format":  outputFormat,
			"original_size":  len(audioData),
			"converted_size": len(convertedData),
			"base64_length":  len(base64Data),
		},
		nil, // request
		nil, // response
		nil) // error

	return &AudioData{
		Data:   base64Data,
		Format: outputFormat,
	}, nil
}

// downloadAudio downloads audio from a URL with custom headers
func (p *AudioProcessor) downloadAudio(ctx context.Context, audioURL string, headers map[string]string) ([]byte, string, error) {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, audioURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "Generative-API-Router/1.0")

	// Add custom headers if provided
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	// Download the audio
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download audio: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download audio: status %d", resp.StatusCode)
	}

	// Get content type
	contentType := resp.Header.Get("Content-Type")

	// Read with size limit
	limitedReader := io.LimitReader(resp.Body, p.maxSize)
	audioData, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read audio data: %w", err)
	}

	// Check if we hit the size limit
	if int64(len(audioData)) >= p.maxSize {
		return nil, "", fmt.Errorf("audio size exceeds limit of %d bytes", p.maxSize)
	}

	return audioData, contentType, nil
}

// determineOutputFormat determines the best output format based on input
func (p *AudioProcessor) determineOutputFormat(contentType string) string {
	// If already MP3 or WAV, keep the same format
	if strings.Contains(contentType, "audio/mp3") || strings.Contains(contentType, "audio/mpeg") {
		return "mp3"
	}
	if strings.Contains(contentType, "audio/wav") || strings.Contains(contentType, "audio/wave") {
		return "wav"
	}

	// Default to MP3 for smaller file size
	return "mp3"
}

// convertAudio converts audio data to the specified format using FFmpeg
func (p *AudioProcessor) convertAudio(ctx context.Context, audioData []byte, inputContentType, outputFormat string) ([]byte, error) {
	// If already in the desired format, return as-is
	if (outputFormat == "mp3" && (strings.Contains(inputContentType, "audio/mp3") || strings.Contains(inputContentType, "audio/mpeg"))) ||
		(outputFormat == "wav" && (strings.Contains(inputContentType, "audio/wav") || strings.Contains(inputContentType, "audio/wave"))) {
		return audioData, nil
	}

	// Create temporary input file
	inputFile, err := os.CreateTemp("/tmp", "audio_input_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %w", err)
	}
	defer os.Remove(inputFile.Name())
	defer inputFile.Close()

	// Write input data
	_, err = inputFile.Write(audioData)
	if err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}
	inputFile.Close()

	// Create temporary output file
	outputExt := outputFormat
	outputFile, err := os.CreateTemp("/tmp", fmt.Sprintf("audio_output_*.%s", outputExt))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	defer os.Remove(outputFile.Name())
	outputFile.Close()

	// Prepare FFmpeg command based on output format
	var args []string
	args = append(args, "-i", inputFile.Name()) // Input file
	args = append(args, "-y")                   // Overwrite output

	// Set quality parameters based on output format
	switch outputFormat {
	case "mp3":
		args = append(args, "-acodec", "mp3") // MP3 codec
		args = append(args, "-b:a", "128k")   // Bitrate 128kbps
		args = append(args, "-ar", "44100")   // Sample rate 44.1kHz
	case "wav":
		args = append(args, "-acodec", "pcm_s16le") // PCM 16-bit little-endian
		args = append(args, "-ar", "44100")         // Sample rate 44.1kHz
		args = append(args, "-ac", "1")             // Mono for smaller size
	}

	args = append(args, outputFile.Name()) // Output file

	// Run FFmpeg
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg conversion failed: %w, stderr: %s", err, stderr.String())
	}

	// Read the converted output
	convertedData, err := os.ReadFile(outputFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read converted audio: %w", err)
	}

	return convertedData, nil
}

// isValidAudioType checks if the content type is a supported audio format
func (p *AudioProcessor) isValidAudioType(contentType string) bool {
	validTypes := []string{
		"audio/",                   // Any audio type
		"application/octet-stream", // Generic binary that might be audio
	}

	for _, validType := range validTypes {
		if strings.HasPrefix(contentType, validType) {
			return true
		}
	}

	return false
}

// detectAudioFormat detects audio format from the first few bytes (magic numbers)
func (p *AudioProcessor) detectAudioFormat(data []byte) (string, bool) {
	if len(data) < 12 {
		return "", false
	}

	// WAV: RIFF....WAVE
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x41 && data[10] == 0x56 && data[11] == 0x45 {
		return "audio/wav", true
	}

	// MP3: ID3 tag or MPEG sync
	if len(data) >= 3 {
		// ID3v2 tag
		if data[0] == 0x49 && data[1] == 0x44 && data[2] == 0x33 {
			return "audio/mp3", true
		}
		// MPEG sync (11 bits set to 1)
		if data[0] == 0xFF && (data[1]&0xE0) == 0xE0 {
			return "audio/mp3", true
		}
	}

	// FLAC: fLaC
	if len(data) >= 4 && data[0] == 0x66 && data[1] == 0x4C && data[2] == 0x61 && data[3] == 0x43 {
		return "audio/flac", true
	}

	// OGG: OggS
	if len(data) >= 4 && data[0] == 0x4F && data[1] == 0x67 && data[2] == 0x67 && data[3] == 0x53 {
		return "audio/ogg", true
	}

	// M4A/AAC: ....ftyp
	if len(data) >= 8 && data[4] == 0x66 && data[5] == 0x74 && data[6] == 0x79 && data[7] == 0x70 {
		return "audio/mp4", true
	}

	return "", false
}

// generateAudioFailureMessage creates a user-friendly error message for audio processing failures
func (p *AudioProcessor) generateAudioFailureMessage(err error, audioPosition, totalAudios int, hasMixedScenario bool) string {
	errorMsg := err.Error()
	var baseMessage string
	var contextPrefix string

	// Create context prefix for mixed scenarios
	if hasMixedScenario && totalAudios > 1 {
		contextPrefix = fmt.Sprintf("Audio %d of %d could not be processed. ", audioPosition, totalAudios)
	} else if totalAudios > 1 {
		contextPrefix = fmt.Sprintf("One of the %d audio files provided could not be processed. ", totalAudios)
	} else {
		contextPrefix = "The audio file provided could not be processed. "
	}

	// Determine specific error message based on error type
	if strings.Contains(errorMsg, "no such host") || strings.Contains(errorMsg, "dial tcp") {
		baseMessage = "I couldn't access the audio file due to network connectivity issues. The server appears to be unreachable or the domain doesn't exist. Please verify the URL or provide an alternative audio file."
	} else if strings.Contains(errorMsg, "status 401") || strings.Contains(errorMsg, "status 403") {
		baseMessage = "The audio file requires authentication or access permissions that weren't provided. Please provide proper authentication headers or use a publicly accessible audio file."
	} else if strings.Contains(errorMsg, "status 404") {
		baseMessage = "The audio URL appears to be broken or the file has been moved/deleted (404 Not Found). Please provide a valid audio URL."
	} else if strings.Contains(errorMsg, "size exceeds limit") {
		baseMessage = "The audio file is too large to process (exceeds 25MB limit). Please provide a smaller audio file or compress it before sharing."
	} else if strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "deadline exceeded") {
		baseMessage = "The audio file took too long to download. Please try again later or provide an alternative audio file."
	} else if strings.Contains(errorMsg, "ffmpeg") {
		baseMessage = "The audio file couldn't be converted to a supported format. Please provide the audio in a different format (MP3, WAV, FLAC, OGG, etc.)."
	} else {
		baseMessage = "There was a technical issue processing this audio file. Please try providing the audio again or use an alternative file."
	}

	// Add guidance for mixed scenarios
	var mixedScenarioGuidance string
	if hasMixedScenario && totalAudios > 1 {
		mixedScenarioGuidance = " You can still analyze and respond to the other content that was successfully processed."
	}

	return fmt.Sprintf("%s%s%s", contextPrefix, baseMessage, mixedScenarioGuidance)
}
