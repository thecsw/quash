package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unicode"

	"github.com/eiannone/keyboard"
)

// takeInput reads a newline-terminated input from a bufio reader
func takeInput(reader *bufio.Reader) string {
	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	cmdNum := len(goodHistory)
	var readCharacter rune
	input := ""
	curPosition := 0

	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			quashError("bad input: %s", err.Error())
		}
		readCharacter = char

		// See what key we actually pressed, I tried doing switch
		// but it works kinda wonky. If statements forever <3
		// --------------------------------------------------

		// On enter, flush a newline and return whatever we have
		if key == keyboard.KeyEnter {
			fmt.Fprint(os.Stdout, NEWLINE)
			return input + string(char)
		}
		// On Ctrl-D or Escape just close the shell altogether
		if key == keyboard.KeyEsc {
			if isTerminal {
				fmt.Fprint(os.Stdout, NEWLINE)
			}
			exit(nil)
		}
		// Only exit on Ctrl-D if input is empty
		if key == keyboard.KeyCtrlD {
			if curPosition != 0 || len(input) != 0 {
				continue
			}
			if isTerminal {
				fmt.Fprint(os.Stdout, NEWLINE)
			}
			exit(nil)
		}
		// On a space just set readCharacter to a space run
		if key == keyboard.KeySpace {
			readCharacter = ' '
		}
		// On backspace, move cursor to the left, clean character,
		// and move the cursor again to the left. Delete last input element
		if key == keyboard.KeyBackspace || key == keyboard.KeyBackspace2 {
			// If cursor is already at the home position, don't move
			if curPosition < 1 {
				continue
			}
			fmt.Fprintf(os.Stdout, "\b \b")
			input = input[:curPosition-1]
			curPosition--
			continue
		}
		// On arrow up press, clean out the terminal and replace the user input
		// with whatever previous good command we can find. Works on multiple
		// arrow up key presses too
		if key == keyboard.KeyArrowUp {
			if len(goodHistory) < 1 {
				continue
			}
			// Clear the input first
			resetTermInput(len(input))
			cmdNum = prevCmdNum(cmdNum)
			input = printOldGoodCommand(cmdNum)
			curPosition = len(input)
			continue
		}
		// On arrow down press, clean out the terminal and replace with whatever
		// command came after. Only makes sense if run after one or mory presses
		// of the arrow up key. On the bottom it will set user input to just clean
		if key == keyboard.KeyArrowDown {
			if len(goodHistory) < 1 {
				continue
			}
			resetTermInput(len(input))
			// If at the end of history, just clear the input
			if cmdNum >= len(goodHistory)-1 {
				input = ""
				cmdNum = len(goodHistory)
				curPosition = 0
				continue
			}
			// Get the later good command
			cmdNum = nextCmdNum(cmdNum)
			input = printOldGoodCommand(cmdNum)
			curPosition = len(input)
			continue
		}
		// Ignore left and right arrow keys
		if key == keyboard.KeyArrowLeft || key == keyboard.KeyArrowRight {
			continue
		}
		// Send kill signals if ctrl is encountered or clear the input
		if key == keyboard.KeyCtrlC {
			// Don't do anything if we have an empty command
			if curPosition == 0 && len(input) == 0 {
				sigintChan <- syscall.SIGINT
				continue
			}
			fmt.Fprintf(os.Stdout, "\033[41m^C\033[0m\n")
			input = ""
			curPosition = 0
			greet()
			continue
		}
		// Ctrl-L should clear the screen
		if key == keyboard.KeyCtrlL {
			executeInput("clear")
			greet()
			// Reprint whatever we had before
			fmt.Fprintf(os.Stdout, "%s", input)
			continue
		}
		// If the character is NOT printable, skip saving it
		if !unicode.IsPrint(readCharacter) {
			continue
		}
		// Print the character that we swallowed up and append to input
		fmt.Fprint(os.Stdout, string(readCharacter))
		input += string(readCharacter)
		curPosition = len(input)
	}

	// input, err := reader.ReadString('\n')
	// if err != nil {
	// 	// If user clicked Ctrl-D, then exit
	// 	if err == io.EOF {
	// 		if isTerminal {
	// 			fmt.Fprint(os.Stdout, NEWLINE)
	// 		}
	// 		exit(nil)
	// 	}
	// 	// If something happened while reading, spit it out
	// 	quashError("%s", err.Error())
	// 	return NEWLINE
	// }
	// return input
}

// printOldGoodCommand prints the old good command and returns it
func printOldGoodCommand(cmdNum int) string {
	// If outside of bounds, return empty just in case
	if cmdNum < 0 || cmdNum >= len(goodHistory) {
		return ""
	}
	command := strings.TrimSpace(goodHistory[cmdNum])
	fmt.Fprint(os.Stdout, command)
	return command
}

// resetTermInput resets the terminal input
func resetTermInput(what int) {
	// Wipe out the user input AND the greeting
	// re-greet them later
	//
	// printN(what+greetLength, "\b")
	// printN(what+greetLength, " ")
	// printN(what+greetLength, "\b")
	// greet()

	// Only wipe out the user input
	//
	printN(what, "\b")
	printN(what, " ")
	printN(what, "\b")
}

// prevCmdNum gives last good command index
func prevCmdNum(cmdNum int) int {
	if cmdNum == 0 {
		return cmdNum
	}
	return cmdNum - 1
}

// nextCmdNum gives next good command index
func nextCmdNum(cmdNum int) int {
	if cmdNum == len(goodHistory)-1 {
		return cmdNum
	}
	return cmdNum + 1
}

// printN prints string N times
func printN(what int, str string) {
	finalPrint := ""
	for i := 0; i < what; i++ {
		finalPrint += str
	}
	fmt.Fprint(os.Stdout, finalPrint)
}
