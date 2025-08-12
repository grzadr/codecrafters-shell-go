package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

const (
	defaultArgsBuffer  = 8
	defaultHistorySize = 16
	defaultFileMode    = 0o644
)

type CommandStatus struct {
	code int
	// err       error
	terminate bool
}

func newErrorStatus() CommandStatus {
	return CommandStatus{code: 1}
}

func (s CommandStatus) Exit() (bool, int) {
	return s.terminate, s.code
}

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

type CommandHistory struct {
	cmds    []string
	current int
}

func (h *CommandHistory) append(cmd string) {
	h.cmds = append(h.cmds, cmd)
	h.current = len(h.cmds) - 1
}

var (
	history     *CommandHistory
	historyOnce sync.Once
)

func getCommandHistory() *CommandHistory {
	historyOnce.Do(func() {
		history = &CommandHistory{
			cmds: make([]string, 0, defaultHistorySize),
		}
	})

	return history
}

func ExecCommand(argsStr string) (status CommandStatus) {
	getCommandHistory().append(argsStr)
	parsedArgs, stdout, stderr := parseCommandArgs(argsStr)

	var stdin io.Reader

	lastStatus := make(chan CommandStatus, 1)

	for i, parsed := range parsedArgs.cmds {
		cmd, found := GetCommandsIndex().Get(parsed.name)

		if !found {
			fmt.Fprintf(stderr, "%s: command not found\n", parsed.name)

			return newErrorStatus()
		}

		cmd.SetIO(stdin, stderr)
		stdin = cmd.GetStdout()

		if i == len(parsedArgs.cmds)-1 {
			go cmd.ExecGo(parsed.args, lastStatus)
		} else {
			go cmd.Exec(parsed.args)
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		io.Copy(stdout, stdin)
		wg.Done()
	}()
	wg.Add(1)

	go func() {
		status = <-lastStatus

		wg.Done()
	}()

	wg.Wait()

	return status
}
