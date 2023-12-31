package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
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
	log.Infof("cloning repo %s into %s\n", repo, repo_path)
	config, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	if config == "" {
		return fmt.Errorf("config is empty")
	}
	// If the directory exists, update the repo
	if _, err := os.Stat(repo_path); err == nil {
		log.Info("updating repo")
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

	ai_commands_path := filepath.Join(configPath, "commands-wiki", "ai")
	err = filepath.Walk(ai_commands_path, func(path string, info os.FileInfo, err error) error {
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
		log.Infof("creating index")
		// Create all folders up to that point
		err := os.MkdirAll(filepath.Dir(indexFilePath), 0744)
		if err != nil {
			return err
		}
		_, err = os.Create(indexFilePath)
		if err != nil {
			return err
		}
		log.Info("created index", "indexPath", indexFilePath)
	}

	var commands []Command

	// Write the index file
	for _, file := range commandsFiles {
		// Read the file contents
		contentsBytes, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		var isAiCommand bool
		if strings.HasPrefix(file, ai_commands_path) {
			isAiCommand = true
		}

		contents := string(contentsBytes)
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
				if title != "" {
					addCmd(title, codeBlockContent, &commands, description, markdown, markdownRoot, metadata, isAiCommand)
				}
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
		if title != "" {
			addCmd(title, codeBlockContent, &commands, description, markdown, markdownRoot, metadata, isAiCommand)
		}
	}

	err = writeIndex(commands)
	if err != nil {
		return err
	}

	setIndexUpdateTimeToNow()
	setIndexBranch(branch)

	return nil
}

func addCmd(title string, codeBlockContent string, commands *[]Command, description string, markdown string, markdownRoot string, metadata map[string]map[string]string, isAi bool) {
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
	markdownFilePath := filepath.Join(markdownRoot, title+".md")
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
	cmd := Command{
		CmdTitle:       title,
		Content:        codeBlockContent,
		Variables:      variables,
		CmdDescription: description,
		MarkdownFile:   markdownFilePath,
		Metadata:       metadata,
		AiGenerated:    isAi,
	}
	*commands = append(*commands, cmd)
}
