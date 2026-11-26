//go:build !windows

package main

import "os/exec"

func hideCmdWindow(cmd *exec.Cmd) {
	// No hidden console window attribute needed on Unix-like systems.
}
