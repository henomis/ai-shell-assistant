package completion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/sashabaranov/go-openai"
)

const (
	systemPromptTemplate = `You are a professional script developer.
	I will ask you to create a {{ .shell }} script for the operating system {{ .os }} that one can execute in a terminal.
	You must reply using the following json format:
	{
		"command": "the {{ .shell }} script content as unique json escaped line. It should be able to be directly run in a terminal. Do not include any other text.",
		"executables": ["list of executables that are used in the script as json array of strings"],
		"explain": "description of the {{ .shell }} script as json escaped line. You must describe succintly, use as few words as possible, do not be verbose. If there are multiple steps, please display them as bullet points."
	}`
)

type Completion struct {
	openAIClient    *openai.Client
	operatingSystem string
	shellName       string
}

type CompletionResponse struct {
	Command     string
	Explain     string
	Executables []string
}

func New(openAIClient *openai.Client, operatingSystem, shellName string) *Completion {
	return &Completion{
		openAIClient:    openAIClient,
		operatingSystem: operatingSystem,
		shellName:       shellName,
	}
}

func (c *Completion) Suggest(prompt string) (*CompletionResponse, error) {
	if prompt == "" {
		return nil, fmt.Errorf("input is empty")
	}

	response, err := c.openAIClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: c.buildSystemPrompt(),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("openai: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	content := response.Choices[0].Message.Content

	var completionResponse CompletionResponse

	err = json.Unmarshal([]byte(content), &completionResponse)
	if err != nil {
		return nil, fmt.Errorf("json: %w\n %s", err, content)
	}

	return &completionResponse, nil

}

func (c *Completion) buildSystemPrompt() string {
	var output bytes.Buffer

	templ := template.Must(template.New("prompt").Parse(systemPromptTemplate))
	err := templ.Execute(&output, map[string]interface{}{
		"os":    c.operatingSystem,
		"shell": c.shellName,
	})
	if err != nil {
		panic(err)
	}

	return removeInitialSpaces(output.String())
}

// ---------------
// support methods
// ---------------

func removeInitialSpaces(input string) string {
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimLeft(line, " ")
		lines[i] = strings.TrimLeft(lines[i], "\t")
	}
	return strings.Join(lines, "\n")
}
