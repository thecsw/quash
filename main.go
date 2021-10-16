package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

var (
	isTerminal = terminal.IsTerminal(int(os.Stdin.Fd()))
	// currJob is the pid of the current foreground job
	currJob = int(0)
	// sigintChan listens to SIGINT and signals currJob
	sigintChan = make(chan os.Signal, 1)
	// sigints is a slice of signals corresponding to Ctrl-C
	sigints = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGINT}
)

func main() {
	// Set the current directory
	var err error
	currDir, err = os.Getwd()
	if err != nil {
		quashError("couldn't pwd, defaulting to /:", err.Error())
		currDir = "/"
	}

	// Ignore SIGINT
	signal.Ignore(sigints...)
	signal.Notify(sigintChan, sigints...)
	go jobStopper()

	// Show a warm welcoming message
	if isTerminal {
		hello()
	}

	// Our reader buffers the input
	reader := bufio.NewReader(os.Stdin)
	for {
		runShell(reader)
	}
}

// runShell takes the user's shell input and runs that command
func runShell(reader *bufio.Reader) {
	// Update the current job
	currJob = 0

	// Greet the user if we are in the terminal
	if isTerminal {
		greet()
	}

	// Actually execute the user input
	executeInput(expandEnv(takeInput(reader)))
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

	// Bad ampersand usage
	if ampCount > 1 {
		// idk error? its valid for shell commands, but way out of
		// the scope of this project to have & as anything but a foreground/background indicator
		quashError("bad number of ampersands")
		return
	}
	// We have to send a job to background
	if ampCount == 1 {
		isBackground = true
		nextJobID++
		//should probably check that the amp is at the very end?
		input = strings.TrimSpace(strings.Replace(input, "&", "", 1))
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
				// having trouble setting Foreground to true without program failing
				// to terminate probably extra flags and such that need to be set,
				// but dont know which
				Sys: &syscall.SysProcAttr{
					// Setsid allows us to ignore Ctrl-C in background processes
					Setsid: true,
				},
			})
		if err != nil {
			quashError("failed to fork: %s", err.Error())
			return
		}
		if !isBackground {
			currJob = pid
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
