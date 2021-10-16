package main

import (
	"os"
)

var (
	nextJobID         = 1
	jobList           = make(map[int]job)
	runningProcessPid = make(map[int]int)
)

/*
// trackChild keeps track of jobs that run in the background
// The main goal is printing when the process is created,
// terminates, or is killed
func trackChild(jid int) {
	itr := 0
	for len(jobList[jid].processes) > 0 {
		pid := jobList[jid].pid[itr]
		runningProcessPid[jid] = pid
		state, err := jobList[jid].processes[pid].Wait()
		if err != nil {
			panic(err)
		} else {
			if state.ExitCode() == -1 {
				fmt.Printf("[%d] %d killed by error or signal",
					jobList[jid].jid, jobList[jid].pid[0])
				delete(jobList, jid)
				return
			}
			delete(jobList[jid].processes, pid)
			itr++
		}
	}

	fmt.Printf("[%d] %d finished %s\n",
		jobList[jid].jid, jobList[jid].pid[0],
		jobList[jid].command)
	delete(jobList, jid)
	delete(runningProcessPid, jid)
}
*/

// job is the struct that holds info about background processes
type job struct {
	firstPid int
	// jid associated with this job
	jid int
	// command that created this job
	command string
	// reference to the current process
	process *os.Process
}
