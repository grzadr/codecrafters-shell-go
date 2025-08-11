package commands

import (
	"io"
	"iter"
	"os"
	"strings"
)

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

func (parser *argIterator) escapeBackslash() (r rune) {
	switch parser.terminator {
	case '"':
		switch r = parser.peek(); r {
		case '\\', '"':
			parser.skip()
		default:
			r = parser.prev()
		}
	case '\'':
		r = parser.prev()
	case ' ':
		r, _ = parser.readRune()
	}

	return
}

func (parser *argIterator) isConcatenated() bool {
	return !parser.isSpaceTerminated() && !parser.done() && parser.peek() != ' '
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

loop:
	for {
		r, ok := parser.readRune()
		if !ok {
			break
		}

		switch r {
		case '\\':
			r = parser.escapeBackslash()
		// case '\'', '"':

		case parser.terminator:
			if parser.skipTwinQuotes() || parser.isConcatenated() {
				continue
			}
			// } else if parser.isSpaceTerminated() || parser.done() {
			// 	break loop
			// } else {
			// 	continue
			// }
			break loop
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

type parsedCmd struct {
	name string
	args []string
}

type parsedArgs struct {
	cmds []parsedCmd
	// stdout io.Writer
	// stderr io.Writer
}

func newParsedArgs() *parsedArgs {
	return &parsedArgs{
		cmds: make([]parsedCmd, 0, defaultArgsBuffer),
		// stdout: os.Stdout,
		// stderr: os.Stderr,
	}
}

func (a *parsedArgs) append(name string, args []string) {
	if len(name) > 0 {
		a.cmds = append(a.cmds, parsedCmd{name: name, args: args})
	}
}

func (a *parsedArgs) getIO(
	name string,
	args []string,
) (stdout, stderr io.Writer) {
	stdout = os.Stdout
	stderr = os.Stderr

	switch name {
	case ">", "1>":
		// log.Printf("%s %s\n", name, args[0])
		stdout, _ = CreateEmptyFile(args[0])

		if name == "1>" {
			panic("1>")
		}
	case ">>", "1>>":
		stdout, _ = CreateAppendFile(args[0])
	case "2>":
		stderr, _ = CreateEmptyFile(args[0])
	case "2>>":
		stderr, _ = CreateAppendFile(args[0])
	default:
		// log.Printf("appending %q %+v\n", name, args)
		a.append(name, args)
		// log.Println(len(a.cmds))
	}

	if name == "1>" {
		panic("1>")
	}

	return stdout, stderr
}

func parseCommandArgs(
	args string,
) (parsed *parsedArgs, stdout, stderr io.Writer) {
	parsed = newParsedArgs()

	var cmdReady bool

	var cmdName string

	var cmdArgs []string

	for arg := range newArgParser(strings.TrimSpace(args)).parseArgs() {
		if arg == "1>" {
			panic("1>")
		}

		if arg == "|" {
			cmdReady = false
		} else if strings.HasSuffix(arg, ">") || !cmdReady {
			parsed.append(cmdName, cmdArgs)
			cmdName = arg
			cmdReady = true
			cmdArgs = make([]string, 0, defaultArgsBuffer)
		} else {
			cmdArgs = append(cmdArgs, arg)
		}
	}

	stdout, stderr = parsed.getIO(cmdName, cmdArgs)

	return parsed, stdout, stderr
}
