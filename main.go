package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const mcpBinary = "/Users/fabien/SideProjects/fabian-mcp-server/fabian-mcp-server"

type ChatPayload struct {
	Message string
	Time    int64
}

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

var clientInstance *openai.Client

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

	listOfToolsPayload := `{ "jsonrpc": "2.0", "method": "tools/list", "params": {}, "id": -1 }`
	toolsResponse := callMcpTool(listOfToolsPayload)

	llmResponse := callLLM(payload.Message, toolsResponse)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(llmResponse))
}

func LLMClient() *openai.Client {
	if clientInstance == nil {
		client := openai.NewClient(
			option.WithBaseURL("http://localhost:1234/v1"),
		)
		clientInstance = &client
	}
	return clientInstance
}

func extractTools(content string) []openai.ChatCompletionToolParam {
	mcpResponse := ToolsMcpResponse{}

	if err := json.Unmarshal([]byte(content), &mcpResponse); err != nil {
		fmt.Println("Error parsing MCP response:", err)
		return []openai.ChatCompletionToolParam{}
	}

	var tools []openai.ChatCompletionToolParam
	for _, tool := range mcpResponse.Result.Tools {
		required := tool.InputSchema.Required
		if required == nil {
			required = []string{}
		}

		openaiTool := openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        tool.Name,
				Description: openai.String(tool.Description),
				Parameters: openai.FunctionParameters{
					"type":       tool.InputSchema.Type,
					"properties": tool.InputSchema.Properties,
					"required":   required,
				},
			},
		}
		tools = append(tools, openaiTool)
	}

	return tools
}

func callLLM(message string, tools string) string {
	models, err := LLMClient().Models.List(context.TODO())

	if err != nil {
		panic(err.Error())
	}

	context := context.TODO()
	chat := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(message),
		},
		Model: string(models.Data[0].ID),
		Tools: extractTools(tools),
	}

	chatCompletion, err := LLMClient().Chat.Completions.New(context, chat)

	if err != nil {
		panic(err.Error())
	}

	chat.Messages = append(chat.Messages, chatCompletion.Choices[0].Message.ToParam())
	for _, toolCall := range chatCompletion.Choices[0].Message.ToolCalls {
		if strings.HasPrefix(toolCall.Function.Name, "fmcp-") {
			toolPayload := fmt.Sprintf(`{ "jsonrpc": "2.0", "method": "tools/call", "params": { "name": "%s", "arguments": %s }, "id": 1}`,
				toolCall.Function.Name,
				toolCall.Function.Arguments)

			toolResponse := callMcpTool(toolPayload)
			chat.Messages = append(chat.Messages, openai.ToolMessage(toolResponse, toolCall.ID))
		}
	}

	chatCompletion, err = LLMClient().Chat.Completions.New(context, chat)

	if err != nil {
		panic(err.Error())
	}

	return cleanLLMResponse(chatCompletion.Choices[0].Message.Content)
}

func cleanLLMResponse(content string) string {
	// Remove <think>...</think> tags and their content
	for {
		start := strings.Index(content, "<think>")
		if start == -1 {
			break
		}
		end := strings.Index(content[start:], "</think>")
		if end == -1 {
			break
		}
		end += start + len("</think>")
		content = content[:start] + content[end:]
	}
	return strings.TrimSpace(content)
}
