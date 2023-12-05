package main

import (
	"sort"
	"strings"

	"github.com/charmbracelet/log"

	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
				Render
)

type listKeyMap struct {
	toggleSpinner    key.Binding
	toggleTitleBar   key.Binding
	toggleStatusBar  key.Binding
	togglePagination key.Binding
	toggleHelpMenu   key.Binding
	insertItem       key.Binding
}

func search(searchterm string) {
	checkIfUpdateNeeded()
	commands, err := readIndex()
	if err != nil {
		if os.IsNotExist(err) {
			log.Infof("Index not found, creating index using default settings")
			default_repo, err := GetRepo()
			if err != nil {
				log.Fatal("some error occured whilst getting the repo", "error", err)
				return
			}
			err = updateIndex(default_repo, getDefaultBranch())
			if err != nil {
				log.Fatal("some error occured whilst updating the index", "error", err)
				return
			}
			commands, err = readIndex()
			if err != nil {
				log.Fatal("some error occured whilst reading the index", "error", err)
				return
			}
		} else {
			log.Fatal("some error occured whilst reading the index", "error", err)
			return
		}
	}

	searchtermWords := strings.Split(searchterm, " ")
	// Remove empty entries
	for i := 0; i < len(searchtermWords); i++ {
		if searchtermWords[i] == "" {
			searchtermWords = append(searchtermWords[:i], searchtermWords[i+1:]...)
			i--
		}
	}

	// Create a map of all commands and their scores
	commandScores := make(map[string]float32)

	// Search if the command title or content contains one or more words from the searchterm
	// Use case-insensitive search
	for _, cmd := range commands {
		var score float32
		titleWords := strings.Split(cmd.CmdTitle, " ")
		descriptionWords := strings.Split(cmd.CmdDescription, " ")
		contentWords := strings.Split(cmd.Content, " ")
		for _, searchtermWord := range searchtermWords {
			for _, titleWord := range titleWords {
				if strings.HasPrefix(strings.ToLower(titleWord), strings.ToLower(searchtermWord)) {
					score += 1
				}
				if strings.Contains(strings.ToLower(titleWord), strings.ToLower(searchtermWord)) {
					score += 0.125
				}
			}
			for _, descriptionWord := range descriptionWords {
				if strings.HasPrefix(strings.ToLower(descriptionWord), strings.ToLower(searchtermWord)) {
					score += 0.5
				}
				if strings.Contains(strings.ToLower(descriptionWord), strings.ToLower(searchtermWord)) {
					score += 0.125 / 2
				}
			}
			for _, contentWord := range contentWords {
				if strings.HasPrefix(strings.ToLower(contentWord), strings.ToLower(searchtermWord)) {
					score += 0.5
				}
				if strings.Contains(strings.ToLower(contentWord), strings.ToLower(searchtermWord)) {
					score += 0.125 / 4
				}
			}
		}

		if score > 0 {
			// Add the command to commandscores
			commandScores[cmd.CmdTitle] = score
		}
	}

	keys := make([]string, 0, len(commandScores))

	for key := range commandScores {
		keys = append(keys, key)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return commandScores[keys[i]] > commandScores[keys[j]]
	})

	var filteredCommands []Command
	if searchterm == "" {
		filteredCommands = commands
	} else {
		for _, k := range keys {
			// Find the command
			for _, cmd := range commands {
				if cmd.CmdTitle == k {
					filteredCommands = append(filteredCommands, cmd)
				}
			}
		}
	}

	if len(filteredCommands) == 0 {
		log.Info("No commands found", "searchterm", searchterm)
		return
	}

	if len(filteredCommands) == 1 {
		showCommmand(filteredCommands[0])
		return
	}

	if _, err := tea.NewProgram(newSearchModel(filteredCommands)).Run(); err != nil {
		log.Fatal("error during program execution", "error", err)
	}

	if selectedCommand != nil {
		showCommmand(*selectedCommand)
	}

}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		toggleSpinner: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "toggle spinner"),
		),
		toggleTitleBar: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "toggle title"),
		),
		toggleStatusBar: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "toggle status"),
		),
		togglePagination: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "toggle pagination"),
		),
		toggleHelpMenu: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "toggle help"),
		),
	}
}

type searchModel struct {
	list         list.Model
	keys         *listKeyMap
	delegateKeys *searchDelegateKeyMap
}

func newSearchModel(cmds []Command) searchModel {
	var (
		delegateKeys = newSearchDelegateKeyMap()
		listKeys     = newListKeyMap()
	)

	// Make initial list of items
	var numItems = len(cmds)
	items := make([]list.Item, numItems)
	for i := 0; i < numItems; i++ {
		items[i] = cmds[i]
	}

	// Setup list
	delegate := newSearchItemDelegate(delegateKeys)
	commandsList := list.New(items, delegate, 0, 0)
	commandsList.Title = "Found Commands"
	commandsList.Styles.Title = titleStyle
	commandsList.SetStatusBarItemName("command", "commands")
	commandsList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.toggleSpinner,
			listKeys.toggleTitleBar,
			listKeys.toggleStatusBar,
			listKeys.togglePagination,
			listKeys.toggleHelpMenu,
		}
	}

	return searchModel{
		list:         commandsList,
		keys:         listKeys,
		delegateKeys: delegateKeys,
	}
}

func (m searchModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.toggleSpinner):
			cmd := m.list.ToggleSpinner()
			return m, cmd

		case key.Matches(msg, m.keys.toggleTitleBar):
			v := !m.list.ShowTitle()
			m.list.SetShowTitle(v)
			m.list.SetShowFilter(v)
			m.list.SetFilteringEnabled(v)
			return m, nil

		case key.Matches(msg, m.keys.toggleStatusBar):
			m.list.SetShowStatusBar(!m.list.ShowStatusBar())
			return m, nil

		case key.Matches(msg, m.keys.togglePagination):
			m.list.SetShowPagination(!m.list.ShowPagination())
			return m, nil

		case key.Matches(msg, m.keys.toggleHelpMenu):
			m.list.SetShowHelp(!m.list.ShowHelp())
			return m, nil
		}
	}

	// This will also call our delegate's update function.
	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m searchModel) View() string {
	return appStyle.Render(m.list.View())
}
