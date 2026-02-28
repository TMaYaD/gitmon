//go:build windows

package main

import (
	"log"
	"os"
	"os/exec"
	"time"
)

func shutdownSignals() []os.Signal {
	return nil
}

func setProcGroup(cmd *exec.Cmd) {
	// Windows does not support Setpgid; processes are managed individually.
}

func killCmd(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	// On Windows, Process.Kill is the only reliable way to terminate.
	if err := cmd.Process.Kill(); err != nil {
		return
	}

	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Printf("process did not exit after kill")
	}
}
