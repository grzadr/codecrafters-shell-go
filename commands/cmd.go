package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
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

type CmdPrefixMatched []string

func (m *CmdPrefixMatched) FindClosest(
	prefix string,
) (name string, found bool) {
	switch len(*m) {
	case 0:
		return
	case 1:
		name, found = (*m)[0], true
	default:
		name, found = m.findNext(prefix)
	}

	return
}

func (m *CmdPrefixMatched) findNext(prefix string) (next string, found bool) {
	// ref := (*m)[0]
	var ref string

	refNum := -1

	for i, other := range *m {
		if strings.HasPrefix(other, prefix) {
			ref = other
			refNum = i

			break
		}
	}

	if refNum == -1 {
		return next, found
	} else if refNum == len(*m)-1 {
		return ref, true
	}

	total := 0
	isPrefixed := 0

	for _, other := range (*m)[refNum+1:] {
		total++

		if strings.HasPrefix(other, ref) {
			isPrefixed++
		}
	}

	return (*m)[refNum+1], total == isPrefixed
}

func FindCmdPrefixPath(prefix string) (matched CmdPrefixMatched) {
	matched = make(CmdPrefixMatched, 0)

	for dir := range strings.SplitSeq(os.Getenv("PATH"), ":") {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil || info.Mode()&0o111 == 0 ||
				!strings.HasPrefix(info.Name(), prefix) {
				continue
			}

			matched = append(matched, info.Name())
		}
	}

	slices.Sort(matched)

	return
}

func findCmdPath(name string) (cmdPath string, found bool) {
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

func (h *CommandHistory) Prev() (cmd string) {
	cmd = h.cmds[h.current]
	h.current = max(h.current-1, 0)

	return
}

func (h *CommandHistory) Next() (cmd string) {
	h.current = min(h.current+2, h.size()-1)
	cmd = h.cmds[h.current]

	return
}

func (h *CommandHistory) size() int {
	return len(h.cmds)
}

func (h *CommandHistory) append(cmd string) {
	h.cmds = append(h.cmds, strings.TrimSpace(cmd))
	h.current = h.size() - 1
}

var (
	history     *CommandHistory
	historyOnce sync.Once
)

func GetCommandHistory() *CommandHistory {
	historyOnce.Do(func() {
		history = &CommandHistory{
			cmds: make([]string, 0, defaultHistorySize),
		}
	})

	return history
}

func ExecCommand(argsStr string) (status CommandStatus) {
	GetCommandHistory().append(argsStr)
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
