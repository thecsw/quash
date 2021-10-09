package main

import "os"

// createPipes allocates and initializes num pipes, where
// the first is the slice of read pipes, second is writes
func createPipes(num int) ([]*os.File, []*os.File) {
	// make pipes to communicate between the different processes
	pipeRead := make([]*os.File, num)
	pipeWrite := make([]*os.File, num)
	// actually initiate all the pipes we will need
	for index := range pipeRead {
		pipeRead[index], pipeWrite[index], _ = os.Pipe()
	}
	return pipeRead, pipeWrite

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
