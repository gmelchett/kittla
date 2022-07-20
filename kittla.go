package kittla

import (
	"fmt"
	"log"
	"runtime/debug"
)

type CodeBlock struct {
	Code    string
	idx     int
	LineNum int
	eof     bool
}

func isBlank(c byte) bool {
	return c == ' ' || c == '\t'
}

func validStartChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_'
}

func validChar(c byte) bool {
	return validStartChar(c) || (c >= '0' && c <= '9')
}

func (cb *CodeBlock) nextPeek(peek bool) byte {
	var c byte
	if cb.eof {
		debug.PrintStack()
		log.Fatal("nextPeek past end!")
		return 'X' // next/peek shouldn't be used at eof
	}

	c = cb.Code[cb.idx]

	if peek {
		if c == '\\' && cb.idx+1 < len(cb.Code) && cb.Code[cb.idx+1] == '\n' {
			return ' '
		}
		return c
	}

	cb.idx++
	cb.eof = cb.idx == len(cb.Code)

	// handle lines which ends with \ and translate to space
	if !cb.eof && c == '\\' && cb.Code[cb.idx] == '\n' {
		cb.LineNum++
		cb.idx++
		c = ' '
		cb.eof = cb.idx == len(cb.Code)
	} else if c == '\n' {
		cb.LineNum++
	}

	return c
}

func (cb *CodeBlock) next() byte {
	return cb.nextPeek(false)
}

func (cb *CodeBlock) peek() byte {
	return cb.nextPeek(true)
}

func (cb *CodeBlock) skipBlanks() {
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

func (cb *CodeBlock) untilBrackedEnd() ([]byte, error) {
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
			return nil, fmt.Errorf("Premature end of file. Line: %d", cb.LineNum)
		}
	}
}

type frame struct {
	prevFunc funcId
	ifTaken  bool // Changed if prevFunc == FUNC_IF || FUNC_ELIF
	objects  map[string][]byte
}

type Kittla struct {
	PrevFunc  funcId
	functions map[string]*function
	currLine  int
	frames    []*frame
	currFrame *frame
}

func New() *Kittla {
	k := &Kittla{functions: getFuncMap()}
	k.currFrame = &frame{objects: make(map[string][]byte)}
	return k
}

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

func (k *Kittla) expandVar(cb *CodeBlock) ([]byte, error) {

	var varName []byte
	var err error

	if cb.eof {
		return nil, fmt.Errorf("Unexpected end of file. Line: %d", cb.LineNum)
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
			return nil, fmt.Errorf("Invalid variable start character. Line: %d", cb.LineNum)
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
	return nil, fmt.Errorf("Unknown variable: %s Line: %d", string(varName), cb.LineNum)
}

func (k *Kittla) Parse(cb *CodeBlock, isPre bool) ([][]byte, error) {

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
			return nil, fmt.Errorf("Stray ]. Line: %d", cb.LineNum)
		case '[':
			k.currLine = cb.LineNum
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

func (k *Kittla) Execute(cb *CodeBlock) ([]byte, error) {

	var res []byte
	var args [][]byte
	var err error

	k.frames = append(k.frames, k.currFrame)
	k.currFrame = &frame{objects: k.currFrame.objects}

	k.currLine = cb.LineNum

	for !cb.eof && err == nil {
		args, err = k.Parse(cb, false)

		if err != nil {
			break
		}
		res, err = k.executeCmd(args)
	}
	k.PrevFunc = k.currFrame.prevFunc

	k.currFrame = k.frames[len(k.frames)-1]
	k.frames = k.frames[:len(k.frames)-1]

	return res, err
}
