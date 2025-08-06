package commands

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type CommandStatus struct {
	code      int
	err       error
	terminate bool
}

func newGenericStatusError(err error) CommandStatus {
	return CommandStatus{
		code:      1,
		err:       err,
		terminate: true,
	}
}

func newUnknownCommandError(name string) CommandStatus {
	return CommandStatus{
		code:      1,
		err:       fmt.Errorf("%s: command not found", name),
		terminate: false,
	}
}

func (s CommandStatus) Failed() bool {
	return s.err != nil
}

func (s CommandStatus) Error() string {
	return s.err.Error()
}

func (s CommandStatus) Exit() (bool, int) {
	return s.terminate, s.code
}

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

var commands = [...]Command{
	CmdExit{},
	CmdEcho{},
}

const commandIndexDefaultCapacity = 16

type CommandIndex struct {
	index map[string]Command
}

func NewCommandIndex() (index *CommandIndex) {
	index = &CommandIndex{
		index: make(
			map[string]Command,
			min(len(commands), commandIndexDefaultCapacity),
		),
	}
	for _, cmd := range commands {
		index.index[cmd.Name()] = cmd
	}

	return
}

func (i *CommandIndex) Get(name string) (cmd Command, found bool) {
	cmd, found = i.index[name]

	return
}

var (
	commandIndex     *CommandIndex
	commandIndexOnce sync.Once
)

func GetCommandIndex() *CommandIndex {
	commandIndexOnce.Do(func() {
		commandIndex = NewCommandIndex()
	})

	return commandIndex
}

func parseCommandString(cmdStr string) (name, args string) {
	name, args, _ = strings.Cut(strings.TrimSpace(cmdStr), " ")

	return
}

func ExecCommand(cmdStr string) CommandStatus {
	name, args := parseCommandString(cmdStr)
	cmd, found := GetCommandIndex().Get(name)

	if !found {
		return newUnknownCommandError(name)
	}

	return cmd.Exec(args)
}
