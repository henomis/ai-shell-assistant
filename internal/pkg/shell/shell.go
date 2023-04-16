package shell

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/commander-cli/cmd"
	"github.com/eiannone/keyboard"
	"github.com/fatih/color"

	"github.com/henomis/ai-shell-assistant/internal/pkg/completion"
)

type Shell struct {
	completion       *completion.Completion
	shellInterpreter string
}

type ShellResponse struct {
	CommandAction CommandAction
	Command       string
}

type CommandAction string

const (
	CommandActionExecute CommandAction = "execute"
	CommandActionRevise  CommandAction = "retry"
	CommandActionExit    CommandAction = "exit"
)

var (
	yellow = color.New(color.FgYellow).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	white  = color.New(color.FgWhite).SprintFunc()
)

func New(completion *completion.Completion, shellInterpreter string) *Shell {
	return &Shell{
		completion:       completion,
		shellInterpreter: shellInterpreter,
	}
}

func (s *Shell) Suggest(prompt string) (*ShellResponse, error) {

	if prompt == "" {
		prompt = getUserPromptFromStdin()
	}

	return s.handleSuggestion(prompt)
}

func (s *Shell) Execute(script string) error {

	file, err := ioutil.TempFile("", "ai-go-shell")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	file.WriteString(script)

	command := fmt.Sprintf("%s %s", s.shellInterpreter, file.Name())
	c := cmd.NewCommand(command, cmd.WithStandardStreams, cmd.WithInheritedEnvironment(cmd.EnvVars{}))

	err = c.Execute()
	if err != nil {
		return fmt.Errorf("command: %w", err)
	}

	return nil
}

func (s *Shell) handleSuggestion(prompt string) (*ShellResponse, error) {
	response, err := s.completion.Suggest(prompt)
	if err != nil {
		return nil, fmt.Errorf("completion: %w", err)
	}

	printCommandLineSuggestionFromResponse(response)

	for _, executable := range response.Executables {
		_, err := exec.LookPath(executable)
		if err != nil {
			return nil, err
		}
	}

	userAction, err := getUserActionFromStdin()
	if err != nil {
		return nil, err
	}

	return &ShellResponse{
		Command:       response.Command,
		CommandAction: newCommandActionFromUserAction(userAction),
	}, nil

}

// ---------------
// support methods
// ---------------

func printCommandLineSuggestionFromResponse(response *completion.CompletionResponse) {

	color.NoColor = false
	color.New(color.FgWhite, color.Bold).Printf("\n🤖 Here is your script:\n\n")
	color.New(color.FgCyan, color.Bold).Printf("%s\n", response.Command)
	color.New(color.FgWhite, color.Italic).Printf("--\n")
	color.New(color.FgYellow, color.Italic).Printf("This script uses: %v\n", response.Executables)
	color.New(color.FgWhite, color.Italic).Printf("%s\n\n", response.Explain)
	color.NoColor = true
}

func getUserPromptFromStdin() string {

	color.NoColor = false
	color.New(color.FgWhite, color.Bold).Printf("\n🤖 How may I help you? > ")
	color.NoColor = true

	reader := bufio.NewReader(os.Stdin)
	userInput, _ := reader.ReadString('\n')

	return userInput
}

func getUserActionFromStdin() (string, error) {

	if err := keyboard.Open(); err != nil {
		return "", err
	}
	defer func() {
		_ = keyboard.Close()
	}()

	color.NoColor = false
	fmt.Printf("%s%s%s%s%s",
		white("["), green("E"), white("]xecute, ["), red("Q"), white("]uit> "),
	)
	color.NoColor = true

	userAction, _, err := keyboard.GetSingleKey()
	if err != nil {
		return "", err
	}

	return string(userAction), nil
}

func newCommandActionFromUserAction(userAction string) CommandAction {
	switch strings.ToLower(userAction) {
	case "e":
		return CommandActionExecute
	case "r":
		return CommandActionRevise
	case "q":
		return CommandActionExit
	default:
		return CommandActionExit
	}
}
