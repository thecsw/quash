package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	input := "date"
	fmt.Printf("> ")
	fmt.Scanf("%s", &input)

	// pid, err := syscall.ForkExec(input, []string{}, &syscall.ProcAttr{})
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(pid)

	cmd := exec.Command(input)
	cmd.Stdout = os.Stdout
	cmd.Run()

}
