package commands

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

const defaultCommandStatusBufferSize = 1024

type CommandStatus struct {
	code      int
	err       error
	terminate bool
	Stdout    []byte
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

func newNotFoundError(name string) CommandStatus {
	return CommandStatus{
		code:      1,
		err:       fmt.Errorf("%s: not found", name),
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

func (s CommandStatus) initBuffer() {
	s.Stdout = make([]byte, 0, defaultCommandStatusBufferSize)
}

func findCmdInPath(name string) (cmd CmdGeneric, found bool) {
	// for dir := range strings.SplitSeq(os.Getenv("PATH"), ":") {
	// 	entries, err := os.ReadDir(dir)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	for _, entry := range entries {
	// 		info, err := entry.Info()
	// 		if err != nil || info.Mode()&0o111 == 0 || info.Name() != name {
	// 			continue
	// 		}
	// 		return path.Join(dir, info.Name()), true
	// 	}
	// }
	cmdPath, err := exec.LookPath(name)

	if found = err == nil; found {
		cmd = newCmdGeneric(name, cmdPath)
	}

	return
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

func (i *CommandIndex) Find(name string) (found bool) {
	_, found = i.index[name]

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

	if found {
		return cmd.Exec(args)
	} else if cmd, found = findCmdInPath(name); found {
		return cmd.Exec(args)
	}

	return newUnknownCommandError(name)
}
