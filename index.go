package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Command struct {
	CmdTitle       string
	CmdDescription string
	Content        string
	Variables      []string
	MarkdownFile   string
	Metadata       map[string]map[string]string
	AiGenerated    bool
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

func getLastIndexUpdate() (uint64, error) {
	configPath, err := os.UserConfigDir()
	if err != nil {
		return 0, err
	}

	repo_name, err := GetRepoName()
	if err != nil {
		return 0, err
	}
	lastUpdatePath := filepath.Join(configPath, "commands-wiki", "index", repo_name, "lastUpdate")
	file, err := os.Open(lastUpdatePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	var lastUpdate uint64
	err = json.NewDecoder(file).Decode(&lastUpdate)
	if err != nil {
		return 0, err
	}
	return lastUpdate, nil
}

func setIndexUpdateTimeToNow() {
	configPath, err := os.UserConfigDir()
	if err != nil {
		return
	}

	repo_name, err := GetRepoName()
	if err != nil {
		return
	}
	lastUpdatePath := filepath.Join(configPath, "commands-wiki", "index", repo_name, "lastUpdate")
	file, err := os.Create(lastUpdatePath)
	if err != nil {
		return
	}
	defer file.Close()
	json.NewEncoder(file).Encode(uint64(time.Now().UnixMilli()))
}

func checkIfUpdateNeeded() {
	lastUpdate, err := getLastIndexUpdate()
	if err != nil {
		return
	}

	// Get the update interval from the config
	timeout, err := strconv.ParseUint(GetValueNoError("git-update-interval", "86400000"), 10, 64)
	if err != nil {
		timeout = 86400000
	}
	if uint64(time.Now().UnixMilli())-lastUpdate > timeout {
		// Update the index
		default_repo, err := GetRepo()
		if err != nil {
			return
		}
		err = updateIndex(default_repo, getDefaultBranch())
		if err != nil {
			return
		}
	}
}

func getDefaultBranch() string {
	defaultBranch := GetValueNoError("branch", "master")
	configPath, err := os.UserConfigDir()
	if err != nil {
		return defaultBranch
	}

	repo_name, err := GetRepoName()
	if err != nil {
		return defaultBranch
	}
	branchPath := filepath.Join(configPath, "commands-wiki", "index", repo_name, "branch")
	file, err := os.Open(branchPath)
	if err != nil {
		return defaultBranch
	}
	defer file.Close()
	var branch string
	err = json.NewDecoder(file).Decode(&branch)
	if err != nil {
		return defaultBranch
	}
	return branch

}

func setIndexBranch(branch string) {
	configPath, err := os.UserConfigDir()
	if err != nil {
		return
	}

	repo_name, err := GetRepoName()
	if err != nil {
		return
	}
	branchPath := filepath.Join(configPath, "commands-wiki", "index", repo_name, "branch")
	file, err := os.Create(branchPath)
	if err != nil {
		return
	}
	defer file.Close()
	json.NewEncoder(file).Encode(branch)
}

func getIndexBranch() (string, error) {
	configPath, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	repo_name, err := GetRepoName()
	if err != nil {
		return "", err
	}
	branchPath := filepath.Join(configPath, "commands-wiki", "index", repo_name, "branch")
	file, err := os.Open(branchPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var branch string
	err = json.NewDecoder(file).Decode(&branch)
	if err != nil {
		return "", err
	}
	return branch, nil
}

func writeIndex(commands []Command) error {

	configPath, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	repo_name, err := GetRepoName()
	if err != nil {
		return err
	}

	indexPath := filepath.Join(configPath, "commands-wiki", "index", repo_name)
	// Create the indexPath if it does not exist
	err = os.MkdirAll(indexPath, 0744)
	if err != nil {
		return err
	}
	indexFilePath := filepath.Join(indexPath, "index")

	// Open the index file
	indexFile, err := os.OpenFile(indexFilePath, os.O_WRONLY, 0744)
	if err != nil {
		return err
	}

	// Write the commands array to the json file
	jsonBytes, err := json.Marshal(commands)
	if err != nil {
		return err
	}
	_, err = indexFile.Write(jsonBytes)
	if err != nil {
		return err
	}
	indexFile.WriteString("\n")
	err = indexFile.Close()
	if err != nil {
		return err
	}
	return nil
}
