package main

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

func main() {
	input := "date"
	fmt.Printf("> ")
	fmt.Scanf("%s", &input)

	// Create pipes
	// pipeOneRead, pipeOneWrite, _ := os.Pipe()
	// pipeTwoRead, pipeTwoWrite, _ := os.Pipe()

	commands := strings.Split(input, "|")
	for _, command := range commands {

		paths, _ := exec.LookPath(command)

		pid, err := syscall.ForkExec(
			paths, []string{command}, &syscall.ProcAttr{
				Dir:   "",
				Env:   []string{},
				Files: []uintptr{
					// os.Stdin.Fd(),
					// pipeOneWrite.Fd()},
					// pipeOneRead.Fd(),
					// pipeTwoWrite.Fd()
					// ...
					// pipeNRead().Fd()
					// os.Stdout.Fd()
				},
				Sys: &syscall.SysProcAttr{},
			})
		if err != nil {
			panic(err)
		}
		fmt.Println("New process's ID:", pid)

	}
}
