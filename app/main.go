package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	if _, err := fmt.Fprint(os.Stdout, "$ "); err != nil {
		panic(err)
	}

	// Wait for user input
	if _, err := bufio.NewReader(os.Stdin).ReadString('\n'); err != nil {
		panic(err)
	}
}
