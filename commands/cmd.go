package commands

import (
	"fmt"
	"iter"
	"log"
	"os/exec"
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

type runeBuffer []rune

func (b runeBuffer) first() rune {
	// if len(b) > 0 {
	// 	r = b[0]
	// }
	return b[0]
}

type argRuneIterator struct {
	source     runeBuffer
	offset     int
	terminator rune
}

func newArgParser(args runeBuffer) *argRuneIterator {
	return &argRuneIterator{
		source:     args,
		terminator: ' ',
	}
}

func (parser *argRuneIterator) size() int {
	return len(parser.source)
}

func (parser *argRuneIterator) left() int {
	return parser.size() - parser.offset
}

func (parser *argRuneIterator) done() bool {
	return parser.left() == 0
}

func (parser *argRuneIterator) rest() runeBuffer {
	return parser.source[parser.offset:]
}

func (parser *argRuneIterator) peekRune() rune {
	return parser.source[parser.offset]
}

func (parser *argRuneIterator) peekNext() (next rune, ok bool) {
	if parser.left() > 0 {
		return
	}

	next = parser.source[parser.offset+1]

	return
}

func (parser *argRuneIterator) skip() {
	parser.offset++
}

func (parser *argRuneIterator) readRune() (r rune, ok bool) {
	if parser.done() {
		return
	}

	r = parser.peekRune()
	parser.skip()

	ok = true

	return
}

func (parser *argRuneIterator) hasSpaceterminator() bool {
	return parser.terminator == ' '
}

func (parser *argRuneIterator) setTerminator() {
	switch r := parser.peekRune(); r {
	case rune('"'), rune('\''):
		parser.terminator = r
		parser.skip()
	}
}

func (parser *argRuneIterator) firstOther(other rune) {
	for {
		if r, ok := parser.readRune(); !ok || r != other {
			return
		}
	}
}

func (parser *argRuneIterator) terminate() (r rune, done bool) {
	r, ok := parser.readRune()
	if !ok || r != parser.terminator {
		log.Println("terminate()", string(r), done)

		return
	}

	done = true

	if parser.hasSpaceterminator() {
		parser.firstOther(' ')
	} else if next, hasNext := parser.peekNext(); hasNext && next == parser.terminator {
		parser.skip()
	}

	return
}

func (parser *argRuneIterator) nextArg() (string, bool) {
	buffer := make(runeBuffer, len(parser.source))
	buffOffset := 0

	parser.setTerminator()

	for {
		r, done := parser.terminate()

		if done {
			log.Println(
				"terminating",
				buffOffset,
				parser.offset,
				string(parser.rest()),
			)

			break
		}

		buffer[buffOffset] = r
		buffOffset += 1
	}

	if buffOffset > 0 {
		return string(buffer[:buffOffset]), true
	}

	return "", false
}

func (parser *argRuneIterator) parseArgs() iter.Seq[string] {
	return func(yield func(string) bool) {
		for {
			arg, ok := parser.nextArg()

			if !ok || !yield(arg) {
				log.Println("done")

				return
			}

			log.Println(arg)
		}
	}
}

func parseCommandArgs(cmdStr string) (name string, args []string) {
	// name, args, _ = strings.Cut(strings.TrimSpace(cmdStr), " ")
	// reader := bufio.NewReader()
	// reader.ReadString()
	// buffer := make([]rune, len(cmdStr))
	// offset := 0
	// lastEnd := 0
	// spaceCount := 0
	// initial := rune(0)
	args = make([]string, 0, defaultArgsBuffer)

	for arg := range newArgParser(runeBuffer(cmdStr)).parseArgs() {
		args = append(args, arg)
	}

	name = args[0]
	args = args[1:]

	return
}

func ExecCommand(args string) CommandStatus {
	cmdName, cmdArgs := parseCommandArgs(args)
	cmd, found := GetCommandIndex().Get(cmdName)

	if found {
		return cmd.Exec(cmdArgs)
	} else if cmd, found = findCmdInPath(cmdName); found {
		return cmd.Exec(cmdArgs)
	}

	return newUnknownCommandError(cmdName)
}
