package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/google/go-github/v57/github"
	uuid "github.com/satori/go.uuid"
)

var sha = "local"
var branch = "local"

func main() {
	checkIfUpdate()

	default_repo, err := GetRepo()
	if err != nil {
		log.Fatal("an error occurred whilst trying to get the default repo to use", "error", err)
	}

	// updateIndex [--repo <repo>]
	updateIndexCmd := flag.NewFlagSet("updateIndex", flag.ExitOnError)
	updateIndexRepo := updateIndexCmd.String("repo", default_repo, "repo <repo>")
	updateIndexBranch := updateIndexCmd.String("branch", getDefaultBranch(), "branch <branch>")

	// search <term>
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)

	// ai <prompt>
	aiCmd := flag.NewFlagSet("ai", flag.ExitOnError)

	if len(os.Args) < 2 {
		search("")
		os.Exit(0)
	}

	switch os.Args[1] {
	case "update", "updateIndex":
		updateIndexCmd.Parse(os.Args[2:])
		err := updateIndex(*updateIndexRepo, *updateIndexBranch)
		if err != nil {
			log.Fatal("an error ocurred whilst updating the index", "error", err)
		}
	case "s", "search":
		searchCmd.Parse(os.Args[2:])
		// Join all remaining arguments
		query := ""
		for _, arg := range searchCmd.Args() {
			query += arg + " "
		}
		search(query)
	case "clean":
		err := CleanConfig()
		if err != nil {
			log.Fatal("an error occurred whilst cleaning the config", "error", err)
		}
	case "version":
		fmt.Println("cwc - commands.wiki in your terminal")
		fmt.Println("Commit: " + sha)
		fmt.Println("Branch: " + branch)
		fmt.Println("Github: https://github.com/BL19/commands-wiki-cli")
		os.Exit(0)
	case "ai":
		aiCmd.Parse(os.Args[2:])
		// ai <prompt>
		if len(os.Args) < 3 {
			log.Fatal("ai command requires a prompt")
		}
		query := ""
		for _, arg := range aiCmd.Args() {
			query += arg + " "
		}
		runCommandAiCommandGeneration(query)
	default:
		// Assume we are searching and try to search
		// Join all remaining arguments
		query := ""
		for _, arg := range os.Args[1:] {
			query += arg + " "
		}
		search(query)
	}
}

func checkIfUpdate() {
	if branch == "local" || sha == "local" {
		return
	}

	client := github.NewClient(nil)
	commits, _, err := client.Repositories.ListCommits(context.Background(), "BL19", "commands-wiki-cli", &github.CommitsListOptions{
		SHA: branch,
	})
	if err != nil {
		log.Warn("an error occurred whilst trying to check for updates", "error", err)
		return // Ignore for now and try to update next time instead
	}

	var newCommits []*github.RepositoryCommit
	for _, commit := range commits {
		if commit.GetSHA() == sha {
			break
		}
		newCommits = append(newCommits, commit)
	}

	if len(newCommits) > 0 {
		fmt.Println("Update available: ")

		var changelog string
		for _, commit := range newCommits {
			changelog += "* " + commit.GetCommit().GetMessage() + "\n"
		}
		fmt.Println(changelog)
		fmt.Println()
		fmt.Println("Compare changes: https://github.com/BL19/commands-wiki-cli/compare/" + sha + "..." + *newCommits[0].SHA)
		fmt.Println()
		fmt.Print("Do you want to update (Y/n)? ")
		var input string
		fmt.Scanln(&input)
		if input == "Y" || input == "y" || input == "" {
			log.Info("Updating...")
			// Write a temp update script
			updateScriptPath := "/tmp/" + uuid.NewV4().String() + ".sh"
			file, err := os.Create(updateScriptPath)
			if err != nil {
				log.Fatal("failed to create update script file, try running as sudo", "error", err)
			}
			file.WriteString("#!/bin/bash\n")
			file.WriteString("\necho \"Starting update\"\n")
			// Move the cwc to temp
			file.WriteString("sudo mv /usr/local/bin/cwc /tmp/cwc-" + uuid.NewV4().String() + "\n")
			file.WriteString("curl https://raw.githubusercontent.com/BL19/commands-wiki-cli/main/clone_and_install.sh | bash\n")
			file.WriteString("rm " + updateScriptPath + "\n")
			file.Close()
			// Execute the script
			execCommand("bash", []string{updateScriptPath})
			log.Infof("Update successful")
			os.Exit(0)
		}
	}
}
