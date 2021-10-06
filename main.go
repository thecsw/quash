package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	input := "date"
	fmt.Printf("> ")
	fmt.Scanf("%s", &input)

	paths, _ := exec.LookPath(input)

	pid, err := syscall.ForkExec(paths, []string{input}, &syscall.ProcAttr{
		Dir:   "",
		Env:   []string{},
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
		Sys:   &syscall.SysProcAttr{},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("New process's ID:", pid)
}
