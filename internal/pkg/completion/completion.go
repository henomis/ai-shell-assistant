package completion

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/sashabaranov/go-openai"
)

const (
	systemPromptTemplate = `You are a professional {{ .shell }} shell script developer.
	I will ask you to create a {{ .shell }} script for the operating system {{ .os }} that can be execute in a terminal. 
	You must reply using the following data format:

	--script--
	Place here the {{ .shell }} script, you must use the correct syntax for the {{ .shell }} shell, avoiding installing additional software.
	--end-script--
	--explain--
	Place here the description of the script, You must describe succintly, use as few words as possible, do not be verbose and use plain text. If there are multiple steps, please display them as bullet points
	--end-explain--
	--commands-list--
	Place here a list comma separated, single line of commands that are used in the script
	--end-commands-list--

	That is the data format. Do not add any additional text to your response than the required data.`
)

var outputParserRegex = regexp.MustCompile(`(?s)--script--(.*?)--end-script--.*--explain--(.*?)--end-explain--.*--commands-list--(.*?)--end-commands-list--`)

type Completion struct {
	openAIClient    *openai.Client
	operatingSystem string
	shellName       string
}

type CompletionResponse struct {
	Script      string
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

	matches := outputParserRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no matches found: %s", content)
	}

	if len(matches[0]) != 4 {
		return nil, fmt.Errorf("invalid number of matches")
	}

	var completionResponse CompletionResponse

	var executables []string
	for _, executable := range strings.Split(strings.TrimSpace(matches[0][3]), ",") {
		executables = append(executables, strings.TrimSpace(executable))
	}

	completionResponse.Script = strings.TrimSpace(matches[0][1])
	completionResponse.Explain = strings.TrimSpace(matches[0][2])
	completionResponse.Executables = executables

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
