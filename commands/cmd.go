package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

const (
	defaultArgsBuffer              = 8
	defaultCommandIndexCapacity    = 16
	defaultCommandStatusBufferSize = 1024
	defaultFileMode                = 0o644
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
		terminate: false,
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

// func (s CommandStatus) initBuffer() {
// 	s.Stdout = make([]byte, 0, defaultCommandStatusBufferSize)
// }

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

func CreateEmptyFile(path string) (*os.File, error) {
	return os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		defaultFileMode,
	)
}

func CreateAppendFile(path string) (*os.File, error) {
	return os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		defaultFileMode,
	)
}

type CommandIndex struct {
	index map[string]Command
}

func NewCommandIndex() (index *CommandIndex) {
	index = &CommandIndex{
		index: make(
			map[string]Command,
			min(len(commands), defaultCommandIndexCapacity),
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

func ExecCommand(argsStr string) CommandStatus {
	cmdName, cmdArgs := parseCommandArgs(strings.TrimSpace(argsStr))
	cmd, found := GetCommandIndex().Get(cmdName)

	if found {
		return cmd.Exec(cmdArgs)
	} else if cmd, found = findCmdInPath(cmdName); found {
		return cmd.Exec(cmdArgs)
	}

	return newUnknownCommandError(cmdName)
}
