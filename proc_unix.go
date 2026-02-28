//go:build !windows

package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func shutdownSignals() []os.Signal {
	return []os.Signal{syscall.SIGTERM}
}

func setProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killCmd(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Process already exited
		return
	}

	// Kill the entire process group
	syscall.Kill(-pgid, syscall.SIGTERM)

	// Give it a moment to exit gracefully
	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Printf("process did not exit gracefully, sending SIGKILL")
		syscall.Kill(-pgid, syscall.SIGKILL)
	}
}
