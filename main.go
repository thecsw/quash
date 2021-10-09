package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

const (
	quashErrPrefix  = "\033[91mquash: \033[0m"
	quashErrBadSet  = "bad set format"
	quashErrBadCd   = "bad cd format"
	quashErrNoDir   = "bad target directory"
	quashErrBadKill = "bad kill format"
)

var (
	currDir   = ""
	myEnv     = os.Environ()
	nextJobID = int(1)
	jobList   = make(map[int]job)

	stdinFileRegex  = regexp.MustCompile(`<\s*([^<> ]+)`)
	stdoutFileRegex = regexp.MustCompile(`[^2]?>\s*([^<> ]+)`)
	stderrFileRegex = regexp.MustCompile(`2>\s*([^<> ]+)`)
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
	// The print format is "quash:DIRNAME λ"
	fmt.Fprintf(
		os.Stdout,
		"\033[94m%s\033[0m:\033[96m%s\033[0m \033[93m%s\033[0m ",
		"quash",
		filepath.Base(currDir),
		"λ",
	)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		// If user clicked Ctrl-D, then exit
		if err == io.EOF {
			fmt.Fprintf(os.Stdout, "\n")
			exit()
		}
		// If something happened while reading, spit it out
		quashError("%s", err.Error())
		return
	}

	if input == "\n" {
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

		// Find all of our destinations
		stdinDestination := os.Stdin
		stdoutDestination := os.Stdout
		stderrDestination := os.Stderr

		// See if we are redirecting to an error file with "2>"
		isStderrRedirected := strings.Contains(command, "2>")
		if isStderrRedirected {
			matches := stderrFileRegex.FindAllStringSubmatch(command, -1)
			if len(matches) < 1 {
				quashError("bad stderr redirect")
				return
			}
			filename := matches[0][1]
			errfile, err := os.Create(filename)
			defer errfile.Close()
			if err != nil {
				quashError("couldn't open stderr file %s", err.Error())
				return
			}
			stderrDestination = errfile
			command = stderrFileRegex.ReplaceAllString(command, "")
		}

		// See if we are redirecting to a file with ">" or "1>"
		// Make sure we're not matching the error redirect with "2>"
		isStdoutRedirected := strings.Contains(command, ">")
		if isStdoutRedirected {
			matches := stdoutFileRegex.FindAllStringSubmatch(command, -1)
			if len(matches) < 1 {
				quashError("bad stdout redirect")
				return
			}
			filename := matches[0][1]
			outfile, err := os.Create(filename)
			defer outfile.Close()
			if err != nil {
				quashError("couldn't open stdout file %s", err.Error())
				return
			}
			stdoutDestination = outfile
			command = stdoutFileRegex.ReplaceAllString(command, "")
		}

		isStdinRedirected := strings.Contains(command, "<")
		if isStdinRedirected {
			matches := stdinFileRegex.FindAllStringSubmatch(command, -1)
			if len(matches) < 1 {
				quashError("bad stdin redirect")
				return
			}
			filename := matches[0][1]
			infile, err := os.Open(filename)
			defer infile.Close()
			if err != nil {
				quashError("couldn't open stdin file: %s", err.Error())
				return
			}
			stdinDestination = infile
			command = stdinFileRegex.ReplaceAllString(command, "")
		}

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
				quashError(quashErrBadSet)
				return
			}
			parts := strings.Split(args[1], "=")
			if len(parts) != 2 {
				quashError(quashErrBadSet)
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
				quashError(quashErrBadCd)
				return
			}
			// Join our current directory with the relative one
			dirToChange := filepath.Join(currDir, args[1])
			// If absolute path given, switch to it absolutely
			if filepath.IsAbs(args[1]) {
				dirToChange = args[1]
			}
			// Check if that directory actually exists or not
			_, err := os.Stat(dirToChange)
			if os.IsNotExist(err) {
				quashError(quashErrNoDir+": %s", dirToChange)
				return
			}
			currDir = dirToChange
			return
		}

		if cmdName == "kill" {
			if len(args) != 3 {
				quashError(quashErrBadKill)
				return
			}
			sig, err1 := strconv.Atoi(args[1])
			jobID, err2 := strconv.Atoi(args[2])
			if err1 != nil || err2 != nil {
				quashError("Incorrect usage for kill command : SIGNUM and JOBID must be integers\n")
				return
			}
			killed, ok := jobList[jobID]
			if !ok {
				quashError("Job number %d does not exist", jobID)
				return
			}

			err := killed.process.Signal(syscall.Signal(sig))
			if err != nil {
				panic(err)
			}
			return
		}

		if cmdName == "jobs" {
			for _, v := range jobList {
				fmt.Printf("[%d] %d %s\n", v.jid, v.pid, v.command)
			}
			return
		}

		jid := nextJobID
		var newJob = job{jid: jid, command: command}

		// see if & present, signifies if program runs in background
		background := strings.Contains(command, "&")
		if background {
			// remove the & from args
			args = args[:len(args)-1]
			nextJobID += 1
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
		if !background {
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

// fileDescriptor returns a custom file descriptor for a call to ForkExec
// if there is only one command with no pipes, Stdin Stdout and Stderr are used
// pipes overwrite read, write, or both for processes inside of a pipe chain.
func fileDescriptor(
	index int,
	readPipe []*os.File,
	writePipe []*os.File,
	in *os.File,
	out *os.File,
	err *os.File,
) []uintptr {
	// One command, so no pipes
	if len(readPipe) == 0 {
		return []uintptr{
			in.Fd(),
			out.Fd(),
			err.Fd(),
		}
	}
	// first in a chain
	if index == 0 {
		return []uintptr{
			in.Fd(),
			writePipe[0].Fd(),
			err.Fd(),
		}
	}
	// last in a chain
	if index == len(readPipe) {
		return []uintptr{
			readPipe[index-1].Fd(),
			out.Fd(),
			err.Fd(),
		}
	}
	// middle of a chain
	return []uintptr{
		readPipe[index-1].Fd(),
		writePipe[index].Fd(),
		err.Fd(),
	}
}

// getenv gets an env value from myEnv
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

// quashError prints a quash error into Stderr
func quashError(str string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, quashErrPrefix+str+"\n", args...)
}

// setenv sets an env by key in myEnv
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

// lookPath tries to find an absolute path to an executable name by searching directories on the PATH.
// If the name is an absolute path or a shortened path (./) then this path is returned
func lookPath(name string) (string, error) {
	if filepath.IsAbs(name) { //if the user has absolute path then we good
		return name, nil
	}

	absPath := filepath.Join(currDir, name)
	_, err := os.Stat(absPath)
	if !os.IsNotExist(err) {
		return absPath, nil
	}

	// if strings.Index(name, "./") == 0 { //if the user uses ./ as a shortcut to currDir. Still a predefined path so we good
	// 	name = strings.Replace(name, ".", currDir, 1) // ./ becomes /.../name
	// 	return name, nil
	// }
	path := getenv("PATH")
	if path == "" {
		err := errors.New("executable not found")
		return "", err
	}
	directories := strings.Split(path, ":")
	for _, directory := range directories {
		dirInfo, err := os.ReadDir(directory)
		if err != nil {
			quashError("%s : %s", errors.Unwrap(err), directory)
		}
		for _, file := range dirInfo {
			if file.Name() == name && !file.IsDir() {
				return directory + "/" + name, nil
			}
		}
	}
	err = errors.New("executable not found")
	return "", err

}

// trackChild keeps track of jobs that run in the background.
// The main goal is printing when the process is created, terminates, or is killed
func trackChild(jid int) {
	state, err := jobList[jid].process.Wait()
	if err != nil {
		panic(err)
	}

	if state.ExitCode() == 0 {
		fmt.Printf("[%d] %d finished %s\n", jobList[jid].jid, jobList[jid].pid, jobList[jid].command)
	} else if state.ExitCode() == -1 {
		fmt.Printf("[%d] %d killed by error or signal", jobList[jid].jid, jobList[jid].pid)
	}
	delete(jobList, jid)
}

type job struct {
	pid     int         //pid associated with running process
	jid     int         //jid associated with this job
	command string      //the command that created this job
	process *os.Process //a reference to the running process
}
