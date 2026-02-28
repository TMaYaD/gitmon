# gitmon

A lightweight tool that monitors a git repository for changes, automatically pulls updates, and restarts a given command.

## Installation

### Pre-built binaries

Download the latest release for your platform from the [Releases](https://github.com/TMaYaD/gitmon/releases) page.

Available binaries:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

### From source

```sh
go install github.com/tmayad/gitmon@latest
```

## Usage

```
gitmon [-i seconds] command [args...]
```

Run `gitmon` inside a git repository with a tracking branch. It will periodically fetch from the remote, pull when changes are detected, and restart your command.

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `-i` | Poll interval in seconds | 30 |

### Examples

Watch for changes every 60 seconds and restart a web server:

```sh
gitmon -i 60 python -m http.server 8000
```

Restart a Node.js app on changes:

```sh
gitmon node app.js
```

## How it works

1. Starts the specified command as a child process
2. Periodically runs `git fetch` and compares local HEAD with the upstream ref
3. When a difference is detected, runs `git pull`, stops the running process, and restarts the command
4. On Unix, manages process groups so child processes are cleaned up properly
5. Handles SIGINT/SIGTERM for graceful shutdown

## License

MIT
