package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/mistakenelf/teacup/markdown"
	"github.com/sashabaranov/go-openai"
	uuid "github.com/satori/go.uuid"
)

type aiCommandGenerationModel struct {
	prompt   string
	markdown markdown.Model
	program  *tea.Program
}

func runCommandAiCommandGeneration(description string) {
	tryReadOpenAIKey()

	b := newAiCommandGeneration(description)
	p := tea.NewProgram(b, tea.WithAltScreen())
	go func() {
		for {
			<-time.After(100 * time.Millisecond)
			p.Send(timeMsg(time.Now()))
		}
	}()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

	if !isAiGenCompleted {
		return
	}

	// Save the markdown to a file in the config dir
	configPath, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("Failed to get config directory", "error", err)
	}
	markdownFilePath := filepath.Join(configPath, "commands-wiki", "ai", "ai-"+uuid.NewV4().String()+"-"+description+".md")
	err = os.MkdirAll(filepath.Dir(markdownFilePath), 0744)
	if err != nil {
		log.Fatal("Failed to create markdown file parent directories", "error", err)
	}

	markdownFile, err := os.Create(markdownFilePath)
	if err != nil {
		log.Fatal("Failed to create markdown file", "error", err)
	}
	_, err = markdownFile.WriteString(currentGptMarkdown)
	if err != nil {
		log.Fatal("Failed to write markdown content", "error", err)
	}
	err = markdownFile.Close()
	if err != nil {
		log.Fatal("Failed to close markdown file", "error", err)
	}

	// Add command to the index and save
	commands, err := readIndex()
	if err != nil {
		log.Fatal("Failed to read index", "error", err)
	}
	commands = append(commands, markdownToCommand(currentGptMarkdown))
	err = writeIndex(commands)
	if err != nil {
		log.Fatal("Failed to write index", "error", err)
	}

	cmd := markdownToCommand(currentGptMarkdown)
	showCommmand(cmd)
}

var currentDisplayMarkdown string
var currentGptMarkdown string
var openAiToken string

func tryReadOpenAIKey() {
	// Try to read the openai key
	openAiToken = os.Getenv("OPENAI_API_KEY")
	if openAiToken == "" {
		log.Fatal("OPENAI_API_KEY is not set")
	}
}

func newAiCommandGeneration(prompt string) aiCommandGenerationModel {
	return aiCommandGenerationModel{
		prompt: prompt,
	}
}

func (m aiCommandGenerationModel) Init() tea.Cmd {
	// Run the gpt ai
	go startCompletion(m)
	return nil
}

const markdownExample = `
### Create a dummy networking interface
This command creates a dummy network interface and assigns it an IP address.
` + "```bash" + `
ip link add <interface_name> type dummy &&
sudo ip addr add <cidr> brd + dev <interface_name> label <interface_name>:0
` + "```" + `
<!-- 
[interface_name]: <> (placeholder=vip0 validation="regex [a-z\d]+" desc="The name of the interface to create")
[cidr]: <> (placeholder="10.0.0.1/16" validation="regex ([0-9]{1,3}\.){3}[0-9]{1,3}(\/(([0-9]|[12][0-9]|3[0-2])))")
-->

1. ` + "`ip link add <interface_name> type dummy`" + `: Creates a new dummy network interface named ` + "`<interface_name>`" + `. Replace ` + "`<interface_name>`" + ` with the name you want for the interface, one example being ` + "`vip0`" + `.
2. ` + "`sudo ip addr add <cidr> brd + dev <interface_name> label <interface_name>:0`" + `: Assigns the cidr ` + "`<cidr>`" + ` to the interface ` + "`<interface_name>`" + `, the cidr should be in the format of ` + "`<ip>/<subnetmask>`" + `. Please replace the ` + "`<cidr>`" + ` with the one that fits your network. The ` + "`brd +`" + ` option sets the broadcast address to the default value. The ` + "`label <interface_name>:0`" + ` option assigns a label to the interface.
`

const cwcInstructions = `
The validation methods you have access to are:
- regex <regex pattern>
- file <regex pattern for mimetype>

Examples for these are:
- regex [a-z\d]+
- file image\/.+

The regex matches will wrap the regex in "^" and "$", so please keep that in mind.
`

var isAiGenCompleted = false

func startCompletion(m aiCommandGenerationModel) {
	c := openai.NewClient(openAiToken)
	ctx := context.Background()
	req := openai.ChatCompletionRequest{
		Model:     GetValueNoError("openai-model", "gpt-4-1106-preview"),
		MaxTokens: 1024,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "You will get a description for a command, you should then write a title, short description and a description of each argument for this command. Where applicable use variables like `<variable_name>` in the commands. The markdown should be formatted like below:\n" + markdownExample,
			},
			{
				Role:    "user",
				Content: m.prompt,
			},
		},
		Stream: true,
	}
	stream, err := c.CreateChatCompletionStream(ctx, req)
	if err != nil {
		fmt.Printf("CompletionStream error: %v\n", err)
		return
	}
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			render, err := markdown.RenderMarkdown(m.markdown.Viewport.Width, currentGptMarkdown)
			if err == nil {
				currentDisplayMarkdown = render
				isAiGenCompleted = true
			} else {
				log.Fatal("Failed to render markdown", "error", err)
			}
			return
		}

		if err != nil {
			log.Fatal("Stream error", "error", err)
			return
		}

		deltaContent := response.Choices[0].Delta.Content
		if deltaContent != "" {
			currentGptMarkdown += deltaContent + ""
			render, err := markdown.RenderMarkdown(m.markdown.Viewport.Width, currentGptMarkdown)
			if err == nil {
				currentDisplayMarkdown = render
			}
		}
	}
}

