package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	default_repo, err := GetRepo()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// updateIndex [--repo <repo>]
	updateIndexCmd := flag.NewFlagSet("updateIndex", flag.ExitOnError)
	updateIndexRepo := updateIndexCmd.String("repo", default_repo, "repo <repo>")
	updateIndexBranch := updateIndexCmd.String("branch", getDefaultBranch(), "branch <branch>")

	// search <term>
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)

	if len(os.Args) < 2 {
		search("")
		os.Exit(0)
	}

	switch os.Args[1] {
	case "update", "updateIndex":
		updateIndexCmd.Parse(os.Args[2:])
		err := updateIndex(*updateIndexRepo, *updateIndexBranch)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
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
			fmt.Println(err)
			os.Exit(1)
		}
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
