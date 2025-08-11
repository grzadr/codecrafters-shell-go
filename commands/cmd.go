package commands

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
)

const (
	defaultArgsBuffer              = 8
	defaultCommandIndexCapacity    = 16
	defaultCommandStatusBufferSize = 1024
	defaultFileMode                = 0o644
)

type CommandStatus struct {
	code int
	// err       error
	terminate bool
}

func newErrorStatus() CommandStatus {
	return CommandStatus{code: 1}
}

// func newUnknownCommandError(name string) CommandStatus {
// 	return CommandStatus{
// 		code:      1,
// 		err:       fmt.Errorf("%s: command not found", name),
// 		terminate: false,
// 	}
// }

// func newNotFoundError(name string) CommandStatus {
// 	return CommandStatus{
// 		code:      1,
// 		err:       fmt.Errorf("%s: not found", name),
// 		terminate: false,
// 	}
// }

func (s CommandStatus) Failed() bool {
	return s.code != 0
}

// func (s CommandStatus) Error() string {
// 	return s.err.Error()
// }

func (s CommandStatus) Exit() (bool, int) {
	return s.terminate, s.code
}

// func (s CommandStatus) initBuffer() {
// 	s.Stdout = make([]byte, 0, defaultCommandStatusBufferSize)
// }

func findCmdPath(name string) (cmdPath string, found bool) {
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
	found = err == nil

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

// type CommandIndex struct {
// 	index map[string]Command
// }

// func NewCommandIndex() (index *CommandIndex) {
// 	index = &CommandIndex{
// 		index: make(
// 			map[string]Command,
// 			min(len(commands), defaultCommandIndexCapacity),
// 		),
// 	}

// 	for _, cmd := range commands {
// 		index.index[cmd.Name()] = cmd
// 	}

// 	return
// }

// var (
// 	commandIndex     *CommandIndex
// 	commandIndexOnce sync.Once
// )

// func GetCommandIndex() *CommandIndex {
// 	commandIndexOnce.Do(func() {
// 		commandIndex = NewCommandIndex()
// 	})

// 	return commandIndex
// }

func ExecCommand(argsStr string) (status CommandStatus) {
	parsedArgs, stdout, stderr := parseCommandArgs(argsStr)
	// var stdout io.Writer = os.Stdout
	// var stderr io.Writer = os.Stderr
	var stdin io.Reader

	lastStatus := make(chan CommandStatus, 1)
	// copyDone := make(chan struct{}, 1)
	// defer close(lastStatus)
	// defer close(copyDone)

	for i, parsed := range parsedArgs.cmds {
		// log.Println(parsed)
		cmd, found := GetCommandsIndex().Get(parsed.name)

		if !found {
			fmt.Fprintf(stderr, "%s: command not found\n", parsed.name)

			return newErrorStatus()
		}

		cmd.SetIO(stdin, stderr)
		// log.Println(cmd)
		stdin = cmd.GetStdout()

		if i == len(parsedArgs.cmds)-1 {
			// log.Println(cmd.Name())
			if cmd.name == "echo" {
				log.Println(stdout, os.Stdout)
			}

			go cmd.ExecGo(parsed.args, lastStatus)
		} else {
			go cmd.Exec(parsed.args)
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		io.Copy(stdout, stdin)
		wg.Done()
	}()

	go func() {
		status = <-lastStatus

		wg.Done()
	}()

	wg.Wait()

	// for range 2 {
	// 	select {
	// 	case <-copyDone:
	// 	case status = <-lastStatus:
	// 	}
	// }

	// status = <-lastStatus

	return status
}
