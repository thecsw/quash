package main

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func main() {
	// read one line of input from Stdin
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSuffix(input, "\n")

	// fmt.Printf("> ")
	// fmt.Scanf("%s", &input)
	// fmt.Println(input)

	// split input into different commands to be executed
	commands := strings.Split(input, " | ")

	// make pipes to communicate between the different processes
	pipeRead := make([]*os.File, len(commands)-1)
	pipeWrite := make([]*os.File, len(commands)-1)
	for index := range pipeRead {
		pipeRead[index], pipeWrite[index], _ = os.Pipe()
	}

	// fork and execute each command as its own process
	for index, command := range commands {
		//seperate command into its executable name and arguments
		args := strings.Split(command, " ")
		cmdName := args[0]

		// see if & present, signifies if program runs in background
		background := strings.Contains(command, "&")
		if background {
			// remove the & from args. Can we guarentee that & is the last command? I believe bash syntax forces this
			args = args[:len(args)-1]
		}

		// find path to executable
		paths, _ := exec.LookPath(cmdName)

		// make actual fork happen
		pid, err := syscall.ForkExec(
			//_, err := syscall.ForkExec(
			paths, args, &syscall.ProcAttr{
				Dir:   "",
				Env:   os.Environ(),
				Files: fileDescriptor(index, pipeRead, pipeWrite),
				//Sys: &syscall.SysProcAttr{Foreground: !background},
				Sys: &syscall.SysProcAttr{}, // having trouble setting Foreground to true without program failing to terminate probably extra flags and such that need to be set, but dont know which
			})
		if err != nil {
			panic(err)
		}

		// close pipes that have been used to prevent stalling
		closePipe(index, pipeRead, pipeWrite)

		// wait for new process to finish if running in foreground
		if !background {
			process, _ := os.FindProcess(pid)
			process.Wait()
		}
		//fmt.Println("New process's ID:", pid)

	}
}

// returns a custom file descriptor for a call to ForkExec.
// if there is only one command with no pipes, Stdin Stdout and Stderr are used
// pipes overwrite read, write, or both for processes inside of a pipe chain
func fileDescriptor(index int, readPipe []*os.File, writePipe []*os.File) []uintptr {
	if len(readPipe) == 0 { // One command, so no pipes
		return []uintptr{
			os.Stdin.Fd(),
			os.Stdout.Fd(),
			os.Stderr.Fd(),
		}
	}
	if index == 0 { // first in a chain
		return []uintptr{
			os.Stdin.Fd(),
			writePipe[0].Fd(),
			os.Stderr.Fd(),
		}
	} else if index == len(readPipe) { // last in a chain
		return []uintptr{
			readPipe[index-1].Fd(),
			os.Stdout.Fd(),
			os.Stderr.Fd(),
		}
	} else { // middle of a chain
		return []uintptr{
			readPipe[index-1].Fd(),
			writePipe[index].Fd(),
			os.Stderr.Fd(),
		}
	}
}

// closes used pipe ends based on where they are in a chain of piped commands
// if only one command exists, there are no pipes and this function does nothing
func closePipe(index int, readPipe []*os.File, writePipe []*os.File) {
	if len(readPipe) == 0 { // One command, so no pipes
	} else if index == 0 { // first in a chain
		writePipe[0].Close()
	} else if index == len(readPipe) { // last in a chain
		readPipe[index-1].Close()
	} else { // middle of a chain
		readPipe[index-1].Close()
		writePipe[index].Close()
	}
}
