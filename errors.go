package main

import (
	"fmt"
	"os"
)

const (
	// quashErrPrefix is the prefix for quash errors
	quashErrPrefix = "\033[91mquash: \033[0m"
	// quashErrBadArgs is the error message for bad arguments
	quashErrBadSet = "bad set format"
	// quashErrBadCd is the error message for bad cd
	quashErrBadCd = "bad cd format"
	// quashErrNoDir is the error message for no such directory
	quashErrNoDir = "bad target directory"
	// quashErrBadKill is the error message for bad kill
	quashErrBadKill = "bad kill format"
)

// quashError prints a quash error into Stderr
func quashError(str string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, quashErrPrefix+str+"\n", args...)
}
