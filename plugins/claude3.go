package plugins

// Ensure all imports are correctly placed and necessary packages are imported.
import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"strings"
)

// Define constants and structures.
const (
	promptTemplate   = "You will act as an Ubuntu Linux terminal. User commands and expected terminal outputs are provided. Your responses must be contained within a single code block, reflecting the terminal's behavior without additional explanations unless explicitly requested.\n\nA:pwd\n\nQ:/home/user\n\nA:cat hello.txt\n\nQ:world\n\nA:echo 1234\n\nQ:1234\n\nA:pwd\n\nQ:"
	ClaudePluginName = "Claude3LinuxTerminal"
	apiEndpoint      = "https://api.anthropic.com/v1/completions"
)

type VirtualTerminal struct {
	Histories []History
	APIKey    string
	Client    *resty.Client
}

type ResponseChoice struct {
	Text         string      `json:"text"`
	Logprobs     interface{} `json:"logprobs"`
	FinishReason string      `json:"finish_reason"`
}

type APIResponse struct {
	ID      string           `json:"id"`
	Choices []ResponseChoice `json:"choices"`
	Usage   struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

type APIRequest struct {
	Model            string   `json:"model"`
	Prompt           string   `json:"prompt"`
	Temperature      float64  `json:"temperature"`
	MaxTokens        int      `json:"max_tokens"`
	TopP             float64  `json:"top_p"`
	FrequencyPenalty float64  `json:"frequency_penalty"`
	PresencePenalty  float64  `json:"presence_penalty"`
	Stop             []string `json:"stop"`
}

func Initialize(histories []History, apiKey string) *VirtualTerminal {
	return &VirtualTerminal{
		Histories: histories,
		APIKey:    apiKey,
		Client:    resty.New(),
	}
}

func (vt *VirtualTerminal) BuildPrompt(command string) string {
	var sb strings.Builder
	sb.WriteString(promptTemplate)
	for _, h := range vt.Histories {
		sb.WriteString(fmt.Sprintf("A:%s\n\nQ:%s\n\n", h.Input, h.Output))
	}
	sb.WriteString(fmt.Sprintf("A:%s\n\nQ:", command))
	return sb.String()
}

func (vt *VirtualTerminal) GetCompletions(command string) (string, error) {
	if vt.APIKey == "" {
		return "", errors.New("API key is missing")
	}

	reqData := APIRequest{
		Model:            "claude-3-opus-20240229", // Ensure this matches your model
		Prompt:           vt.BuildPrompt(command),
		Temperature:      0,
		MaxTokens:        100,
		TopP:             1.0,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
		Stop:             []string{"\n"},
	}

	reqJSON, err := json.Marshal(reqData)
	if err != nil {
		return "", fmt.Errorf("error marshalling request: %v", err)
	}

	response, err := vt.Client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(reqJSON).
		SetAuthToken(vt.APIKey).
		SetResult(new(APIResponse)). // Ensure you're creating a new instance of APIResponse
		Post(apiEndpoint)

	if err != nil {
		return "", fmt.Errorf("error making API request: %v", err)
	}

	apiResponse, ok := response.Result().(*APIResponse) // Added type assertion with check
	if !ok || apiResponse == nil {
		return "", errors.New("invalid response format or nil response")
	}

	if len(apiResponse.Choices) == 0 {
		return "", errors.New("no completion choices returned by the API")
	}

	return apiResponse.Choices[0].Text, nil
}
