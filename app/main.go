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
		case keys.Enter, keys.Tab, keys.Up, keys.Down:
			return true, nil
		case keys.CtrlC:
			return true, nil
		default:
			if key.String() != "" {
				input.WriteString(key.String())
				// fmt.Print(key.String())
			}
		}

		return false, nil
	})

	return input.String(), lastKey
}

func main() {
	for {
		if _, err := fmt.Fprint(os.Stdout, "$ "); err != nil {
			panic(err)
		}

		// input, err := bufio.NewReader(os.Stdin).ReadString('\n')
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
