package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Command interface {
	Name() string
	Exec(args []string) (value CommandStatus)
}

type CmdBase struct {
	outWriter io.Writer
	errWriter io.Writer
	inReader  io.Reader
}

func NewCmdBase(stdin io.Reader) (cmd *CmdBase, stdout, stderr io.Reader) {
	cmd = &CmdBase{
		inReader: stdin,
	}
	stdout, cmd.outWriter = io.Pipe()
	stderr, cmd.errWriter = io.Pipe()

	return
}

type CmdGeneric struct {
	name string
	path string
	CmdBase
}

func newCmdGeneric(
	cmdName, cmdPath string, stdin io.Reader,
) (cmd *CmdGeneric, stdout, stderr io.Reader) {
	base, stdout, stderr := NewCmdBase(stdin)
	cmd = &CmdGeneric{
		name:    cmdName,
		path:    cmdPath,
		CmdBase: *base,
	}

	// stdout, cmd.outWriter = io.Pipe()
	// stderr, cmd.errWriter = io.Pipe()
	// cmd.Stderr = bufio.NewWriter(
	// 	bytes.NewBuffer(make([]byte, 0, defaultCommandStatusBufferSize)),
	// )

	return
}

func (c CmdGeneric) Name() string {
	return c.name
}

func (c CmdGeneric) Exec(args []string) (value CommandStatus) {
	cmd := exec.Command(c.name, args...)
	cmd.Path = c.path
	cmd.Stdin = c.inReader
	cmd.Stdout = c.outWriter
	cmd.Stderr = c.errWriter
	cmd.Start()
	value.err = cmd.Wait()

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

func (c CmdExit) Exec(args []string) (value CommandStatus) {
	value.terminate = true
	if value.code, value.err = strconv.Atoi(args[0]); value.err != nil {
		return cmdExitErr
	}

	return
}

type CmdEcho struct{}

func (c CmdEcho) Name() string {
	return "echo"
}

func (c CmdEcho) Exec(args []string) (value CommandStatus) {
	value.Stdout = []byte(strings.Join(args, " ") + "\n")

	return
}

type CmdType struct{}

func (c CmdType) Name() string {
	return "type"
}

func (c CmdType) Exec(args []string) (value CommandStatus) {
	value.initBuffer()

	cmdStr := args[0]

	if GetCommandIndex().Find(cmdStr) {
		value.Stdout = fmt.Appendf(
			value.Stdout,
			"%s is a shell builtin\n",
			cmdStr,
		)
	} else if cmd, found := findCmdInPath(cmdStr); found {
		value.Stdout = fmt.Appendf(value.Stdout, "%s is %s\n", cmdStr, cmd.path)
	} else {
		value = newNotFoundError(cmdStr)
	}

	return
}

type CmdPwd struct{}

func (c CmdPwd) Name() string {
	return "pwd"
}

func (c CmdPwd) Exec(args []string) (value CommandStatus) {
	if pwd, err := os.Getwd(); err != nil {
		value = newGenericStatusError(err)
	} else {
		value.Stdout = []byte(pwd + "\n")
	}

	return
}

type CmdCd struct{}

func (c CmdCd) Name() string {
	return "cd"
}

func (c CmdCd) Exec(args []string) (value CommandStatus) {
	dir := args[0]
	if dir == "~" {
		dir, _ = os.UserHomeDir()
	}

	if os.Chdir(dir) != nil {
		value = newGenericStatusError(c.noDirError(dir))
	}

	return
}

func (c CmdCd) noDirError(dirname string) error {
	return fmt.Errorf("%s: %s: No such file or directory", c.Name(), dirname)
}

var commands = [...]Command{
	CmdCd{},
	CmdEcho{},
	CmdExit{},
	CmdPwd{},
	CmdType{},
}
