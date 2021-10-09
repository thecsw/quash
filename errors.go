package main

import (
	"fmt"
	"os"
)

const (
	quashErrPrefix  = "\033[91mquash: \033[0m"
	quashErrBadSet  = "bad set format"
	quashErrBadCd   = "bad cd format"
	quashErrNoDir   = "bad target directory"
	quashErrBadKill = "bad kill format"
)

// quashError prints a quash error into Stderr
func quashError(str string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, quashErrPrefix+str+"\n", args...)
}
