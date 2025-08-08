package commands

import (
	"fmt"
	"iter"
	"os/exec"
	"strings"
	"sync"
)

const (
	defaultCommandStatusBufferSize = 1024
	defaultArgsBuffer              = 8
)

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

type runeBuffer struct {
	buff   []rune
	offset int
}

func newRuneBuffer(size int) runeBuffer {
	return runeBuffer{
		buff: make([]rune, size),
	}
}

func (b *runeBuffer) append(r rune) {
	b.buff[b.offset] = r
	b.offset++
}

func (b *runeBuffer) empty() bool {
	return b.offset == 0
}

func (b *runeBuffer) content() string {
	return string(b.buff[:b.offset])
}

func (b *runeBuffer) clear() {
	clear(b.buff)
	b.offset = 0
}

type argIterator struct {
	source     []rune
	offset     int
	buffer     runeBuffer
	terminator rune
}

func newArgParser(args string) *argIterator {
	return &argIterator{
		source:     []rune(args),
		buffer:     newRuneBuffer(len(args)),
		terminator: ' ',
	}
}

func (parser *argIterator) size() int {
	return len(parser.source)
}

func (parser *argIterator) left() int {
	return parser.size() - parser.offset
}

func (parser *argIterator) done() bool {
	return parser.left() == 0
}

func (parser *argIterator) rest() string {
	return string(parser.source[parser.offset:])
}

func (parser *argIterator) peek() rune {
	return parser.source[parser.offset]
}

func (parser *argIterator) prev() rune {
	return parser.source[parser.offset-1]
}

func (parser *argIterator) skip() bool {
	if parser.done() {
		return false
	}

	parser.offset++

	return true
}

func (parser *argIterator) readRune() (r rune, ok bool) {
	if parser.done() {
		return
	}

	r = parser.peek()
	parser.skip()

	ok = true

	return
}

func (parser *argIterator) isSpaceTerminated() bool {
	return parser.terminator == ' '
}

func (parser *argIterator) skipRune(item rune) {
	for {
		if !parser.done() && parser.peek() == item {
			parser.skip()

			continue
		}

		break
	}
}

func (parser *argIterator) skipTwinQuotes() bool {
	if parser.isSpaceTerminated() || parser.done() {
		return false
	}

	if parser.prev() == parser.terminator &&
		parser.peek() == parser.terminator {
		parser.skip()

		return true
	}

	return false
}

func (parser *argIterator) setup() {
	parser.skipRune(' ')
	parser.buffer.clear()

	switch r := parser.peek(); r {
	case rune('"'), rune('\''):
		parser.terminator = r
		parser.skip()
	}
}

func (parser *argIterator) nextArg() (arg string, ok bool) {
	if parser.done() {
		return arg, ok
	}

	parser.setup()

	for {
		r, ok := parser.readRune()
		if !ok {
			break
		}

		if r == '\\' && parser.isSpaceTerminated() {
			r, _ = parser.readRune()
		} else if r == parser.terminator {
			if parser.skipTwinQuotes() {
				continue
			}

			break
		}

		parser.buffer.append(r)
	}

	return parser.buffer.content(), !parser.buffer.empty()
}

func (parser *argIterator) parseArgs() iter.Seq[string] {
	return func(yield func(string) bool) {
		for {
			arg, ok := parser.nextArg()

			if !ok || !yield(arg) {
				return
			}
		}
	}
}

func parseCommandArgs(argsStr string) (name string, args []string) {
	args = make([]string, 0, defaultArgsBuffer)

	for arg := range newArgParser(argsStr).parseArgs() {
		args = append(args, arg)
	}

	name = args[0]
	args = args[1:]

	return
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
