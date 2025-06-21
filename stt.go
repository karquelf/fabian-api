package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

const whisperCli = "/Users/fabien/SideProjects/fabian-stt/whisper.cpp/build/bin/whisper-cli"
const whisperModel = "/Users/fabien/SideProjects/fabian-stt/whisper.cpp/models/ggml-small.bin"

func transcribeAudio(file io.Reader) (string, error) {
	tempFile, err := os.CreateTemp("", "audio-*.wav")
	if err != nil {
		return "", fmt.Errorf("error creating temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		return "", fmt.Errorf("error saving audio file: %w", err)
	}

	cmd := exec.Command(whisperCli,
		"-m", whisperModel,
		"-f", tempFile.Name(),
		"--no-prints",
		"--no-timestamps")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running whisper: %w", err)
	}

	return string(output), nil
}
