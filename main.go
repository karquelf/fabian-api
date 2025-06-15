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
	http.HandleFunc("/", handleChatRequest)

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	fmt.Println("Run server on port 3000")
	http.ListenAndServe((":3000"), nil)
}

func handleChatRequest(w http.ResponseWriter, r *http.Request) {
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
