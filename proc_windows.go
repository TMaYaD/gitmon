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

func killCmd(cmd *exec.Cmd, graceful bool) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	if graceful {
		// Windows has no SIGQUIT; send interrupt and wait for graceful timeout
		log.Printf("sending interrupt, waiting up to %s for graceful shutdown...", gracefulTimeout)
		cmd.Process.Signal(os.Interrupt)

		select {
		case <-done:
			return
		case <-time.After(gracefulTimeout):
			log.Printf("graceful shutdown timed out, killing process...")
		}
	}

	cmd.Process.Kill()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Printf("process did not exit after kill")
	}
}
