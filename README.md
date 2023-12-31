# Commands.wiki CLI

This is a commandline version for [commands.wiki](https://commands.wiki). It can be used to run commands without having to access the website.

## Install
```sh
curl https://raw.githubusercontent.com/BL19/commands-wiki-cli/main/clone_and_install.sh -sSf | sh
cwc
```

## Demo
![Showcase](https://cdn.bl19.dev/random-stuff/render1701722402957.gif)

### Ai Features
![Showcase](https://cdn.bl19.dev/random-stuff/ai-demo.mp4)

## Features
- [x] List all commands in the wiki (list)
- [x] Display the markdown for commands
- [x] Run commands with placeholders (<>,{})
- [x] Search for commands
- [x] Update from the git repository for commands.wiki
- [x] Validation of variables from markdown commands using a custom syntax in the markdown
- [x] Update index automatically every 24 hours
- [x] Automatic updates to `cwc` when the branch is updates
- [x] Using AI to generate commands

## Usage
To begin, install `cwc`, then run `cwc`.

### Update the commands
To update the command index run `cwc update`, this will pull the git repository and index all commands again.

### Reset the installation
To reset the cli to default settings run `cwc clean`.

### Search for a command
Either run `cwc` and search using `/<searchterm>`, or run `cwc <searchterm>`.

## Installation From source
Run the `install.sh` script as root, this will build and install `cwc` in `/usr/local/bin`.
```bash
git clone https://github.com/BL19/commands-wiki-cli
cd commands-wiki-cli
sudo bash build_and_install.sh
```

### Updating
```
cd commands-wiki-cli
git pull
sudo bash build_and_install.sh
```
