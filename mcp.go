package main

import (
	"fmt"
	"io"
	"os/exec"
	"time"
)

const mcpBinary = "/Users/fabien/SideProjects/fabian-mcp-server/fabian-mcp-server"

type ToolsMcpResponse struct {
	Result struct {
		Tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			InputSchema struct {
				Type       string                 `json:"type"`
				Properties map[string]interface{} `json:"properties"`
				Required   []string               `json:"required"`
			} `json:"inputSchema"`
		} `json:"tools"`
	} `json:"result"`
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

func callMcpTool(payload string) string {
	fmt.Printf("Calling MCP tool with : %s\n", payload)

	mcpServer := exec.Command(mcpBinary)
	stdin, stdout, err := initializeMcpServer(mcpServer)
	if err != nil {
		fmt.Println("Failed to initialize MCP server:", err)
		return ""
	}
	defer stdin.Close()
	defer stdout.Close()
	defer mcpServer.Process.Kill()

	buf := make([]byte, 1024)
	timeout := time.After(10 * time.Second)
	var response string

	_, err = io.WriteString(stdin, payload+"\n")
	if err != nil {
		fmt.Println("Error writing to stdin:", err)
		return ""
	}
	stdin.Close()

	for {
		select {
		case <-timeout:
			fmt.Println("Timeout: reading from MCP tool took more than 30 seconds")
			return response
		default:
			n, err := stdout.Read(buf)
			if n > 0 {
				response += string(buf[:n])
			}
			if err != nil {
				if err == io.EOF {
					return response
				}
				fmt.Println("Error reading stdout:", err)
				return response
			}
		}
	}
}
