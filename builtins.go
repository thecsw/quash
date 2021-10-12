package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

var (
	builtins = map[string]func(args []string){
		"exit": exit,
		"quit": exit,
		"set":  setVariable,
		"cd":   changeDirectory,
		"kill": killJob,
		"jobs": showJobs,
	}
)

// changeDirectory updates our current directory
func changeDirectory(args []string) {
	// cd should only have 2 strings: "cd DIR"
	if len(args) > 2 {
		quashError(quashErrBadCd)
		return
	}
	// No directory given, switch to HOME
	newCurrDir := getenv("HOME")
	// If 2 args, join the currDir with proposed path
	if len(args) > 1 {
		// Join our current directory with the relative one
		newCurrDir = filepath.Join(currDir, args[1])
		// If absolute path given, switch to it absolutely
		if filepath.IsAbs(args[1]) {
			newCurrDir = args[1]
		}
	}
	// Check if that directory actually exists or not
	if _, err := os.Stat(newCurrDir); os.IsNotExist(err) {
		quashError(quashErrNoDir+": %s", newCurrDir)
		return
	}
	currDir = newCurrDir
}

// killJob sends a signal kill to the job we have
func killJob(args []string) {
	// the proper format is "kill SIGNUM JOBID"
	if len(args) != 3 {
		quashError(quashErrBadKill)
		return
	}
	// Actually try to parse what SIGNUM and JOBID are
	sig, err1 := strconv.Atoi(args[1])
	jobID, err2 := strconv.Atoi(args[2])
	if err1 != nil || err2 != nil {
		quashError("Incorrect usage for kill command : SIGNUM and JOBID must be integers\n")
		return
	}
	// See whether we have records of the job
	killed, ok := jobList[jobID]
	if !ok {
		quashError("Job number %d does not exist", jobID)
		return
	}
	// Try to send the signal to the job
	if err := killed.process.Signal(syscall.Signal(sig)); err != nil {
		quashError("Couldn't kill %d: %s", jobID, err.Error())
		return
	}
}

// showJobs shows all the jobs we have in the background and running
func showJobs(args []string) {
	for _, v := range jobList {
		fmt.Printf("[%d] %d %s\n", v.jid, v.pid, v.command)
	}
}

// setVariable sets an environmental variable with "set" command
func setVariable(args []string) {
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
}

// exit exits quash on "exit" or "quit" or "Ctrl-D"
func exit(args []string) {
	if isTerminal {
		fmt.Fprintf(os.Stdout, "exit\n")
	}
	os.Exit(0)
}
