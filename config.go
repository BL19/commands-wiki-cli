package main

// This file contains everything for reading the config from the users config directory
// The config is a key value file with lines like "repo <value>"

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config is a map of key value pairs
type Config map[string]string

func readConfig(configPath string, config *Config) error {

	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the config directory if it does not exist
			err = os.MkdirAll(filepath.Dir(configPath), 0755)
			if err != nil {
				return err
			}
			// Create the config file if it does not exist
			file, err = os.Create(configPath)
			if err != nil {
				return err
			}
			// Write the default config to the config file
			_, err = file.WriteString("repo https://github.com/lerndmina/commands-wiki")
		} else {
			return err
		}
	}
	defer file.Close()

	var key string
	var value string
	for {
		_, err := fmt.Fscanf(file, "%s %s\n", &key, &value)
		if err != nil {
			break
		}
		if key == "" || value == "" {
			continue
		}
		(*config)[key] = value
	}
	return nil
}

func CleanConfig() error {
	configPath, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	configPath = filepath.Join(configPath, "commands-wiki")
	return os.RemoveAll(configPath)
}

// GetConfig returns the config from the users config directory
func GetConfig() (Config, error) {
	config := Config{}
	configPath, err := os.UserConfigDir()
	if err != nil {
		return config, err
	}
	configPath = filepath.Join(configPath, "commands-wiki", "config")
	err = readConfig(configPath, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func GetValue(key string, defaultValue string) (string, error) {
	config, err := GetConfig()
	if err != nil {
		return "", err
	}
	value, ok := config[key]
	if !ok {
		return defaultValue, nil
	}
	return value, nil
}

func GetValueNoError(key string, defaultValue string) string {
	value, err := GetValue(key, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetRepo returns the repo from the config
func GetRepo() (string, error) {
	return GetValue("repo", "https://github.com/lerndmina/commands-wiki")
}

// GetRepoName returns the repo name from the config
func GetRepoName() (string, error) {
	repo, err := GetRepo()
	if err != nil {
		return "", err
	}
	split := strings.Split(repo, "/")
	if len(split) < 2 {
		return "", fmt.Errorf("repo name not found in repo url")
	}
	return split[len(split)-2] + "/" + split[len(split)-1], nil
}
