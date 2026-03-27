package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AgentDrasil/ssh-wrapper/lib/command"
	"github.com/AgentDrasil/ssh-wrapper/lib/config"
	"github.com/AgentDrasil/ssh-wrapper/lib/files"
)

const (
	rootUID = 0

	ConfigPath = "/etc/config.yaml"
	KeyPath    = "/etc/key"
	RealSSH    = "/usr/bin/ssh.orig"
)

func main() {
	if err := files.VerifySecurity(ConfigPath, rootUID, 0400); err != nil {
		fmt.Fprintf(os.Stderr, "Security Error: %v\n", err)
		os.Exit(1)
	}
	if err := files.VerifySecurity(KeyPath, rootUID, 0400); err != nil {
		fmt.Fprintf(os.Stderr, "Security Error: %v\n", err)
		os.Exit(1)
	}
	if err := files.VerifySecurity(RealSSH, rootUID, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Security Error: %v\n", err)
		os.Exit(1)
	}

	conf, err := config.ReadConfig(ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	args := os.Args[1:]
	fullCmd := strings.Join(args, " ")

	if err := command.VerifyAccess(fullCmd, conf); err != nil {
		fmt.Fprintf(os.Stderr, "Access Denied: %v\n", err)
		os.Exit(1)
	}

	os.Clearenv()
	os.Setenv("PATH", "/usr/bin:/bin")

	newArgs := []string{"-i", KeyPath, "-o", "StrictHostKeyChecking=no"}
	newArgs = append(newArgs, args...)

	cmd := exec.Command(RealSSH, newArgs...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}
