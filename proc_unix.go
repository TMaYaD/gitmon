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

func killCmd(cmd *exec.Cmd, graceful bool) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return
	}

	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	if graceful {
		log.Printf("sending SIGQUIT, waiting up to %s for graceful shutdown...", gracefulTimeout)
		syscall.Kill(-pgid, syscall.SIGQUIT)

		select {
		case <-done:
			return
		case <-time.After(gracefulTimeout):
			log.Printf("graceful shutdown timed out, sending SIGTERM...")
		}
	}

	syscall.Kill(-pgid, syscall.SIGTERM)

	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
		log.Printf("process did not exit after SIGTERM, sending SIGKILL")
		syscall.Kill(-pgid, syscall.SIGKILL)
	}
}
