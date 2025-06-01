package main

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"
)

const mcpBinary = "/Users/fabien/SideProjects/fabian-mcp-server/fabian-mcp-server"

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Request received:", r.Method, r.URL.Path)
		callMcpTool()
		w.WriteHeader(http.StatusOK)
	})

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	fmt.Println("Run server on port 3000")
	http.ListenAndServe((":3000"), nil)
}

func initializeMcpServer(mcpServer *exec.Cmd) (io.WriteCloser, io.ReadCloser, error) {
	stdin, err := mcpServer.StdinPipe()
	if err != nil {
		fmt.Println("Error creating stdin pipe:", err)
		return nil, nil, err
	}

	stdout, err := mcpServer.StdoutPipe()
	if err != nil {
		fmt.Println("Error creating stdout pipe:", err)
		return nil, nil, err
	}

	if err := mcpServer.Start(); err != nil {
		fmt.Println("Error starting MCP tool:", err)
		return nil, nil, err
	}

	return stdin, stdout, nil
}

func callMcpTool() {
	fmt.Println("Calling MCP tool...")

	mcpServer := exec.Command(mcpBinary)
	stdin, stdout, err := initializeMcpServer(mcpServer)
	if err != nil {
		fmt.Println("Failed to initialize MCP server:", err)
		return
	}
	defer stdin.Close()
	defer stdout.Close()
	defer mcpServer.Process.Kill()

	buf := make([]byte, 1024)
	timeout := time.After(10 * time.Second)

	payload := `{ "jsonrpc": "2.0", "method": "tools/call", "params": { "name": "hello_world", "arguments": { "name": "Fabien"} }, "id": 1}` + "\n"
	_, err = io.WriteString(stdin, payload)
	if err != nil {
		fmt.Println("Error writing to stdin:", err)
		return
	}
	stdin.Close()

	for {
		select {
		case <-timeout:
			fmt.Println("Timeout: reading from MCP tool took more than 30 seconds")
			return
		default:
			n, err := stdout.Read(buf)
			if n > 0 {
				fmt.Print(string(buf[:n]))
			}
			if err != nil {
				if err == io.EOF {
					return
				}
				fmt.Println("Error reading stdout:", err)
				return
			}
		}
	}
}
