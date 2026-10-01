package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "resolve project directory:", err)
		os.Exit(1)
	}
	script := filepath.Join(root, "run-backend.ps1")
	if _, err := os.Stat(script); err != nil {
		fmt.Fprintln(os.Stderr, "run this command from the STOCKER project root:", err)
		os.Exit(1)
	}

	powerShell := "powershell"
	if runtime.GOOS != "windows" {
		powerShell = "pwsh"
	}
	arguments := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script}
	arguments = append(arguments, os.Args[1:]...)
	command := exec.Command(powerShell, arguments...)
	command.Dir = root
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			os.Exit(exitError.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "start backend:", err)
		os.Exit(1)
	}
}
