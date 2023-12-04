package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mistakenelf/teacup/markdown"
)

type cmdInfoModel struct {
	keys                 cmdInfoKeymap
	variableKeys         cmdInfoKeymapVariables
	help                 help.Model
	markdown             markdown.Model
	command              Command
	variables            map[string]string
	isReadingVariables   bool
	variableRegex        *regexp.Regexp
	textInput            textinput.Model
	currentVariableInput string
}

type cmdInfoKeymap struct {
	Up      key.Binding
	Down    key.Binding
	Execute key.Binding
	Quit    key.Binding
	Help    key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k cmdInfoKeymap) ShortHelp() []key.Binding {
	return []key.Binding{k.Execute, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k cmdInfoKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Execute}, // first column
		{k.Help, k.Quit},          // second column
	}
}

var DefaultKeyMap = cmdInfoKeymap{
	Execute: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "run the command"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c", "esc"),
		key.WithHelp("q", "quit"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
}

type cmdInfoKeymapVariables struct {
	Execute key.Binding
	Quit    key.Binding
}

var VariablesKeymap = cmdInfoKeymapVariables{
	Execute: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "save the variable/run command"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "esc"),
		key.WithHelp("ctrl+c", "quit"),
	),
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k cmdInfoKeymapVariables) ShortHelp() []key.Binding {
	return []key.Binding{k.Execute, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k cmdInfoKeymapVariables) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Execute, k.Quit}, // first column
	}
}

func newCmdInfoModel(cmd Command) cmdInfoModel {
	markdownModel := markdown.New(true, true, lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffffff"})
	markdownModel.FileName = cmd.MarkdownFile

	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 150
	ti.Width = 30

	return cmdInfoModel{
		markdown:             markdownModel,
		keys:                 DefaultKeyMap,
		variableKeys:         VariablesKeymap,
		help:                 help.New(),
		command:              cmd,
		variables:            make(map[string]string),
		isReadingVariables:   false,
		currentVariableInput: "",
		textInput:            ti,
	}
}

// Init intializes the UI.
func (m cmdInfoModel) Init() tea.Cmd {
	return nil
}

// Update handles all UI interactions.
func (m cmdInfoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmds = append(cmds, m.markdown.SetSize(msg.Width, msg.Height))

		return m, tea.Batch(cmds...)
	case tea.KeyMsg:
		if m.isReadingVariables && !key.Matches(msg, m.keys.Execute) && msg.Type != tea.KeyCtrlC {
			m.textInput, cmd = m.textInput.Update(msg)
			cmds = append(cmds, cmd)
		}
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if !m.isReadingVariables {
			switch msg.String() {
			case "ctrl+c", "esc", "q":
				cmds = append(cmds, tea.Quit)
			}
		}
		switch {
		case key.Matches(msg, m.keys.Execute):
			if m.isReadingVariables {
				setVariable(&m, &cmds)
				m = updateVariableMetadata(m)
			} else if len(m.command.Variables) > 0 {
				// Ask for the variables
				m.isReadingVariables = true
				m.currentVariableInput = m.command.Variables[0]
				m.variableRegex = nil
				m = updateVariableMetadata(m)
			} else {
				// Run the command
				generateExecCommand(m.command, m.variables)
				cmds = append(cmds, tea.Quit)
			}
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Quit) && !m.isReadingVariables:
			return m, tea.Quit
		}
	}

	m.markdown, cmd = m.markdown.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func updateVariableMetadata(m cmdInfoModel) cmdInfoModel {
	commandMetadata := m.command.Metadata
	// Get the variable metadata
	variableMetadata := commandMetadata[m.currentVariableInput]
	// Get the validation
	variableValidation := variableMetadata["validation"]
	if variableValidation != "" {
		// type <data>
		// Extract the validation type
		reValidationType := regexp.MustCompile("([A-Za-z]+) (.+)")
		matches := reValidationType.FindStringSubmatch(variableValidation)
		if len(matches) == 3 {
			validationType := matches[1]
			validationData := matches[2]
			switch validationType {
			case "regex":
				// Check if the regex is valid
				regex, err := regexp.Compile("^" + validationData + "$")
				if err != nil {
					fmt.Println("Invalid regex: " + validationData)
				}
				m.variableRegex = regex
			}
		}
	}

	// Set the placeholder for the textinput if a placeholder exists
	variablePlaceholder := variableMetadata["placeholder"]
	m.textInput.Placeholder = variablePlaceholder
	return m
}

