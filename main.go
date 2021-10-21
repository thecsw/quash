package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

var (
	// isTerminal is a flag that shows us if we are in tty
	isTerminal = terminal.IsTerminal(int(os.Stdin.Fd()))
	// sigintChan listens to SIGINT and signals the current bg job to die
	sigintChan = make(chan os.Signal, 1)
	// sigints is a slice of signals corresponding to Ctrl-C
	sigints = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGINT}
	// goodHistory is a history of good past commands
	goodHistory = []string{}
	// greetLength is the number of bytes our greeting takes
	greetLength int
)

func main() {
	// Set the current directory
	var err error
	currDir, err = os.Getwd()
	if err != nil {
		quashError("couldn't pwd, defaulting to /:", err.Error())
		currDir = "/"
	}

	initFlags()
	flag.Parse()

	// Show version and exit
	if flagVersion {
		fmt.Fprintf(os.Stdout, "quash, version 9000\n")
		return
	}

	// Ignore SIGINT
	//signal.Ignore(sigints...)
	signal.Notify(sigintChan, sigints...)
	go jobStopper()

	// Show a warm welcoming message and set shell line
	if isTerminal {
		hello()
		// The shell's theme
		setenv("QUASH", "quash")
		setenv("TERM", "xterm")
	}

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

	// Actually execute the user input
	executeInput(expandEnv(cleanInput(takeInput(reader))))
}

// executeInput takes an input string and runs (attempts) the commands in it.
func executeInput(input string) {
	// If user presses enter, then skip
	if input == NEWLINE || len(input) < 1 {
		return
	}

	// see if & present, signifies if program runs in background
	ampCount := strings.Count(input, "&")
	//isBackground := false

	// Bad ampersand usage
	if ampCount > 1 {
		// idk error? its valid for shell commands, but way out of
		// the scope of this project to have & as anything but a foreground/background indicator
		quashError("bad number of ampersands")
		return
	}
	// We have to send a job to background
	if ampCount == 1 {
		go backgroundExecution(input)
		addToHistory(input)
		return
	}

	// split input into different commands to be executed
	commands := strings.Split(input, "|")
	for index, command := range commands {
		commands[index] = strings.TrimSpace(command)
		args := strings.Split(commands[index], " ")
		args[0] = strings.TrimSpace(args[0])
		if builtinFunc, ok := builtins[args[0]]; ok && len(commands) == 1 {
			builtinFunc(args)
			addToHistory(input)
			return
		} else if ok {
			quashError("built-in command inside pipe chain")
			return
		}

	}

	pipeRead, pipeWrite := createPipes(len(commands) - 1)

	// fork and execute each command as its own process
	for index, command := range commands {
		pid, err := executeCommand(command, index, pipeRead, pipeWrite)
		if err != nil {
			quashError("failed to execute command (%s): %s", command, err.Error())
			return
		}

		// close pipes that have been used to prevent stalling
		closePipe(index, pipeRead, pipeWrite)

		process, _ := os.FindProcess(pid)
		process.Wait()
	}
	addToHistory(input)
}

func executeCommand(command string, index int, pipeRead []*os.File, pipeWrite []*os.File) (int, error) {
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
		//quashError("redirect failed", err)
		//return -1, fmt.Errorf("redirect failed: %w", err)
		return -1, err
	}

	//seperate command into its executable name and arguments
	args := strings.Split(command, " ")
	cmdName := args[0]

	// find path to executable
	paths, err := lookPath(cmdName)
	if err != nil {
		//quashError("%s : %s", err, cmdName)
		return -1, err
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
		//quashError("failed to fork: %s", err.Error())
		return -1, err
	}
	return pid, nil
}

func backgroundExecution(input string) {
	jid := nextJobID
	newJob := job{jid: jid, command: input}
	input = strings.TrimSpace(strings.Replace(input, "&", "", 1))
	//go trackChild(jid)

	// split input into different commands to be executed
	commands := strings.Split(input, "|")
	for index, command := range commands {
		commands[index] = strings.TrimSpace(command)
		args := strings.Split(commands[index], " ")
		args[0] = strings.TrimSpace(args[0])
		if builtinFunc, ok := builtins[args[0]]; ok && len(commands) == 1 {
			builtinFunc(args)
			addToHistory(input)
			return
		} else if ok {
			quashError("built-in command inside pipe chain")
			return
		}
	}

	pipeRead, pipeWrite := createPipes(len(commands) - 1)

	// fork and execute each command as its own process
	for index, command := range commands {
		pid, err := executeCommand(command, index, pipeRead, pipeWrite)
		if err != nil {
			quashError("failed to execute command: %s", err.Error())
			return
		}

		// close pipes that have been used to prevent stalling
		closePipe(index, pipeRead, pipeWrite)

		process, _ := os.FindProcess(pid)
		newJob.process = process
		newJob.pid = pid
		jobList[jid] = newJob
		if index == 0 {
			//runningProcessPid[jid] = pid
			nextJobID++ // job succesfully started so increment jid counter
			fmt.Printf("[%d] %d running in background\n", jid, pid)
		}

		state, err := process.Wait()
		if err != nil {
			panic(err)
		} else if state.ExitCode() == -1 {
			fmt.Printf("[%d] %d killed by error or signal",
				jobList[jid].jid, jobList[jid].pid)
			delete(jobList, jid)
			//delete(runningProcessPid, jid)
			return
		}
	}
	fmt.Printf("[%d] %d finished %s\n",
		jobList[jid].jid, jobList[jid].pid,
		jobList[jid].command)
	delete(jobList, jid)
	//delete(runningProcessPid, jid)
}

// addToHistory adds a successful command to the current history
func addToHistory(what string) {
	goodHistory = append(goodHistory, what)
}

// cleanInput cleans a string from null bytes
func cleanInput(input string) string {
	toReturn := ""
	for _, c := range input {
		if byte(c) == 0 {
			continue
		}
		toReturn += string(c)
	}
	return toReturn
}
