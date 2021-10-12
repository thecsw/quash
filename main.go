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
)

var (
	isTerminal = false
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

	stat, _ := os.Stdin.Stat()
	isTerminal = stat.Mode()&os.ModeCharDevice != 0
	for {
		runShell(reader, isTerminal)
	}
}

// runShell takes the user's shell input and runs that command
func runShell(reader *bufio.Reader, isTerminal bool) {
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
	// split input into different commands to be executed
	commands := strings.Split(input, "|")
	for index, command := range commands {
		commands[index] = strings.TrimSpace(command)
	}

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

		jid := nextJobID
		newJob := job{jid: jid, command: command}

		// see if & present, signifies if program runs in background
		isBackground := strings.Contains(command, "&")
		if isBackground {
			// remove the & from args
			args = args[:len(args)-1]
			nextJobID++
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
			process, _ := os.FindProcess(pid)
			newJob.pid = pid
			newJob.process = process
			jobList[jid] = newJob
			fmt.Printf("[%d] %d running in background\n", jid, jobList[jid].pid)
			go trackChild(jid)
		}
	}

}
