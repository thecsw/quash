package main

import (
	"fmt"
	"os"
)

var (
	nextJobID = 1
	jobList   = make(map[int]job)
)

// trackChild keeps track of jobs that run in the background
// The main goal is printing when the process is created,
// terminates, or is killed
func trackChild(jid int) {
	state, err := jobList[jid].process.Wait()
	if err != nil {
		panic(err)
	}

	switch state.ExitCode() {
	case 0:
		fmt.Printf("[%d] %d finished %s\n",
			jobList[jid].jid, jobList[jid].pid,
			jobList[jid].command)
	case -1:
		fmt.Printf("[%d] %d killed by error or signal",
			jobList[jid].jid, jobList[jid].pid)
	}
	delete(jobList, jid)
}

// job is the struct that holds info about background processes
type job struct {
	// pid associated with running process
	pid int
	// jid associated with this job
	jid int
	// command that created this job
	command string
	// process references to the running process
	process *os.Process
}
