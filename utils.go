package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	NEWLINE = "\n"
)

// greet prints the shell input line
func greet() {
	fmt.Fprintf(
		os.Stdout,
		"\033[94m%s\033[0m:\033[96m%s\033[0m \033[93m%s\033[0m ",
		"quash",
		filepath.Base(currDir),
		"λ",
	)
}
