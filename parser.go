package kittla

import (
	"fmt"
	"log"
	"runtime/debug"
)

type codeBlock struct {
	code    string
	idx     int
	lineNum int
	eof     bool
}

func isBlank(c byte) bool {
	return c == ' ' || c == '\t'
}

// valid command start character
func validStartChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_'
}

// valid characters in command after first character
func validChar(c byte) bool {
	return validStartChar(c) || (c >= '0' && c <= '9')
}

// Get next character from input. Moves forward in buffer if peek = false.
// Keeps track of current line number, and \ at end of line
func (cb *codeBlock) nextPeek(peek bool) byte {
	var c byte
	if cb.eof {
		debug.PrintStack()
		log.Fatal("nextPeek past end!")
		return 'X' // next/peek shouldn't be used at eof
	}

	c = cb.code[cb.idx]

	if peek {
		if c == '\\' && cb.idx+1 < len(cb.code) && cb.code[cb.idx+1] == '\n' {
			return ' '
		}
		return c
	}

	cb.idx++
	cb.eof = cb.idx == len(cb.code)

	// handle lines which ends with \ and translate to space
	if !cb.eof && c == '\\' && cb.code[cb.idx] == '\n' {
		cb.lineNum++
		cb.idx++
		c = ' '
		cb.eof = cb.idx == len(cb.code)
	} else if c == '\n' {
		cb.lineNum++
	}

	return c
}

func (cb *codeBlock) next() byte {
	return cb.nextPeek(false)
}

func (cb *codeBlock) peek() byte {
	return cb.nextPeek(true)
}

// scans forward until none blank or end-of-file
func (cb *codeBlock) skipBlanks() {
	for {
		if cb.eof {
			return
		}

		c := cb.peek()
		if !isBlank(c) {
			return
		}

		cb.next()
	}
}

// Continues scanning forward until paired } shows up - or end of file.
func (cb *codeBlock) untilBrackedEnd() ([]byte, error) {
	res := make([]byte, 0, 256)
	depth := 1
	for {
		c := cb.next()
		if c == '\\' {
			res = append(res, c)
			res = append(res, cb.next())
			continue
		}

		if c == '}' {
			depth--
			if depth == 0 {
				return res, nil
			}
		} else if c == '{' {
			depth++
		}
		res = append(res, c)
		if cb.eof {
			return nil, fmt.Errorf("Premature end of file. Line: %d", cb.lineNum)
		}
	}
}
