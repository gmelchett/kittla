package kittla

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

type valueType int

const (
	valTypeInt valueType = iota
	valTypeFloat
	valTypeBool
	valTypeStr
	valTypeFn
	valTypeList
)

type obj struct {
	valType valueType
	isConst bool

	valInt   int
	valFloat float64
	valBool  bool
	valStr   []byte
	valFn    *command
	valList  []*obj
}

func (o *obj) clone() *obj {
	oc := &obj{
		isConst:  o.isConst,
		valType:  o.valType,
		valInt:   o.valInt,
		valFloat: o.valFloat,
		valBool:  o.valBool,
		valStr:   make([]byte, cap(o.valStr)),
		valList:  make([]*obj, 0, len(o.valList)),
	}
	for i := range o.valList {
		oc.valList = append(oc.valList, o.valList[i].clone())
	}
	copy(oc.valStr, o.valStr)
	return oc
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
	case valTypeFn:
		return o.valFn.body.toBytes()
	case valTypeList:
		b := make([]byte, 0, 1024)
		b = append(b, '(')
		for i := range o.valList {
			b = append(b, o.valList[i].toBytes()...)
			if i+1 < len(o.valList) {
				b = append(b, []byte(", ")...)
			}
		}
		return append(b, ')')
	}
	return nil
}
func (o *obj) toString() string {
	return string(o.toBytes())
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
	if string(arg) == "true" {
		return &obj{valType: valTypeBool, valBool: true}
	}
	if string(arg) == "false" {
		return &obj{valType: valTypeBool, valBool: false}
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
	prevCmd CmdID
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
	isReturn   bool // set until return is handled

	nextFnId CmdID
}

// New returns a new instance of the kittla language
func New() *Kittla {
	k := &Kittla{commands: getCmdMap(), nextFnId: CMD_END_OF_BUILT_IN + 1}
	k.currFrame = &frame{objects: make(map[string]*obj)}
	return k
}

func (k *Kittla) AddFunction(cmdName string, minArgs, maxArgs int, goFn func(*Kittla, []string) (string, error)) error {
	if maxArgs < minArgs && maxArgs >= 0 {
		return fmt.Errorf("Max number of arguments can't be less than minimum number of arguments")
	}
	k.commands[cmdName] = append(k.commands[cmdName], &command{names: []string{cmdName}, minArgs: minArgs, maxArgs: maxArgs, id: k.nextFnId, goFn: goFn})
	k.nextFnId++
	return nil
}

func (k *Kittla) SetVar(varName, value string) {
	k.currFrame.objects[varName] = toObj([]byte(value))
}

func (k *Kittla) GetVar(varName string) (string, bool) {
	if v, present := k.currFrame.objects[varName]; present {
		return v.toString(), true
	}
	return "", false
}

// Execute one parsed command. First entry in args is the command. Might be recursive in case of
// more complex commands like if {} {body}.
func (k *Kittla) executeCmd(args []*obj) (*obj, error) {
	cmdName := args[0].toString()

	var cmd []*command
	var present bool
	var ano bool

	if o, exists := k.currFrame.objects[cmdName]; exists && k.currFrame.objects[cmdName].valType == valTypeFn {
		cmd = []*command{o.valFn}
		present = true
		ano = true
	} else {
		cmd, present = k.commands[cmdName]
	}

	if !present {
		return k.commands["unknown"][0].fn(k, CMD_UNKNOWN, cmdName, args[1:])
	}

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
				if cmd[i].goFn != nil { // Added go functions
					argsStrs := make([]string, 0, len(args[1:]))
					for i := range args[1:] {
						argsStrs = append(argsStrs, args[i+1].toString())
					}

					v, err := cmd[i].goFn(k, argsStrs)
					if err == nil {
						return toObj([]byte(v)), err
					}
					return nil, err
				} else if !ano { // Functions without a name
					return cmd[i].fn(k, cmd[i].id, cmdName, args[1:])
				} else { // normal functions
					return call(k, cmd[0], cmdName, args[1:])
				}
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

	if len(cb.code) == 0 {
		return nil, nil
	}

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
	appendEmpty := false

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
		} else if appendEmpty {
			args = append(args, toObj(currArg))
		}
		appendEmpty = false
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
				// {} is a valid object
				appendEmpty = true
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
func (k *Kittla) executeCore(cb *codeBlock, pushFrame bool) (*obj, CmdID, error) {

	var res *obj
	var args []*obj
	var err error

	if pushFrame {
		k.frames = append(k.frames, k.currFrame)
		k.currFrame = &frame{objects: k.currFrame.objects}
	}

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
			if k.isReturn {
				k.isReturn = false
				break
			}
		}
	}
	prevCmd := k.currFrame.prevCmd

	if pushFrame {
		k.currFrame = k.frames[len(k.frames)-1]
		k.frames = k.frames[:len(k.frames)-1]
	}

	return res, prevCmd, err
}

// Executes a program. Returns the last commands output, the command id and possible error.
// A wrapper function to handle break & continue errors and codeBlock creation
func (k *Kittla) Execute(prog string) ([]byte, CmdID, error) {
	res, cmdID, err := k.executeCore(&codeBlock{code: prog, lineNum: 1}, true)
	if err == nil {
		if k.isBreak {
			return nil, cmdID, fmt.Errorf("Unhandled break")
		}
		if k.isContinue {
			return nil, cmdID, fmt.Errorf("Unhandled continue")
		}
		if k.isReturn {
			os.Exit(0)
		}

	}
	return res.toBytes(), cmdID, err
}
