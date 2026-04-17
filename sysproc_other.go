//go:build !windows

package main

import (
	"os/exec"
)

// hideConsole es un no-op en sistemas no Windows
func hideConsole(cmd *exec.Cmd) {
	// No-op en Linux/macOS
}

