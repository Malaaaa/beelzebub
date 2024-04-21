package plugins

import (
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildPromptEmptyHistoryClaude(t *testing.T) {
	//Given
	terminal := Initialize([]History{}, "valid_api_key") // Use Initialize to properly set up the VirtualTerminal instance
	command := "pwd"

	//When
	prompt := terminal.BuildPrompt(command) // Ensure you are calling the BuildPrompt method on a VirtualTerminal instance

	//Then
	expectedPrompt := "You will act as an Ubuntu Linux terminal. User commands and expected terminal outputs are provided. Your responses must be contained within a single code block, reflecting the terminal's behavior without additional explanations unless explicitly requested.\n\nA:pwd\n\nQ:/home/user\n\nA:cat hello.txt\n\nQ:world\n\nA:echo 1234\n\nQ:1234\n\nA:pwd\n\nQ:A:pwd\n\nQ:"
	assert.Equal(t, expectedPrompt, prompt)
}

func TestBuildPromptWithHistoryClaude(t *testing.T) {
	//Given
	terminal := Initialize([]History{
		{Input: "cat hello.txt", Output: "world"},
		{Input: "echo 1234", Output: "1234"},
	}, "valid_api_key") // Setup terminal with a predefined history
	command := "pwd"

	//When
	prompt := terminal.BuildPrompt(command)

	//Then
	expectedPrompt := "You will act as an Ubuntu Linux terminal. User commands and expected terminal outputs are provided. Your responses must be contained within a single code block, reflecting the terminal's behavior without additional explanations unless explicitly requested.\n\nA:pwd\n\nQ:/home/user\n\nA:cat hello.txt\n\nQ:world\n\nA:echo 1234\n\nQ:1234\n\nA:pwd\n\nQ:A:cat hello.txt\n\nQ:world\n\nA:echo 1234\n\nQ:1234\n\nA:pwd\n\nQ:"
	assert.Equal(t, expectedPrompt, prompt)
}

func TestGetCompletionsFailValidation(t *testing.T) {
	terminal := Initialize([]History{}, "") // Initialize with an empty API key to simulate validation failure

	//When
	_, err := terminal.GetCompletions("test")

	//Then
	assert.EqualError(t, err, "API key is missing")
}

func TestGetCompletionsWithResults(t *testing.T) {
	client := resty.New()
	httpmock.ActivateNonDefault(client.GetClient())
	defer httpmock.DeactivateAndReset()

	// Mock setup, ensuring it matches exactly what GetCompletions calls
	httpmock.RegisterResponder("POST", apiEndpoint,
		httpmock.NewJsonResponderOrPanic(200, &APIResponse{
			Choices: []ResponseChoice{{Text: "prova.txt"}},
		}))

	terminal := Initialize([]History{}, "valid_api_key")
	terminal.Client = client

	// Execute
	result, err := terminal.GetCompletions("ls")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "prova.txt", result)
}

func TestGetCompletionsWithoutResults(t *testing.T) {
	client := resty.New()
	httpmock.ActivateNonDefault(client.GetClient())
	defer httpmock.DeactivateAndReset()

	// Mock setup
	httpmock.RegisterResponder("POST", apiEndpoint, // Consistent with the actual API endpoint
		httpmock.NewJsonResponderOrPanic(200, APIResponse{
			Choices: []ResponseChoice{},
		}))

	terminal := Initialize([]History{}, "valid_api_key")
	terminal.Client = client

	//When
	_, err := terminal.GetCompletions("ls")

	//Then
	assert.EqualError(t, err, "no completion choices returned by the API")
}
