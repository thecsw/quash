package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func main() {
	input := "date"
	fmt.Printf("> ")
	fmt.Scanf("%s", &input)

	commands := strings.Split(input, "|")
	fmt.Println(commands)
	pipeRead := make([]*os.File, len(commands)-1)
	pipeWrite := make([]*os.File, len(commands)-1)

	for index := range pipeRead {
		pipeRead[index], pipeWrite[index], _ = os.Pipe()
	}

	for index, command := range commands {

		paths, _ := exec.LookPath(command)

		//pid, err := syscall.ForkExec(
		_, err := syscall.ForkExec(
			paths, []string{command}, &syscall.ProcAttr{
				Dir:   "",
				Env:   os.Environ(),
				Files: fileDescriptor(index, pipeRead, pipeWrite),
				Sys:   &syscall.SysProcAttr{},
			})
		if err != nil {
			panic(err)
		}
		//fmt.Println("New process's ID:", pid)

	}
}

func fileDescriptor(index int, readPipe []*os.File, writePipe []*os.File) []uintptr {
	if index == 0 {
		return []uintptr{
			os.Stdin.Fd(),
			writePipe[0].Fd(),
			os.Stderr.Fd(),
		}
	} else if index == len(readPipe) {
		return []uintptr{
			readPipe[index-1].Fd(),
			os.Stdout.Fd(),
			os.Stderr.Fd(),
		}
	} else {
		return []uintptr{
			readPipe[index-1].Fd(),
			writePipe[index].Fd(),
			os.Stderr.Fd(),
		}
	}
}
