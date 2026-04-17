//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// hideConsole oculta la ventana de consola en Windows
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

