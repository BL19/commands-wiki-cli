# Commands.wiki CLI

This is a commandline version for [commands.wiki](https://commands.wiki). It can be used to run commands without having to access the website.

## Features
- [x] List all commands in the wiki (list)
- [x] Display the markdown for commands
- [x] Run commands with placeholders (<>,{})
- [x] Search for commands
- [x] Update from the git repository for commands.wiki

## Usage
To begin, install `cwc`, then run `cwc`.

### Update the commands
To update the command index run `cwc update`, this will pull the git repository and index all commands again.

### Reset the installation
To reset the cli to default settings run `cwc clean`.

### Search for a command
Either run `cwc` and search using `/<searchterm>`, or run `cwc <searchterm>`.

## Installation
### From source
Run the `install.sh` script as root, this will build and install `cwc` in `/usr/local/bin`.
