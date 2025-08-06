package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	for {
		if _, err := fmt.Fprint(os.Stdout, "$ "); err != nil {
			panic(err)
		}

		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		// Wait for user input
		if err != nil {
			panic(err)
		}

		input = strings.TrimSpace(input)

		fmt.Printf("%s: command not found\n", input)
	}
}
