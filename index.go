package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Command struct {
	CmdTitle       string
	CmdDescription string
	Content        string
	Variables      []string
	MarkdownFile   string
}

func (i Command) Title() string       { return i.CmdTitle }
func (i Command) Description() string { return i.CmdDescription }
func (i Command) FilterValue() string { return i.CmdTitle }

func readIndex() ([]Command, error) {
	configPath, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	repo_name, err := GetRepoName()
	if err != nil {
		return nil, err
	}
	indexPath := filepath.Join(configPath, "commands-wiki", "index", repo_name, "index")
	file, err := os.Open(indexPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var commands []Command
	err = json.NewDecoder(file).Decode(&commands)
	if err != nil {
		return nil, err
	}
	return commands, nil
}
