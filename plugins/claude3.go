package plugins

// Ensure all imports are correctly placed and necessary packages are imported.
import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"strings"
)

// Define constants and structures.
const (
	promptTemplate   = "You will act as an Ubuntu Linux terminal. User commands and expected terminal outputs are provided. Your responses must be contained within a single code block, reflecting the terminal's behavior without additional explanations unless explicitly requested.\n\nA:pwd\n\nQ:/home/user\n\nA:cat hello.txt\n\nQ:world\n\nA:echo 1234\n\nQ:1234\n\nA:pwd\n\nQ:"
	ClaudePluginName = "Claude3LinuxTerminal"
	apiEndpoint      = "https://api.anthropic.com/v1/messages" // Updated API endpoint
)

type Claude3LinuxTerminal struct {
	Histories []History
	APIKey    string
	Client    *resty.Client
}

type ClaudeResponse struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Role         string        `json:"role"`
	Model        string        `json:"model"`
	StopSequence interface{}   `json:"stop_sequence"`
	Usage        Usage         `json:"usage"`
	Content      []ContentItem `json:"content"`
	StopReason   string        `json:"stop_reason"`
}

// Usage nested structure
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ContentItem in the content array
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type APIRequest struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	Messages    []Message `json:"messages"`
	System      string    `json:"system"`
}

func Initialize(histories []History, apiKey string) *Claude3LinuxTerminal {
	return &Claude3LinuxTerminal{
		Histories: histories,
		APIKey:    apiKey,
		Client:    resty.New(),
	}
}

func (vt *Claude3LinuxTerminal) BuildPrompt(command string) string {
	var sb strings.Builder
	sb.WriteString(promptTemplate)
	for _, h := range vt.Histories {
		sb.WriteString(fmt.Sprintf("A:%s\n\nQ:%s\n\n", h.Input, h.Output))
	}
	sb.WriteString(fmt.Sprintf("A:%s\n\nQ:", command))
	return sb.String()
}

func (vt *Claude3LinuxTerminal) BuildMessages(command string) []Message {
	return []Message{
		{
			Role: "user",
			Content: []Content{
				{
					Type: "text",
					Text: vt.BuildPrompt(command),
				},
			},
		},
	}
}

func (vt *Claude3LinuxTerminal) GetCompletions(command string) (string, error) {
	if vt.APIKey == "" {
		return "", errors.New("API key is missing")
	}

	reqData := APIRequest{
		Model:       "claude-3-opus-20240229",
		MaxTokens:   1024,
		System:      "You will act as an Ubuntu Linux terminal. The user will type commands, and you are to reply with what the terminal should show. Your responses must be contained within a single code block. Do not provide explanations or type commands unless explicitly instructed by the user. Remember previous commands and consider their effects on subsequent outputs.\\n\\nA:pwd\\n\\nQ:/home/user\\n\\nA:pwd\\n\\nQ:",
		Temperature: 0,
		Messages:    vt.BuildMessages(command),
	}

	reqJSON, err := json.Marshal(reqData)
	if err != nil {
		return "", fmt.Errorf("error marshalling request: %v", err)
	}

	response, err := vt.Client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("x-api-key", vt.APIKey).
		SetHeader("anthropic-version", "2023-06-01").
		SetBody(reqJSON).
		SetResult(new(ClaudeResponse)). // Ensure you're creating a new instance of APIResponse
		Post(apiEndpoint)

	if err != nil {
		return "", err
	}
	log.Debug(response)
	if len(response.Result().(*ClaudeResponse).Content) == 0 {
		return "", errors.New("no choices")
	}

	return strings.Replace(response.Result().(*ClaudeResponse).Content[0].Text, "```", "", -1), nil

}
