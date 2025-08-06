package commands

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Command interface {
	Name() string
	Exec(args string) (value CommandStatus)
}

type CmdGeneric struct {
	name string
	path string
}

func newCmdGeneric(cmdName, cmdPath string) CmdGeneric {
	return CmdGeneric{
		name: cmdName,
		path: cmdPath,
	}
}

func (c CmdGeneric) Name() string {
	return c.name
}

func (c CmdGeneric) Exec(args string) (value CommandStatus) {
	cmd := exec.Command(c.name, strings.Split(args, " ")...)
	cmd.Path = c.path

	value.Stdout, value.err = cmd.Output()
	if value.err != nil {
		value.code = 1
	}

	return
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
	value.Stdout = []byte(args + "\n")

	return
}

type CmdType struct{}

func (c CmdType) Name() string {
	return "type"
}

func (c CmdType) Exec(args string) (value CommandStatus) {
	value.initBuffer()

	if GetCommandIndex().Find(args) {
		value.Stdout = fmt.Appendf(
			value.Stdout,
			"%s is a shell builtin\n",
			args,
		)
	} else if path, found := findCmdInPath(args); found {
		value.Stdout = fmt.Appendf(value.Stdout, "%s is %s\n", args, path)
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
