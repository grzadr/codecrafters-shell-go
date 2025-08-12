package main

import (
	"fmt"
	"os"
	"strings"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/codecrafters-io/shell-starter-go/commands"
)

const (
	ClearLine  = "\033[2K" // Clear entire line
	MoveCursor = "\033[0G" // Move cursor to column 0
	BellRing   = "\x07"
)

func readUntilTerminator() (string, keys.KeyCode) {
	var input strings.Builder

	var lastKey keys.KeyCode

	var lastMatched commands.CmdPrefixMatched

	keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		switch lastKey = key.Code; lastKey {
		case keys.CtrlD:
			os.Exit(0)
		case keys.Up:
			input.Reset()
			input.WriteString(commands.GetCommandHistory().Prev())
			fmt.Printf("%s%s$ %s", ClearLine, MoveCursor, input.String())
		case keys.Down:
			input.Reset()
			input.WriteString(commands.GetCommandHistory().Next())
			fmt.Printf("%s%s$ %s", ClearLine, MoveCursor, input.String())
		case keys.Tab:
			index := commands.GetCommandsIndex()
			prefix := input.String()

			if name, found := index.Match(prefix); found {
				input.Reset()
				input.WriteString(name + " ")
				fmt.Printf("%s%s$ %s", ClearLine, MoveCursor, input.String())
			} else if lastMatched != nil {
				if name, found := lastMatched.FindClosest(prefix); found {
					input.Reset()
					input.WriteString(name)
					fmt.Printf("%s%s$ %s", ClearLine, MoveCursor, input.String())
				} else {
					fmt.Printf("\n%s\n$ %s", strings.Join(lastMatched, "  "), prefix)
				}
			} else if lastMatched = index.MatchInPath(prefix); len(lastMatched) > 0 {
				if name, found := lastMatched.FindClosest(prefix); found {
					input.Reset()
					input.WriteString(name)
					fmt.Printf("%s%s$ %s", ClearLine, MoveCursor, input.String())
				} else {
					fmt.Print(BellRing)
				}
			} else {
				lastMatched = nil

				fmt.Print(BellRing)
			}

		case keys.Enter, keys.CtrlJ:
			fmt.Println()

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
			panic(key)
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
