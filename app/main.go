package main

import (
	"fmt"
	"os"
	"strings"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/codecrafters-io/shell-starter-go/commands"
)

func readUntilTerminator() (string, keys.KeyCode) {
	var input strings.Builder

	var lastKey keys.KeyCode

	keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		switch lastKey = key.Code; lastKey {
		case keys.CtrlD:
			os.Exit(0)
		case keys.Enter, keys.Tab, keys.Up, keys.Down:
			return true, nil
		case keys.Space:
			input.WriteRune(' ')
			fmt.Print(" ")
		case keys.RuneKey:
			input.WriteString(key.String())
			fmt.Print(key.String())

			if strings.HasSuffix(input.String(), "\n") {
				return true, nil
			}

		default:
			if len(key.Runes) == 1 {
				input.WriteString(string(key.Runes))
				fmt.Print(string(key.Runes))
			}
		}

		return false, nil
	})

	// fmt.Fprint(os.Stdout, input.String())

	return input.String(), lastKey
}

func main() {
	for {
		if _, err := fmt.Fprint(os.Stdout, "$ "); err != nil {
			panic(err)
		}

		// input := readArrowKey()

		// input, err := bufio.NewReader(os.Stdin).ReadByte()
		// if err != nil {
		// 	panic(err)
		// }

		input, _ := readUntilTerminator()

		if input == "" {
			continue
		}

		status := commands.ExecCommand(input)

		if exit, code := status.Exit(); exit {
			os.Exit(code)
		}
	}
}
