package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ChatPayload struct {
	Message string
	Time    int64
}

func main() {
	http.HandleFunc("/text", handleTextRequest)
	http.HandleFunc("/voice", handleVoiceRequest)

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	fmt.Println("Run server on port 3000")
	http.ListenAndServe((":3000"), nil)
}

func handleTextRequest(w http.ResponseWriter, r *http.Request) {
	var payload ChatPayload
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Error reading request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(bodyBytes, &payload)
	if err != nil {
		fmt.Println("Error Unmarshal:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	llmResponse := callLLM(payload.Message)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(llmResponse))
}

func handleVoiceRequest(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form to extract audio file
	err := r.ParseMultipartForm(32 << 20) // 32MB max memory
	if err != nil {
		http.Error(w, "Error parsing multipart form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "Error retrieving audio file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	message, err := transcribeAudio(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error transcribing audio: %v", err), http.StatusInternalServerError)
		return
	}

	llmResponse := callLLM(message)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(llmResponse))
}