func setVariable(m *cmdInfoModel, cmds *[]tea.Cmd) {
	if (*m).variableRegex != nil {
		if !(*m).variableRegex.MatchString((*m).textInput.Value()) {
			return
		}
	}
	// Save the variable
	(*m).variables[m.currentVariableInput] = (*m).textInput.Value()
	(*m).textInput.SetValue("")

	// Check if all variables have been read
	var hasMissingVar bool
	for _, variable := range (*m).command.Variables {
		if _, ok := (*m).variables[variable]; !ok {
			// Ask for this variable
			(*m).currentVariableInput = variable
			(*m).variableRegex = nil
			hasMissingVar = true
		}
	}
	if !hasMissingVar {
		// Run the command
		generateExecCommand((*m).command, (*m).variables)
		*cmds = append(*cmds, tea.Quit)
	}
}

var fileToExecuteOnExit *string
var bashFileContent *string

func generateExecCommand(cmd Command, variables map[string]string) {

	// Replace the variables in the content
	var newContent string
	for _, line := range strings.Split(cmd.Content, "\n") {
		for variable, value := range variables {
			line = strings.ReplaceAll(line, "{"+variable+"}", value)
			line = strings.ReplaceAll(line, "<"+variable+">", value)
		}
		newContent += line + "\n"
	}

	bashFileContent = &newContent

	// Write the content to a bash file and run that files
	tempDir := os.TempDir()
	filePath := filepath.Join(tempDir, "cmdwiki-exec-"+cmd.CmdTitle+".sh")
	file, err := os.Create(filePath)
	if err != nil {
		return
	}
	defer file.Close()
	_, err = file.WriteString(newContent)
	if err != nil {
		return
	}
	err = file.Close()
	if err != nil {
		return
	}
	err = os.Chmod(filePath, 0755)
	if err != nil {
		return
	}
	fileToExecuteOnExit = &filePath
}

// View returns a string representation of the UI.
func (m cmdInfoModel) View() string {
	view := m.markdown.View()
	if m.isReadingVariables {
		var variableLines string
		variableLines += "\n\n"

		commandMetadata := m.command.Metadata
		// Get the variable metadata
		variableMetadata := commandMetadata[m.currentVariableInput]
		if variableMetadata == nil {
			variableMetadata = make(map[string]string)
		}

		// Get the description for the variable
		variableDescription := variableMetadata["desc"]
		if variableDescription != "" {
			variableLines += "Description: " + variableDescription + "\n"
		}

		// Remove 4 lines from the from the bottom
		lines := strings.Split(view, "\n")
		variableLinesCount := len(strings.Split(variableLines, "\n"))
		lines = lines[:len(lines)-4-variableLinesCount]
		view = strings.Join(lines, "\n")
		view += variableLines

		view += "Enter the value for " + m.currentVariableInput + ""
		view += m.textInput.View()

		// If we have a regex, validate it
		if m.variableRegex != nil {
			if !m.variableRegex.MatchString(m.textInput.Value()) {
				view += " ❌"
			} else {
				view += " ✔️"
			}
		}

		view += "\n\n"
		view += m.help.View(m.variableKeys)
	} else {
		view += "\n\n"
		view += m.help.View(m.keys)
	}

	return view
}

func showCommmand(cmd Command) {
	b := newCmdInfoModel(cmd)
	p := tea.NewProgram(b, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

	if fileToExecuteOnExit != nil {

		// Run the bash script in the terminal
		fmt.Println("Running command:")
		fmt.Println(*bashFileContent)
		fmt.Println("")

		execCommand("bash", []string{*fileToExecuteOnExit})

		// Remove the file
		err := os.Remove(*fileToExecuteOnExit)
		if err != nil {
			log.Fatal(err)
		}
	}
}
