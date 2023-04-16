package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	openai "github.com/sashabaranov/go-openai"

	"github.com/henomis/ai-shell-assistant/internal/pkg/completion"
	"github.com/henomis/ai-shell-assistant/internal/pkg/shell"
)

var (
	ErrorShellAI = fmt.Errorf("🚨 OOPS")
)

func main() {

	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		fmt.Printf("%s: OPEN_AI_KEY is not set. Please set the OPENAI_API_KEY environment variable to your OpenAI API key\n", ErrorShellAI)
		return
	}

	operatingSystem := runtime.GOOS
	if len(operatingSystem) == 0 {
		fmt.Printf("%s: unable to determine OS\n", ErrorShellAI)
		return
	}

	shellName := os.Getenv("SHELL")
	if len(shellName) == 0 {
		fmt.Printf("%s: unable to determine shell\n", ErrorShellAI)
		return
	}

	shellInterpreter, err := exec.LookPath(shellName)
	if err != nil {
		fmt.Printf("%s: unable to find %s interpreter\n", shellName, ErrorShellAI)
		return
	}

	userInput := strings.Join(os.Args[1:], " ")

	openAIClient := openai.NewClient(openAIKey)
	completionInstance := completion.New(openAIClient, operatingSystem, shellName)
	shellInstance := shell.New(completionInstance, shellInterpreter)

	for {
		shellResponse, err := shellInstance.Suggest(userInput)
		if err != nil {
			fmt.Printf("%s: %s\n", ErrorShellAI, err)
			continue
		}
		if shellResponse.CommandAction == shell.CommandActionExecute {
			err = shellInstance.Execute(shellResponse.Command)
			if err != nil {
				fmt.Printf("%s: %s\n", ErrorShellAI, err)
				continue
			}
		} else if shellResponse.CommandAction == shell.CommandActionSkip {
			continue
		}
	}
}
