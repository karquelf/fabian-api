package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

var clientInstance *openai.Client

func LLMClient() *openai.Client {
	if clientInstance == nil {
		client := openai.NewClient(
			option.WithBaseURL("http://localhost:1234/v1"),
		)
		clientInstance = &client
	}
	return clientInstance
}

func callLLM(message string) string {
	listOfToolsPayload := `{ "jsonrpc": "2.0", "method": "tools/list", "params": {}, "id": -1 }`
	tools := callMcpTool(listOfToolsPayload)

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
