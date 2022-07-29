package kittla

import (
	"fmt"
	"math"
	"strconv"
)

type valueType int

const (
	valTypeInt valueType = iota
	valTypeFloat
	valTypeBool
	valTypeStr
)

type obj struct {
	valType  valueType
	valInt   int
	valFloat float64
	valBool  bool
	valStr   []byte
}

func (o *obj) toBytes() []byte {
	if o == nil {
		return nil
	}
	switch o.valType {
	case valTypeInt:
		return []byte(fmt.Sprintf("%d", o.valInt))
	case valTypeFloat:
		return []byte(fmt.Sprintf("%f", o.valFloat))
	case valTypeBool:
		return []byte(fmt.Sprintf("%t", o.valBool))
	case valTypeStr:
		return o.valStr
	}
	return nil
}

func (o *obj) isTrue() bool {
	return (o.valType == valTypeBool && o.valBool) || (o.valType == valTypeInt && o.valInt != 0)
}

func toObj(arg []byte) *obj {
	if v, err := strconv.ParseInt(string(arg), 0, 64); err == nil {
		return &obj{valType: valTypeInt, valInt: int(v)}
	}
	if v, err := strconv.ParseFloat(string(arg), 64); err == nil {
		return &obj{valType: valTypeFloat, valFloat: v}
	}
	if v, err := strconv.ParseBool(string(arg)); err == nil {
		return &obj{valType: valTypeBool, valBool: v}
	}
	return &obj{valType: valTypeStr, valStr: arg}
}

func (o *obj) optimize() *obj {
	if o.valType == valTypeStr {
		return toObj(o.valStr)
	}
	return o
}

type frame struct {
	prevCmd cmdId
	ifTaken bool // Changed if prevCmd == CMD_IF || CMD_ELIF
	objects map[string]*obj
}

// Kittla instance
type Kittla struct {
	commands  map[string][]*command
	currLine  int
	frames    []*frame
	currFrame *frame

	isContinue bool // Set until continue is handled
	isBreak    bool // Set until break is handled
}

// New returns a new instance of the kittla language
func New() *Kittla {
	k := &Kittla{commands: getCmdMap()}
	k.currFrame = &frame{objects: make(map[string]*obj)}
	return k
}

// Execute one parsed command. First entry in args is the command. Might be recursive in case of
// more complex commands like if {} {body}.
func (k *Kittla) executeCmd(args []*obj) (*obj, error) {
	cmdName := string(args[0].toBytes())

	if cmd, present := k.commands[cmdName]; present {
		minArgs := math.MaxInt
		maxArgs := 0

		for i := range cmd {
			if cmd[i].minArgs < minArgs {
				minArgs = cmd[i].minArgs
			}
			if cmd[i].maxArgs > maxArgs || cmd[i].maxArgs == -1 {
				if maxArgs != -1 {
					maxArgs = cmd[i].maxArgs
				}
			}

			if cmd[i].minArgs == -1 || len(args[1:]) >= cmd[i].minArgs {
				if cmd[i].maxArgs == -1 || len(args[1:]) <= cmd[i].maxArgs {
					defer func() { k.currFrame.prevCmd = cmd[i].id }()
					return cmd[i].fn(k, cmd[i].id, cmdName, args[1:])
				}
			}
		}
		if minArgs != -1 && len(args[1:]) < minArgs {
			return nil, fmt.Errorf("%s must have atleast %d arguments. Got %d. Line: %d", cmdName, minArgs, len(args[1:]), k.currLine)
		}

		if maxArgs != -1 && len(args[1:]) > maxArgs {
			return nil, fmt.Errorf("%s must have at most %d arguments. Got %d. Line: %d", cmdName, maxArgs, len(args[1:]), k.currLine)
		}

		return nil, fmt.Errorf("%s wrong number of arguments. Line: %d", cmdName, k.currLine)
	}
	return k.commands["unknown"][0].fn(k, CMD_UNKNOWN, cmdName, args[1:])
}

// Expands any $name to the actual value.
func (k *Kittla) expandVar(cb *codeBlock) (*obj, error) {

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

func (k *Kittla) parse(cb *codeBlock, isPre bool) ([]*obj, error) {

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

	args := make([]*obj, 0, 256)
	currArg := make([]byte, 0, 256)
	var currObj *obj

	appendResult := func(result *obj) {
		if len(currArg) != 0 {
			currArg = append(currArg, result.toBytes()...)
		} else if currObj != nil {
			currArg = append(currObj.toBytes(), result.toBytes()...)
			currObj = nil
		} else {
			currObj = result
		}
	}

	appendArg := func() {
		if len(currArg) > 0 {
			args = append(args, toObj(currArg))
		} else if currObj != nil {
			args = append(args, currObj)
		}
		currArg = make([]byte, 0, 256)
		currObj = nil
	}

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
			if largs, err := k.parse(cb, true); err == nil {
				if result, err := k.executeCmd(largs); err == nil {
					appendResult(result)
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		case '$':
			if result, err := k.expandVar(cb); err == nil {
				appendResult(result)
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
				appendArg()
			} else {
				currArg = append(currArg, c)
			}
		default:
			currArg = append(currArg, c)
		}
	}
	appendArg()
	return args, nil
}

// main execution command. Returns the last commands output, its command id and possible error
func (k *Kittla) executeCore(cb *codeBlock) (*obj, cmdId, error) {

	var res *obj
	var args []*obj
	var err error

	k.frames = append(k.frames, k.currFrame)
	k.currFrame = &frame{objects: k.currFrame.objects}

	k.currLine = cb.lineNum

	for !cb.eof && err == nil {
		args, err = k.parse(cb, false)

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
	prevCmd := k.currFrame.prevCmd

	k.currFrame = k.frames[len(k.frames)-1]
	k.frames = k.frames[:len(k.frames)-1]

	return res, prevCmd, err
}

// Executes a program. Returns the last commands output, the command id and possible error.
// A wrapper function to handle break & continue errors and codeBlock creation
func (k *Kittla) Execute(prog string) ([]byte, cmdId, error) {
	res, cmdId, err := k.executeCore(&codeBlock{code: prog, lineNum: 1})
	if err == nil {
		if k.isBreak {
			return nil, cmdId, fmt.Errorf("Unhandled break")
		}
		if k.isContinue {
			return nil, cmdId, fmt.Errorf("Unhandled continue")
		}
	}
	return res.toBytes(), cmdId, err
}
