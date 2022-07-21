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

type frame struct {
	prevFunc funcId
	ifTaken  bool // Changed if prevFunc == FUNC_IF || FUNC_ELIF
	objects  map[string][]byte
}

type Kittla struct {
	functions map[string]*function
	currLine  int
	frames    []*frame
	currFrame *frame

	isContinue bool // Set until continue is handled
	isBreak    bool // Set until break is handled
}

// Returns a new instance of the kittla language
func New() *Kittla {
	k := &Kittla{functions: getFuncMap()}
	k.currFrame = &frame{objects: make(map[string][]byte)}
	return k
}

// Execute one parsed command. First entry in args is the command. Might be recursive in case of
// more complex commands like if {} {body}.
func (k *Kittla) executeCmd(args [][]byte) ([]byte, error) {
	fName := string(args[0])

	if fn, present := k.functions[fName]; present {
		if fn.minArgs != -1 && len(args[1:]) < fn.minArgs {
			return nil, fmt.Errorf("%s must have at least %d argument(s). Line: %d", fName, fn.minArgs, k.currLine)
		}
		if fn.maxArgs != -1 && len(args[1:]) > fn.maxArgs {
			return nil, fmt.Errorf("%s must have at most %d argument(s). Line: %d", fName, fn.maxArgs, k.currLine)
		}
		defer func() { k.currFrame.prevFunc = fn.funcId }()
		return fn.fn(k, fn.funcId, fName, args[1:])

	}
	return k.functions["unknown"].fn(k, FUNC_UNKNOWN, fName, args[1:])
}

// Expands any $name to the actual value.
func (k *Kittla) expandVar(cb *codeBlock) ([]byte, error) {

	var varName []byte
	var err error

	if cb.eof {
		return nil, fmt.Errorf("Unexpected end of file. Line: %d", cb.lineNum)
	}

	c := cb.peek()
	if c == '{' {
		varName, err = cb.untilBrackedEnd()
		if err != nil {
			return nil, err
		}
	} else {

		c = cb.next()

		if !validStartChar(c) {
			return nil, fmt.Errorf("Invalid variable start character. Line: %d", cb.lineNum)
		}
		varName = append(varName, c)
		for {
			if cb.eof {
				break
			}

			c = cb.peek()

			if validChar(c) {
				varName = append(varName, c)
				cb.next()
				if !cb.eof {
					continue
				}
			}
			break
		}

	}
	if v, present := k.currFrame.objects[string(varName)]; present {
		return v, nil
	}
	return nil, fmt.Errorf("Unknown variable: %s Line: %d", string(varName), cb.lineNum)
}

func (k *Kittla) Parse(cb *codeBlock, isPre bool) ([][]byte, error) {

	for {
		cb.skipBlanks()
		if cb.eof {
			return nil, nil
		}

		c := cb.peek()
		if c == '#' {
			for {
				c = cb.next()
				if cb.eof || c == '\n' {
					break
				}
			}
			continue
		}
		break
	}

	args := make([][]byte, 0, 256)
	currArg := make([]byte, 0, 256)

	insideString := false
parseLoop:
	for {
		if cb.eof {
			break
		}

		c := cb.next()

		switch c {
		case '\\':
			c = cb.next()
			switch c {
			case 'a':
				c = '\a'
			case 'b':
				c = '\b'
			case 'f':
				c = '\f'
			case 'n':
				c = '\n'
			case 'r':
				c = '\r'
			case 't':
				c = '\t'
			case 'v':
				c = '\v'
			default:
				currArg = append(currArg, '\\')
			}
			currArg = append(currArg, c)
		case '"':
			insideString = !insideString

		case ';', '\n':
			if !insideString {
				break parseLoop
			}
		case ']':
			if isPre {
				break parseLoop
			}
			return nil, fmt.Errorf("Stray ]. Line: %d", cb.lineNum)
		case '[':
			k.currLine = cb.lineNum
			if largs, err := k.Parse(cb, true); err == nil {
				if result, err := k.executeCmd(largs); err == nil {
					currArg = append(currArg, result...)
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		case '$':
			if result, err := k.expandVar(cb); err == nil {
				currArg = append(currArg, result...)
			} else {
				return nil, err
			}
		case '{':
			if result, err := cb.untilBrackedEnd(); err == nil {
				currArg = append(currArg, result...)
			} else {
				return nil, err
			}

		case ' ', '\t':
			if !insideString {
				if len(currArg) > 0 {
					args = append(args, currArg)
					currArg = make([]byte, 0, 256)
				}
			} else {
				currArg = append(currArg, c)
			}
		default:
			currArg = append(currArg, c)
		}
	}
	if len(currArg) > 0 {
		args = append(args, currArg)
	}
	return args, nil
}

// main execution function. Returns the last commands output, its function id and possible error
func (k *Kittla) executeCore(cb *codeBlock) ([]byte, funcId, error) {

	var res []byte
	var args [][]byte
	var err error

	k.frames = append(k.frames, k.currFrame)
	k.currFrame = &frame{objects: k.currFrame.objects}

	k.currLine = cb.lineNum

	for !cb.eof && err == nil {
		args, err = k.Parse(cb, false)

		if err != nil {
			break
		}
		if len(args) > 0 {
			res, err = k.executeCmd(args)
			if k.isBreak || k.isContinue {
				break
			}
		}
	}
	prevFunc := k.currFrame.prevFunc

	k.currFrame = k.frames[len(k.frames)-1]
	k.frames = k.frames[:len(k.frames)-1]

	return res, prevFunc, err
}

// Executes a program. Returns the last commands output, the function id and possible error.
// A wrapper function to handle break & continue errors and codeBlock creation
func (k *Kittla) Execute(prog string) ([]byte, funcId, error) {
	res, funcId, err := k.executeCore(&codeBlock{code: prog, lineNum: 1})
	if err == nil {
		if k.isBreak {
			return nil, funcId, fmt.Errorf("Unhandled break")
		}
		if k.isContinue {
			return nil, funcId, fmt.Errorf("Unhandled continue")
		}
	}
	return res, funcId, err
}
