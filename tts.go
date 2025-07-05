package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/pemistahl/lingua-go"
)

func tts(message string) (string, error) {
	uid := generateUID()
	outputFile := "tmp/out_" + uid + ".wav"

	detectedLanguage := detectLanguage(message)

	payload := map[string]string{
		"language": detectedLanguage,
		"text":     message,
		"uid":      uid,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling payload: %v", err)
	}

	resp, err := http.Post("http://localhost:5555/", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("error making request to TTS server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TTS server returned status code: %d", resp.StatusCode)
	}

	outFile, err := os.Create(outputFile)
	if err != nil {
		return "", fmt.Errorf("error creating output file: %v", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("error writing audio data to file: %v", err)
	}

	return outputFile, nil
}

func generateUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func detectLanguage(text string) string {
	languages := []lingua.Language{
		lingua.English,
		lingua.French,
	}

	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(languages...).
		Build()

	if language, exists := detector.DetectLanguageOf(text); exists {
		if language == lingua.French {
			return "fr"
		}
	}

	return "en"
}
