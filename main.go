package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

var (
	currDir    = ""
	myEnv      = os.Environ()
	cmdHistory []string
)

func main() {
	// Set the current directory
	var err error
	currDir, err = os.Getwd()
	if err != nil {
		panic(errors.Wrap(err, "quash: WHERE AM I?!"))
	}

	for {
		takeInput()
	}
}

// takeInput takes the user's shell input and runs that command
func takeInput() {
	// read one line of input from Stdin
	fmt.Fprintf(os.Stdout, "\033[94m%s\033[0m:\033[96m%s\033[0m \033[93m%s\033[0m ", "quash", filepath.Base(currDir), "λ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		// If user clicked Ctrl-D, then exit
		if err == io.EOF {
			fmt.Fprintf(os.Stdout, "\n")
			exit()
		}
		// If something happened while reading, spit it out
		fmt.Fprintf(os.Stderr, "quash: %s\n", err.Error())
		return
	}

	// split input into different commands to be executed
	commands := strings.Split(input, "|")
	for index, command := range commands {
		commands[index] = strings.TrimSpace(command)
	}

	// make pipes to communicate between the different processes
	pipeRead := make([]*os.File, len(commands)-1)
	pipeWrite := make([]*os.File, len(commands)-1)
	// actually initiate all the pipes we will need
	for index := range pipeRead {
		pipeRead[index], pipeWrite[index], _ = os.Pipe()
	}

	// fork and execute each command as its own process
	for index, command := range commands {
		//seperate command into its executable name and arguments
		args := strings.Split(command, " ")
		cmdName := args[0]

		// Check if we need to exit
		if cmdName == "exit" || cmdName == "quit" {
			exit()
		}

		// Check if we have the set command
		if cmdName == "set" {
			if len(args) != 2 {
				fmt.Fprintf(os.Stderr, "quash: bad set format")
				return
			}
			parts := strings.Split(args[1], "=")
			if len(parts) != 2 {
				fmt.Fprintf(os.Stderr, "quash: bad set format")
				return
			}
			envName := parts[0]
			envVal := parts[1]
			setenv(envName, envVal)
			return
		}

		// Check if we hit the change directory command
		if cmdName == "cd" {
			if len(args) == 1 {
				// No directory given, switch to HOME
				currDir = getenv("HOME")
				return
			}
			if len(args) != 2 {
				fmt.Fprintf(os.Stdout, "quash: bad cd format")
				return
			}
			dirToChange := filepath.Join(currDir, args[1])
			if filepath.IsAbs(args[1]) {
				dirToChange = args[1]
			}
			// Check if that directory actually exists or not
			_, err := os.Stat(dirToChange)
			if os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "quash: directory doesn't exist: %s", dirToChange)
				return
			}
			currDir = dirToChange
			return
		}

		// see if & present, signifies if program runs in background
		background := strings.Contains(command, "&")
		if background {
			// remove the & from args. Can we guarantee that & is
			// the last command? I believe bash syntax forces this
			args = args[:len(args)-1]
		}

		// find path to executable
		paths, err := exec.LookPath(cmdName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "quash: %s: %s\n", errors.Unwrap(err), cmdName)
			return
		}
		// make actual fork happen
		pid, err := syscall.ForkExec(
			//_, err := syscall.ForkExec(
			paths, args, &syscall.ProcAttr{
				Dir:   currDir,
				Env:   myEnv,
				Files: fileDescriptor(index, pipeRead, pipeWrite),
				//Sys: &syscall.SysProcAttr{Foreground: !background},

				// having trouble setting Foreground to true without program failing
				// to terminate probably extra flags and such that need to be set,
				// but dont know which
				Sys: &syscall.SysProcAttr{},
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
	}
}

// fileDescriptor returns a custom file descriptor for a call to ForkExec
// if there is only one command with no pipes, Stdin Stdout and Stderr are used
// pipes overwrite read, write, or both for processes inside of a pipe chain.
func fileDescriptor(index int, readPipe []*os.File, writePipe []*os.File) []uintptr {
	// One command, so no pipes
	if len(readPipe) == 0 {
		return []uintptr{
			os.Stdin.Fd(),
			os.Stdout.Fd(),
			os.Stderr.Fd(),
		}
	}
	// first in a chain
	if index == 0 {
		return []uintptr{
			os.Stdin.Fd(),
			writePipe[0].Fd(),
			os.Stderr.Fd(),
		}
	}
	// last in a chain
	if index == len(readPipe) {
		return []uintptr{
			readPipe[index-1].Fd(),
			os.Stdout.Fd(),
			os.Stderr.Fd(),
		}
	}
	// middle of a chain
	return []uintptr{
		readPipe[index-1].Fd(),
		writePipe[index].Fd(),
		os.Stderr.Fd(),
	}
}

func getenv(key string) string {
	// Try to find and replace
	for _, env := range myEnv {
		parts := strings.Split(env, "=")
		if parts[0] == key {
			return parts[1]
		}
	}
	// Not found
	return ""
}

func setenv(key, value string) {
	entry := key + "=" + value
	// Try to find and replace
	for ind, env := range myEnv {
		parts := strings.Split(env, "=")
		if parts[0] == key {
			myEnv[ind] = entry
			return
		}
	}
	// If not found, append
	myEnv = append(myEnv, entry)
}

// exit exits quash on "exit" or "quit" or "Ctrl-D"
func exit() {
	fmt.Fprintf(os.Stdout, "exit\n")
	os.Exit(0)
}

// closePipe closes used pipe ends based on where they are in a chain of piped
// commands if only one command exists, there are no pipes and this function
// does nothing.
func closePipe(index int, readPipe []*os.File, writePipe []*os.File) {
	// One command, so no pipes
	if len(readPipe) == 0 {
	} else if index == 0 {
		// first in a chain
		writePipe[0].Close()
	} else if index == len(readPipe) {
		// last in a chain
		readPipe[index-1].Close()
	} else {
		// middle of a chain
		readPipe[index-1].Close()
		writePipe[index].Close()
	}
}
