package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func updateIndex(repo string, branch string) error {
	// Define where to put the cloned repo, it should be in the config directory with the "repos/<reponame>" directory
	// If the directory does not exist, create it
	configPath, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	repo_name, err := GetRepoName()
	if err != nil {
		return err
	}
	repo_path := filepath.Join(configPath, "commands-wiki", "repos", repo_name)
	fmt.Printf("cloning repo %s into %s\n", repo, repo_path)
	config, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	if config == "" {
		return fmt.Errorf("config is empty")
	}
	// If the directory exists, update the repo
	if _, err := os.Stat(repo_path); err == nil {
		fmt.Println("updating repo")
		cmd := execCommand("git", []string{"-C", repo_path, "pull"})
		if cmd.ProcessState.ExitCode() != 0 {
			return fmt.Errorf("git pull failed")
		}
	} else {
		cmd := execCommand("git", []string{"clone", "-b", branch, repo, repo_path})
		if cmd.ProcessState.ExitCode() != 0 {
			return fmt.Errorf("git clone failed")
		}
	}

	// Checkout the branch we have selected
	cmd := execCommand("git", []string{"-C", repo_path, "checkout", branch})
	if cmd.ProcessState.ExitCode() != 0 {
		return fmt.Errorf("git checkout failed")
	}

	// Find all .md files recursively in the "src/content/docs/commands/" of the repo
	var commandsFiles []string
	err = filepath.Walk(filepath.Join(repo_path, "src", "content", "docs", "commands"), func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".md" {
			commandsFiles = append(commandsFiles, path)
		}
		return nil
	})

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
	markdownRoot := filepath.Join(indexPath, "cmds")
	// Remove all files in the markdown root if the directory exists
	if _, err := os.Stat(markdownRoot); err == nil {
		err = os.RemoveAll(markdownRoot)
		if err != nil {
			return err
		}
	}

	// If the index file does not exist, create it
	if _, err := os.Stat(indexFilePath); os.IsNotExist(err) {
		fmt.Println("creating index")
		// Create all folders up to that point
		err := os.MkdirAll(filepath.Dir(indexFilePath), 0744)
		if err != nil {
			return err
		}
		_, err = os.Create(indexFilePath)
		if err != nil {
			return err
		}
		fmt.Println("created index at", indexFilePath)
	}

	// Open the index file
	indexFile, err := os.OpenFile(indexFilePath, os.O_WRONLY, 0744)
	if err != nil {
		return err
	}

	var commands []Command

	// Write the index file
	for _, file := range commandsFiles {
		// Read the file contents
		contentsBytes, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		contents := string(contentsBytes)
		lines := strings.Split(contents, "\n")
		// Read until "##"
		var title string
		var isLookingForCodeblock bool
		var codeBlockContent string
		var description string
		var isLookingForCodeblockEnd bool
		var markdown string
		for _, line := range lines {
			if strings.HasPrefix(line, "### ") {
				if title != "" {
					addCmd(title, codeBlockContent, &commands, description, markdown, markdownRoot)
				}
				title = strings.TrimPrefix(line, "### ")
				description = ""
				isLookingForCodeblock = true
				markdown = ""
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

			markdown += line + "\n"

		}
		if title != "" {
			addCmd(title, codeBlockContent, &commands, description, markdown, markdownRoot)
		}
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

func addCmd(title string, codeBlockContent string, commands *[]Command, description string, markdown string, markdownRoot string) {
	// Extract the names of the variables inside of {}, <>
	codeBlockLines := strings.Split(codeBlockContent, "\n")
	var variables []string
	for _, codeBlockLine := range codeBlockLines {
		// Find all regex matches
		reVariableNameRegex := regexp.MustCompile("[{<]([A-Za-z_\\/]+)[>}]")
		matches := reVariableNameRegex.FindAllStringSubmatch(codeBlockLine, -1)
		for _, match := range matches {
			variables = append(variables, match[1])
		}
	}

	// Trim the last "\n" from the codeBlockContent and description
	codeBlockContent = strings.TrimSuffix(codeBlockContent, "\n")
	description = strings.TrimSuffix(description, "\n")

	// Write the markdown file
	markdownFilePath := filepath.Join(markdownRoot, title+".md")
	err := os.MkdirAll(filepath.Dir(markdownFilePath), 0744)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	markdownFile, err := os.Create(markdownFilePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	_, err = markdownFile.WriteString(markdown)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = markdownFile.Close()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Write the command to the index serialized as json
	cmd := Command{
		CmdTitle:       title,
		Content:        codeBlockContent,
		Variables:      variables,
		CmdDescription: description,
		MarkdownFile:   markdownFilePath,
	}
	*commands = append(*commands, cmd)
}
