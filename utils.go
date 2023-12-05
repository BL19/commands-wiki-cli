package main

import (
	"os"
	"os/exec"
)

// Usage: cmd := execCommand("git", []string{"clone", "-b", "gh-pages", repo, repo_path})
//
// This function is used to execute a command with arguments
// It returns the command object
func execCommand(command string, args []string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Dir = os.TempDir()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Run()
	return cmd
}
