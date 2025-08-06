package commands

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

type Command interface {
	Name() string
	Exec(args string) (value CommandStatus)
}

type CmdExit struct{}

func (c CmdExit) Name() string {
	return "exit"
}

var cmdExitErr = newGenericStatusError(
	fmt.Errorf("exit requires one integer parameter"),
)

func (c CmdExit) Exec(args string) (value CommandStatus) {
	value.terminate = true
	if value.code, value.err = strconv.Atoi(args); value.err != nil {
		return cmdExitErr
	}

	return
}

type CmdEcho struct{}

func (c CmdEcho) Name() string {
	return "echo"
}

func (c CmdEcho) Exec(args string) (value CommandStatus) {
	fmt.Println(args)

	return
}

type CmdType struct{}

func (c CmdType) Name() string {
	return "type"
}

func findCmdInPath(name string) (cmdPath string, found bool) {
	for dir := range strings.SplitSeq(os.Getenv("PATH"), ":") {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil || info.Mode()&0o111 == 0 || info.Name() != name {
				continue
			}

			return path.Join(dir, info.Name()), true
		}
	}

	return
}

func (c CmdType) Exec(args string) (value CommandStatus) {
	if GetCommandIndex().Find(args) {
		fmt.Printf("%s is a shell builtin\n", args)
	} else if path, found := findCmdInPath(args); found {
		fmt.Printf("%s is %s\n", args, path)
	} else {
		value = newNotFoundError(args)
	}

	return
}

var commands = [...]Command{
	CmdExit{},
	CmdEcho{},
	CmdType{},
}
