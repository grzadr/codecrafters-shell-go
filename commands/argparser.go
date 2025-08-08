package commands

import "iter"

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

func parseCommandArgs(argsStr string) (name string, args []string) {
	args = make([]string, 0, defaultArgsBuffer)

	for arg := range newArgParser(argsStr).parseArgs() {
		args = append(args, arg)
	}

	name = args[0]
	args = args[1:]

	return
}
