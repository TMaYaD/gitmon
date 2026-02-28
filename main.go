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
	fmt.Fprintf(os.Stderr, "Usage: gitmon [options] command [args...]\n")
	fmt.Fprintf(os.Stderr, "\nPeriodically git fetch, pull on changes, and restart the command.\n")
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	fmt.Fprintf(os.Stderr, "  -i seconds              Poll interval in seconds (default: 30)\n")
	fmt.Fprintf(os.Stderr, "  -g, --graceful           Send SIGQUIT before SIGTERM to allow graceful shutdown\n")
	fmt.Fprintf(os.Stderr, "  -G, --graceful-timeout dur  Time to wait for graceful shutdown (default: 60m, implies -g)\n")
	os.Exit(1)
}

var gracefulTimeout time.Duration

func main() {
	log.SetPrefix("[gitmon] ")
	log.SetFlags(log.Ltime)

	interval := 30
	graceful := false
	gracefulTimeout = 60 * time.Minute
	args := os.Args[1:]

	for len(args) > 0 {
		switch args[0] {
		case "-i":
			if len(args) < 2 {
				usage()
			}
			v, err := strconv.Atoi(args[1])
			if err != nil || v <= 0 {
				log.Fatalf("invalid interval: %s", args[1])
			}
			interval = v
			args = args[2:]
		case "-g", "--graceful":
			graceful = true
			args = args[1:]
		case "-G", "--graceful-timeout":
			if len(args) < 2 {
				usage()
			}
			d, err := time.ParseDuration(args[1])
			if err != nil || d <= 0 {
				log.Fatalf("invalid shutdown timeout: %s", args[1])
			}
			gracefulTimeout = d
			graceful = true
			args = args[2:]
		case "-h", "--help":
			usage()
		default:
			goto done
		}
	}
done:

	if len(args) == 0 {
		usage()
	}

	cmdArgs := args
	if graceful {
		log.Printf("monitoring every %ds (graceful shutdown: %s), running: %s", interval, gracefulTimeout, strings.Join(cmdArgs, " "))
	} else {
		log.Printf("monitoring every %ds, running: %s", interval, strings.Join(cmdArgs, " "))
	}

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
			killCmd(cmd, graceful)
			os.Exit(0)

		case <-ticker.C:
			if changed := checkForChanges(); changed {
				log.Printf("changes detected, stopping process before pull...")
				killCmd(cmd, graceful)
				if err := gitPull(); err != nil {
					log.Printf("git pull failed: %v", err)
					cmd = startCmd(cmdArgs)
					continue
				}
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
