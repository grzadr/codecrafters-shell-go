package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type Command interface {
	Name() string
	Exec(args []string) (value CommandStatus)
	ExecGo(args []string, status chan<- CommandStatus)
	SetIO(stdin io.Reader, stderr io.Writer)
	GetStdout() io.Reader
}

type ExecFunc func(c *CmdBase, args []string) (value CommandStatus)

type CmdBase struct {
	name      string
	path      string
	outWriter io.Writer
	stderr    io.Writer
	stdin     io.Reader
	stdout    io.Reader
	cmd       ExecFunc
}

func NewCmdBase(name string, exec ExecFunc) (cmd *CmdBase) {
	cmd = &CmdBase{name: name, cmd: exec}
	// cmd.SetIO(stdin, stderr)

	return
}

func (c CmdBase) Name() string {
	return c.name
}

func (c *CmdBase) SetIO(stdin io.Reader, stderr io.Writer) {
	c.stdin = stdin
	c.stderr = stderr
	c.stdout, c.outWriter = io.Pipe()
}

func (c *CmdBase) GetStdout() io.Reader {
	return c.stdout
}

func (c *CmdBase) Exec(args []string) (value CommandStatus) {
	value = c.cmd(c, args)

	if closer, ok := c.outWriter.(io.Closer); ok {
		closer.Close()
	}

	return
}

func (c CmdBase) ExecGo(args []string, status chan<- CommandStatus) {
	status <- c.Exec(args)
	// status <- (*c.Exec)(args)
}

// type CmdGeneric struct {
// 	name string
// 	path string
// 	*CmdBase
// }

// func newCmdGeneric(cmdName, cmdPath string) (cmd CmdGeneric) {
// 	cmd = CmdGeneric{
// 		name: cmdName,
// 		path: cmdPath,
// 	}

// 	// stdout, cmd.outWriter = io.Pipe()
// 	// stderr, cmd.errWriter = io.Pipe()
// 	// cmd.Stderr = bufio.NewWriter(
// 	// 	bytes.NewBuffer(make([]byte, 0, defaultCommandStatusBufferSize)),
// 	// )

// 	return
// }

// func (c CmdGeneric) Name() string {
// 	return c.name
// }

func ExecFromPath(c *CmdBase, args []string) (value CommandStatus) {
	cmd := exec.Command(c.name, args...)
	cmd.Path = c.path
	cmd.Stdin = c.stdin
	cmd.Stdout = c.outWriter
	cmd.Stderr = c.stderr

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	var exitErr *exec.ExitError
	if err := cmd.Wait(); errors.As(err, &exitErr) {
		value.code = exitErr.ExitCode()
	} else if err != nil {
		panic(fmt.Errorf("command %s/%s exited with error: %w", c.path, c.name, err))
	}

	return
}

// type CmdExit struct {
// 	*CmdBase
// }

// func (c CmdExit) Name() string {
// 	return "exit"
// }

// var cmdExitErr = newGenericStatusError(
// 	fmt.Errorf("exit requires one integer parameter"),
// )

func ExecExit(c *CmdBase, args []string) (value CommandStatus) {
	value.terminate = true

	var err error

	if value.code, err = strconv.Atoi(args[0]); err != nil {
		fmt.Fprint(c.stderr, "exit requires one integer parameter\n")

		value.code = 1
	}

	return
}

// type CmdEcho struct {
// 	CmdBase
// }

// func (c CmdEcho) Name() string {
// 	return "echo"
// }

func ExecEcho(c *CmdBase, args []string) (value CommandStatus) {
	fmt.Fprintf(c.outWriter, "%s\n", strings.Join(args, " "))

	return
}

// type CmdType struct {
// 	CmdBase
// }

// func (c CmdType) Name() string {
// 	return "type"
// }

func ExecType(c *CmdBase, args []string) (value CommandStatus) {
	cmdStr := args[0]

	if GetCommandsIndex().Find(cmdStr) {
		fmt.Fprintf(c.outWriter, "%s is a shell builtin\n", cmdStr)
	} else if cmdPath, found := findCmdPath(cmdStr); found {
		fmt.Fprintf(c.outWriter, "%s is %s\n", cmdStr, cmdPath)
	} else {
		fmt.Fprintf(c.stderr, "%s: not found\n", cmdStr)

		value = newErrorStatus()
	}

	return
}

// type CmdPwd struct{ CmdBase }

// func (c CmdPwd) Name() string {
// 	return "pwd"
// }

func ExecPwd(c *CmdBase, args []string) CommandStatus {
	if pwd, err := os.Getwd(); err != nil {
		return newErrorStatus()
	} else {
		fmt.Fprintln(c.outWriter, pwd)
	}

	return CommandStatus{}
}

// type CmdCd struct{ CmdBase }

// func (c CmdCd) Name() string {
// 	return "cd"
// }

func ExecCd(c *CmdBase, args []string) (value CommandStatus) {
	dir := args[0]
	if dir == "~" {
		dir, _ = os.UserHomeDir()
	}

	if os.Chdir(dir) != nil {
		fmt.Fprintf(
			c.stderr,
			"%s: %s: No such file or directory\n",
			c.Name(),
			dir,
		)

		value = newErrorStatus()
	}

	return
}

func ExecHistory(c *CmdBase, args []string) CommandStatus {
	// if pwd, err := os.Getwd(); err != nil {
	// 	return newErrorStatus()
	// } else {
	// 	fmt.Fprintln(c.outWriter, pwd)
	// }
	return CommandStatus{}
}

type CommandsIndex map[string]ExecFunc

var (
	commandsIndex     *CommandsIndex
	commandsIndexOnce sync.Once
)

func GetCommandsIndex() *CommandsIndex {
	commandsIndexOnce.Do(func() {
		commandsIndex = &CommandsIndex{
			"cd":      ExecCd,
			"echo":    ExecEcho,
			"exit":    ExecExit,
			"history": ExecHistory,
			"pwd":     ExecPwd,
			"type":    ExecType,
		}
	})

	return commandsIndex
}

func (i CommandsIndex) Get(name string) (*CmdBase, bool) {
	if cmd, found := i[name]; found {
		return NewCmdBase(name, cmd), found
	}

	cmdPath, found := findCmdPath(name)

	if found {
		cmd := NewCmdBase(name, ExecFromPath)
		cmd.path = cmdPath

		return cmd, true
	}

	return &CmdBase{}, false
}

func (i CommandsIndex) Find(name string) (found bool) {
	_, found = i[name]

	return
}
