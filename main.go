package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: gitmon [-i seconds] command [args...]\n")
	fmt.Fprintf(os.Stderr, "\nPeriodically git fetch, pull on changes, and restart the command.\n")
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	fmt.Fprintf(os.Stderr, "  -i seconds   Poll interval in seconds (default: 30)\n")
	os.Exit(1)
}

func main() {
	log.SetPrefix("[gitmon] ")
	log.SetFlags(log.Ltime)

	interval := 30
	args := os.Args[1:]

	// Parse -i flag
	for len(args) > 0 {
		if args[0] == "-i" {
			if len(args) < 2 {
				usage()
			}
			v, err := strconv.Atoi(args[1])
			if err != nil || v <= 0 {
				log.Fatalf("invalid interval: %s", args[1])
			}
			interval = v
			args = args[2:]
		} else if args[0] == "-h" || args[0] == "--help" {
			usage()
		} else {
			break
		}
	}

	if len(args) == 0 {
		usage()
	}

	cmdArgs := args
	log.Printf("monitoring every %ds, running: %s", interval, strings.Join(cmdArgs, " "))

	// Start the command
	cmd := startCmd(cmdArgs)

	// Handle signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, append([]os.Signal{os.Interrupt}, shutdownSignals()...)...)

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigs:
			log.Printf("shutting down...")
			killCmd(cmd)
			os.Exit(0)

		case <-ticker.C:
			if changed := checkForChanges(); changed {
				log.Printf("changes detected, pulling and restarting...")
				if err := gitPull(); err != nil {
					log.Printf("git pull failed: %v", err)
					continue
				}
				killCmd(cmd)
				cmd = startCmd(cmdArgs)
			}
		}
	}
}

func startCmd(args []string) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	setProcGroup(cmd)

	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start command: %v", err)
	}

	log.Printf("started process (pid %d)", cmd.Process.Pid)

	// Reap the child in background so it doesn't become a zombie
	go cmd.Wait()

	return cmd
}

func checkForChanges() bool {
	// Fetch from remote
	fetch := exec.Command("git", "fetch")
	if out, err := fetch.CombinedOutput(); err != nil {
		log.Printf("git fetch failed: %s %v", string(out), err)
		return false
	}

	local, err := gitRevParse("HEAD")
	if err != nil {
		log.Printf("git rev-parse HEAD failed: %v", err)
		return false
	}

	remote, err := gitRevParse("@{u}")
	if err != nil {
		log.Printf("git rev-parse @{u} failed (no upstream?): %v", err)
		return false
	}

	if local != remote {
		log.Printf("local=%s remote=%s", local[:8], remote[:8])
		return true
	}
	return false
}

func gitRevParse(ref string) (string, error) {
	cmd := exec.Command("git", "rev-parse", ref)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitPull() error {
	cmd := exec.Command("git", "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
