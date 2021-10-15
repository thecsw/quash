package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	isTerminal = terminal.IsTerminal(int(os.Stdin.Fd()))
)

func main() {
	// Set the current directory
	var err error
	currDir, err = os.Getwd()
	if err != nil {
		panic(errors.Wrap(err, "quash: WHERE AM I?!"))
	}

	// Ignore SIGINT
	signal.Ignore(os.Interrupt)

	// Our reader buffers the input
	reader := bufio.NewReader(os.Stdin)
	for {
		runShell(reader)
	}
}

// runShell takes the user's shell input and runs that command
func runShell(reader *bufio.Reader) {
	// Greet the user if we are in the terminal
	if isTerminal {
		greet()
	}

	// Take the input line from the reader
	input := takeInput(reader)

	// Actually execute the user input
	executeInput(input)
}

// takeInput reads a newline-terminated input from a bufio reader
func takeInput(reader *bufio.Reader) string {
	input, err := reader.ReadString('\n')
	if err != nil {
		// If user clicked Ctrl-D, then exit
		if err == io.EOF {
			if isTerminal {
				fmt.Fprint(os.Stdout, NEWLINE)
			}
			exit(nil)
		}
		// If something happened while reading, spit it out
		quashError("%s", err.Error())
		return NEWLINE
	}
	return input
}

// executeInput takes an input string and runs (attempts) the commands in it.
func executeInput(input string) {
	// If user presses enter, then skip
	if input == NEWLINE {
		return
	}

	// see if & present, signifies if program runs in background
	ampCount := strings.Count(input, "&")
	isBackground := false
	jid := nextJobID
	newJob := job{jid: jid, command: input, processes: make(map[int]*os.Process)}
	if ampCount == 1 {
		isBackground = true
		nextJobID++
		input = strings.TrimSpace(strings.Replace(input, "&", "", 1)) //should probably check that the amp is at the very end?
	} else if ampCount > 1 {
		panic("bad number of ampersands")
		//idk error? its valid for shell commands, but way out of the scope of this project to have & as anything but a foreground/background indicator
	}

	// split input into different commands to be executed
	commands := strings.Split(input, "|")
	for index, command := range commands {
		commands[index] = strings.TrimSpace(command)
	}

	newJob.numProcesses = len(commands)
	pipeRead, pipeWrite := createPipes(len(commands) - 1)

	// fork and execute each command as its own process
	for index, command := range commands {
		var err error
		// Find all of our destinations
		stdinDestination := os.Stdin
		stdoutDestination := os.Stdout
		stderrDestination := os.Stderr
		command, err = setReridects(command,
			&stdinDestination,
			&stdoutDestination,
			&stderrDestination,
		)
		if err != nil {
			quashError("redirect failed", err)
			return
		}

		//seperate command into its executable name and arguments
		args := strings.Split(command, " ")
		cmdName := args[0]

		// See if the command is a built-in shell command
		if builtinFunc, ok := builtins[cmdName]; ok {
			builtinFunc(args)
			return
		}

		// find path to executable
		paths, err := lookPath(cmdName)
		if err != nil {
			quashError("%s : %s", err, cmdName)
			return
		}

		// make actual fork happen
		pid, err := syscall.ForkExec(
			//_, err := syscall.ForkExec(
			paths, args, &syscall.ProcAttr{
				Dir: currDir,
				Env: myEnv,
				Files: fileDescriptor(
					index,
					pipeRead,
					pipeWrite,
					stdinDestination,
					stdoutDestination,
					stderrDestination,
				),
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
		if !isBackground {
			process, _ := os.FindProcess(pid)
			process.Wait()
		} else {
			newJob.pid = append(newJob.pid, pid)

			process, _ := os.FindProcess(pid)
			newJob.processes[pid] = process
			jobList[jid] = newJob
			if index == 0 {
				fmt.Printf("[%d] %d running in background\n", jid, jobList[jid].pid)
			}
		}
	}

	if isBackground {
		go trackChild(jid)
	}
}
