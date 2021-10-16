package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const (
	NEWLINE = "\n"
)

// hello is a small funny message to show to the user
func hello() {
	fmt.Fprintf(os.Stdout, "So you're back... about time...\n")
}

// greet prints the shell input line
func greet() {
	greetLength, _ = fmt.Fprintf(
		os.Stdout,
		"\033[94m%s\033[0m:\033[96m%s\033[0m \033[93m%s\033[0m ",
		getenv("QUASH"),
		filepath.Base(currDir),
		"λ",
	)
}

// jobStopper waits for an interrupt to arrive and then
// delivers it straight to the currently active job
func jobStopper() {
	for {
		<-sigintChan
		if currJob == 0 {
			continue
		}
		syscall.Kill(currJob, syscall.SIGINT)
	}
}
