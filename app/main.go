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

		if input == "\n" {
			continue
		}

		status := commands.ExecCommand(input)

		// if status.Stdout != nil {
		// 	fmt.Print(string(status.Stdout))
		// }

		if status.Failed() {
			fmt.Println(status.Error())
		}

		if exit, code := status.Exit(); exit {
			os.Exit(code)
		}
	}
}
