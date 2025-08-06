package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/codecrafters-io/shell-starter-go/commands"
)

func main() {
	for {
		if _, err := fmt.Fprint(os.Stdout, "$ "); err != nil {
			panic(err)
		}

		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			panic(err)
		}

		// input = strings.TrimSpace(input)

		// fmt.Printf("%s: command not found\n", input)

		status := commands.ExecCommand(input)

		if status.Failed() {
			fmt.Println(status.Error())
		}

		if exit, code := status.Exit(); exit {
			os.Exit(code)
		}
	}
}
