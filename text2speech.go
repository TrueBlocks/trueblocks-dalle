package dalle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrueBlocks/trueblocks-art/packages/ai"
	"github.com/TrueBlocks/trueblocks-art/packages/creds"
	logger "github.com/TrueBlocks/trueblocks-dalle/v6/pkg/logging"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/storage"
)

// TextToSpeech converts the given text to speech using OpenAI's audio API and writes it to the provided output directory.
// It returns the full path of the written mp3 file. If the OPENAI_API_KEY is missing, it returns an empty string and no error.
func TextToSpeech(text string, voice string, series string, address string) (string, error) {
	series = strings.ReplaceAll(strings.ToLower(series), " DO NOT PUT TEXT IN THE IMAGE. ", "")
	address = strings.ToLower(address)
	if text == "" {
		return "", errors.New("empty text")
	}
	apiKey, keyErr := creds.Get("OPENAI_API_KEY")
	if keyErr != nil { // silently skip when no key
		logger.Info("speech.skip_no_api_key", "series", series, "addr", address)
		return "", nil
	}
	baseDir := filepath.Join(storage.OutputDir(), series, "audio")
	_ = os.MkdirAll(baseDir, 0o750)

	provider := &ai.OpenAI{APIKey: apiKey}
	audio, err := provider.Speak(context.Background(), text, ai.SpeechOptions{Voice: voice})
	if err != nil {
		logger.InfoR("speech.error", "series", series, "addr", address, "error", err.Error())
		return "", err
	}

	name := "speech.mp3"
	if address != "" {
		name = address + ".mp3"
	}
	outPath := filepath.Join(baseDir, name)
	cleanOut := filepath.Clean(outPath)
	if !strings.HasPrefix(cleanOut, filepath.Clean(baseDir)+string(os.PathSeparator)) {
		return "", errors.New("invalid audio output path")
	}
	if err := os.WriteFile(cleanOut, audio, 0o600); err != nil { // #nosec G304 path validated
		return "", err
	}
	logger.InfoG("speech.write", "series", series, "addr", address, "path", outPath)
	return outPath, nil
}

// GenerateSpeech ensures a text-to-speech mp3 exists for the enhanced prompt of the given address.
// It returns the path to the mp3. If already generated it returns existing path.
func GenerateSpeech(series, address string, lockTTL time.Duration) (string, error) {
	start := time.Now()
	logger.Info("speech.build.start", "series", series, "addr", address)
	if address == "" {
		return "", errors.New("address required")
	}
	cleanupLocks()
	if lockTTL <= 0 {
		lockTTL = 2 * time.Minute
	}
	key := "speech:" + series + ":" + address
	audioPath := filepath.Join(storage.OutputDir(), series, "audio", address+".mp3")
	if fileExists(audioPath) { // fast path
		return audioPath, nil
	}
	if !acquireLock(key, lockTTL) { // another generation in progress
		return audioPath, nil
	}
	defer releaseLock(key)
	mc, err := getContext(series)
	if err != nil {
		return "", err
	}
	dd, err := mc.ctx.MakeDalleDress(address)
	if err != nil {
		return "", err
	}
	text := dd.EnhancedPrompt
	if text == "" {
		text = dd.Prompt
	}
	if text == "" {
		return "", errors.New("no prompt text to speak")
	}
	out, err := TextToSpeech(text, "alloy", series, address)
	if err != nil {
		return "", err
	}
	logger.InfoG("speech.build.end", "series", series, "addr", address, "durMs", time.Since(start).Milliseconds())
	return out, nil
}

// Speak plays (or generates then plays) the speech mp3 for the given series/address.
// Returns the path to the mp3 (even if play fails). Generation is skipped if file exists.
func Speak(series, address string) (string, error) {
	if address == "" {
		return "", errors.New("address required")
	}
	audioPath := filepath.Join(storage.OutputDir(), series, "audio", address+".mp3")
	if !fileExists(audioPath) {
		// Generate (ignore lock TTL customization here; use default 0 which GenerateSpeech adjusts)
		p, err := GenerateSpeech(series, address, 0)
		if err != nil {
			return "", err
		}
		audioPath = p
	}
	if audioPath == "" { // nothing generated (likely no API key)
		return "", nil
	}
	return audioPath, nil
}

// ReadToMe ensures the mp3 exists (generating if necessary) and always attempts playback.
// It returns the path if the file exists or was generated, else empty string.
func ReadToMe(series, address string) (string, error) {
	if address == "" {
		return "", errors.New("address required")
	}
	audioPath := filepath.Join(storage.OutputDir(), series, "audio", address+".mp3")
	if !fileExists(audioPath) {
		p, err := GenerateSpeech(series, address, 0)
		if err != nil {
			return "", err
		}
		audioPath = p
	}
	if audioPath == "" {
		return "", nil
	}
	return audioPath, nil
}

// AudioURL resolves a relative URL (served by internal file server) for an existing or newly generated mp3.
// Returns empty string if generation not possible or file server base URL unknown.
func AudioURL(baseURL, series, address string) (string, error) {
	if address == "" || baseURL == "" {
		return "", nil
	}
	p, err := Speak(series, address)
	if err != nil || p == "" {
		return "", err
	}
	// Derive relative path segment after output dir root
	rel := strings.TrimPrefix(p, storage.OutputDir()+"/")
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return "", nil
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return baseURL + rel, nil
}