type timeMsg time.Time

func (m aiCommandGenerationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Set the size to a bit smaller then we need
		cmd := m.markdown.SetSize(msg.Width-2, msg.Height-6)
		cmds = append(cmds, cmd)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	if isAiGenCompleted {
		return m, tea.Quit
	}

	return m, tea.Batch(cmds...)
}

var aiCommandGenerationModelPromtLipgloss = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFFDF5")).
	Background(lipgloss.Color("#25A065")).
	Padding(0, 1)

func (m aiCommandGenerationModel) View() string {
	m.markdown.Viewport.SetContent(currentDisplayMarkdown)
	var view string = "\n\n"
	// Print the view inside of a lipgloss container
	view += aiCommandGenerationModelPromtLipgloss.Render(m.prompt)
	view += "\n"
	view += m.markdown.View()
	return view
}

func markdownToCommand(contents string) Command {
	lines := strings.Split(contents, "\n")
	// Read until "##"
	var title string
	var isLookingForCodeblock bool
	var codeBlockContent string
	var description string
	var isLookingForCodeblockEnd bool
	var metadata map[string]map[string]string = make(map[string]map[string]string)
	var markdown string
	for _, line := range lines {
		if strings.HasPrefix(line, "### ") {
			title = strings.TrimPrefix(line, "### ")
			description = ""
			isLookingForCodeblock = true
			markdown = ""
			metadata = make(map[string]map[string]string)
		} else if isLookingForCodeblock && !strings.HasPrefix(line, "```") {
			description += line + "\n"
		} else if isLookingForCodeblock && strings.HasPrefix(line, "```") {
			isLookingForCodeblock = false
			isLookingForCodeblockEnd = true
			codeBlockContent = ""
		} else if isLookingForCodeblockEnd && strings.HasPrefix(line, "```") {
			isLookingForCodeblockEnd = false
		} else if isLookingForCodeblockEnd {
			codeBlockContent += line + "\n"
		}

		if strings.HasPrefix(line, "[") && strings.Contains(line, "]: <> (") {
			// [key]: <> (value)
			// The line looks like above, extract the key and the value
			re := regexp.MustCompile(`^\[(.*)\]: <> \((.*)\)$`)
			matches := re.FindStringSubmatch(line)
			if len(matches) == 3 {
				// Parse the value
				var value string = matches[2]
				// The value looks like: key1=value1 key2="value2"
				// Extract all keys and values
				value_re := regexp.MustCompile(`([A-Za-z0-9_]+)=(([^\s"]+)|("[^"]+"))`)
				value_matches := value_re.FindAllStringSubmatch(value, -1)
				if len(value_matches) > 0 {
					metadata[matches[1]] = make(map[string]string)
					for _, match := range value_matches {
						metadata[matches[1]][match[1]] = strings.Trim(match[2], "\"")
					}
				}
			}
		} else {
			markdown += line + "\n"
		}
	}

	// Extract the names of the variables inside of {}, <>
	codeBlockLines := strings.Split(codeBlockContent, "\n")
	var variables []string
	for _, codeBlockLine := range codeBlockLines {
		// Find all regex matches
		reVariableNameRegex := regexp.MustCompile("[{<]([A-Za-z\\d\\-_\\/]+)[>}]")
		matches := reVariableNameRegex.FindAllStringSubmatch(codeBlockLine, -1)
		for _, match := range matches {
			variables = append(variables, match[1])
		}
	}

	// Trim the last "\n" from the codeBlockContent and description
	codeBlockContent = strings.TrimSuffix(codeBlockContent, "\n")
	description = strings.TrimSuffix(description, "\n")

	// Write the markdown file
	markdownFilePath := filepath.Join(os.TempDir(), "cwc-ai-"+title+".md")
	err := os.MkdirAll(filepath.Dir(markdownFilePath), 0744)
	if err != nil {
		log.Fatal("Failed to create markdown file parent directories", "error", err)
	}
	markdownFile, err := os.Create(markdownFilePath)
	if err != nil {
		log.Fatal("Failed to create markdown file", "error", err)
	}
	_, err = markdownFile.WriteString(markdown)
	if err != nil {
		log.Fatal("Failed to write markdown content", "error", err)
	}
	err = markdownFile.Close()
	if err != nil {
		log.Fatal("Failed to close markdown file", "error", err)
	}

	// Write the command to the index serialized as json
	return Command{
		CmdTitle:       title,
		Content:        codeBlockContent,
		Variables:      variables,
		CmdDescription: description,
		MarkdownFile:   markdownFilePath,
		Metadata:       metadata,
	}
}
