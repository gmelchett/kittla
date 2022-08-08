package kittla

import (
	"fmt"
	"strconv"

	"github.com/tidwall/expr"
)

type CmdID int

const (
	CMD_BREAK CmdID = iota
	CMD_DEC
	CMD_CONTINUE
	CMD_ELIF
	CMD_ELSE
	CMD_EVAL
	CMD_FLOAT
	CMD_FN
	CMD_IF
	CMD_INC
	CMD_INT
	CMD_LOOP
	CMD_PRINT
	CMD_RETURN
	CMD_UNKNOWN
	CMD_VAR
	CMD_WHILE

	CMD_END_OF_BUILT_IN
)

type command struct {
	names   []string
	minArgs int
	maxArgs int
	id      CmdID
	fn      func(*Kittla, CmdID, string, []*obj) (*obj, error)

	// For commands in kittla
	args []*obj
	body *obj
}

var builtinCommands = []command{
	{
		names:   []string{"break"},
		minArgs: 0,
		maxArgs: 0,
		id:      CMD_BREAK,
		fn:      cmdBreakContinue,
	},
	{
		names:   []string{"continue"},
		minArgs: 0,
		maxArgs: 0,
		id:      CMD_CONTINUE,
		fn:      cmdBreakContinue,
	},
	{
		names:   []string{"dec", "decr"},
		minArgs: 1,
		maxArgs: 2,
		id:      CMD_DEC,
		fn:      cmdIncDec,
	},
	{
		names:   []string{"elif", "elseif"},
		minArgs: 2,
		maxArgs: 2,
		id:      CMD_ELIF,
		fn:      cmdElIf,
	},
	{
		names:   []string{"else"},
		minArgs: 1,
		maxArgs: 1,
		id:      CMD_ELSE,
		fn:      cmdElse,
	},
	{
		names:   []string{"eval", "expr"},
		minArgs: 1,
		maxArgs: -1,
		id:      CMD_EVAL,
		fn:      cmdEval,
	},
	{
		names:   []string{"float"},
		minArgs: 1,
		maxArgs: 1,
		id:      CMD_FLOAT,
		fn:      cmdFloat,
	},
	{
		names:   []string{"fn"},
		minArgs: 2,
		maxArgs: 3,
		id:      CMD_FN,
		fn:      cmdFn,
	},
	{
		names:   []string{"if"},
		minArgs: 2,
		maxArgs: 2,
		id:      CMD_IF,
		fn:      cmdIf,
	},
	{
		names:   []string{"inc", "incr"},
		minArgs: 1,
		maxArgs: 2,
		id:      CMD_INC,
		fn:      cmdIncDec,
	},
	{
		names:   []string{"int"},
		minArgs: 1,
		maxArgs: 1,
		id:      CMD_INT,
		fn:      cmdInt,
	},
	{
		names:   []string{"loop"},
		minArgs: 1,
		maxArgs: 1,
		id:      CMD_LOOP,
		fn:      cmdLoop,
	},
	{
		names:   []string{"print", "puts"},
		minArgs: 0,
		maxArgs: 1,
		id:      CMD_PRINT,
		fn:      cmdPrint,
	},
	{
		names:   []string{"return"},
		minArgs: 0,
		maxArgs: 1,
		id:      CMD_RETURN,
		fn:      cmdReturn,
	},

	{
		names:   []string{"unknown"},
		minArgs: -1,
		maxArgs: -1,
		id:      CMD_UNKNOWN,
		fn:      cmdUnknown,
	},
	{
		names:   []string{"var", "set"},
		minArgs: 1,
		maxArgs: 2,
		id:      CMD_VAR,
		fn:      cmdVar,
	},
	{
		names:   []string{"while"},
		minArgs: 2,
		maxArgs: 2,
		id:      CMD_WHILE,
		fn:      cmdWhile,
	},
}

func call(k *Kittla, fn *command, cmd string, args []*obj) (*obj, error) {

	k.frames = append(k.frames, k.currFrame)
	newFrame := &frame{objects: make(map[string]*obj), prevCmd: fn.id}

	i := 0
	for i = 0; i < len(args); i++ {
		a, err := k.parse(&codeBlock{code: fn.args[i].toString(), lineNum: k.currLine}, false)
		if err != nil || len(a) == 0 {
			return nil, fmt.Errorf("%s has a malformed argument. Line: %d", cmd, k.currLine)
		}
		newFrame.objects[a[0].toString()] = args[i].clone()
	}

	for ; i < len(fn.args); i++ {
		a, err := k.parse(&codeBlock{code: fn.args[i].toString(), lineNum: k.currLine}, false)
		if err != nil || len(a) == 0 {
			return nil, fmt.Errorf("%s has a malformed argument. Line: %d", cmd, k.currLine)
		}
		newFrame.objects[a[0].toString()] = a[1]
	}

	k.currFrame = newFrame

	res, _, err := k.executeCore(&codeBlock{code: fn.body.toString(), lineNum: k.currLine}, false)

	k.currFrame = k.frames[len(k.frames)-1]
	k.frames = k.frames[:len(k.frames)-1]

	return res, err
}

func callFn(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {

	var fn *command

	for i := range k.commands[cmd] {
		if k.commands[cmd][i].id == cmdID {
			fn = k.commands[cmd][i]
			break
		}
	}

	if fn == nil {
		return nil, fmt.Errorf("Eeeh, there is no command with that id. Line: %d", k.currLine)
	}

	if len(args) < fn.minArgs {
		return nil, fmt.Errorf("Too few arguments. Got %d wants %d. Line: %d", len(args), fn.minArgs, k.currLine)
	}

	if len(args) > fn.maxArgs {
		return nil, fmt.Errorf("Too many arguments. Got %d wants %d. Line: %d", len(args), fn.maxArgs, k.currLine)
	}

	return call(k, fn, cmd, args)

}

func cmdBreakContinue(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {
	switch cmdID {
	case CMD_BREAK:
		k.isBreak = true
	case CMD_CONTINUE:
		k.isContinue = true
	}
	return nil, nil
}

func cmdElIf(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {

	if k.currFrame.prevCmd != CMD_IF && k.currFrame.prevCmd != CMD_ELIF {
		return nil, fmt.Errorf("%s lacks if or else if. Line: %d", cmd, k.currLine)
	}

	if !k.currFrame.ifTaken {
		return cmdIf(k, CMD_IF, "if", args)
	}
	return nil, nil
}

func cmdElse(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {
	if k.currFrame.prevCmd != CMD_IF && k.currFrame.prevCmd != CMD_ELIF {
		return nil, fmt.Errorf("%s lacks if or else if. Line: %d", cmd, k.currLine)
	}

	if !k.currFrame.ifTaken {
		res, _, err := k.executeCore(&codeBlock{code: args[0].toString(), lineNum: k.currLine}, true)
		return res, err
	}
	return nil, nil
}

func exprJoin(args []*obj) (*obj, error) {

	joined := make([]byte, 0, 256)
	for i := range args {
		joined = append(joined, args[i].toBytes()...)
	}

	v, err := expr.Eval(string(joined), nil)
	if err != nil {
		return nil, err
	}
	switch v.Value().(type) {
	case bool:
		return &obj{valType: valTypeBool, valBool: v.Bool()}, nil
	case float64:
		// Is there a better way to see if a float fits in an int?
		if float64(int(v.Float64())) != v.Float64() {
			return &obj{valType: valTypeFloat, valFloat: v.Float64()}, nil
		}
		// why can't I have a fallthrough here?
		return &obj{valType: valTypeInt, valInt: int(v.Int64())}, nil
	case int64:
		return &obj{valType: valTypeInt, valInt: int(v.Int64())}, nil
	case uint64:
		return &obj{valType: valTypeInt, valInt: int(v.Uint64())}, nil
	case string:
		return &obj{valType: valTypeStr, valStr: []byte(v.String())}, nil
	default:
		return nil, fmt.Errorf("expr.Value() returns unknown type: %v", v.Value())
	}
}

func cmdEval(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {
	if res, err := exprJoin(args); err == nil {
		return res, nil
	} else {
		return nil, fmt.Errorf("%s failed with: %v on line: %d", cmd, err, k.currLine)
	}
}

func cmdFloat(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {
	switch args[0].valType {
	case valTypeFloat:
		return args[0], nil
	case valTypeInt:
		args[0].valType = valTypeFloat
		args[0].valFloat = float64(args[0].valInt)
		return args[0], nil
	case valTypeBool:
		return nil, fmt.Errorf("Can't convert boolean to float. Line %d", k.currLine)
	default:
		if v, err := strconv.ParseInt(string(args[0].valStr), 0, 64); err == nil {
			args[0].valType = valTypeFloat
			args[0].valFloat = float64(v)
			return args[0], nil
		}
		if v, err := strconv.ParseFloat(string(args[0].valStr), 64); err == nil {
			args[0].valType = valTypeFloat
			args[0].valFloat = v
			return args[0], nil
		}
	}
	return nil, fmt.Errorf("Can't convert string to float. Line %d", k.currLine)
}

func cmdFn(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {

	var fnName string
	argIdx := 1
	bodyIdx := 2

	if len(args) == 3 {
		fnName = args[0].toString()
	} else { // == 2
		argIdx--
		bodyIdx--
	}

	errFnName := func() string {
		if fnName == "" {
			return "anonymous command"
		} else {
			return "command " + fnName
		}
	}

	fnArgs, err := k.parse(&codeBlock{code: args[argIdx].toString(), lineNum: k.currLine}, false)
	if err != nil {
		return nil, fmt.Errorf("Parsing arguments of %s failed with: %s. Line: %d", errFnName(), err, k.currLine)
	}

	minArgs := 0
	for i := range fnArgs {
		arg, err := k.parse(&codeBlock{code: fnArgs[i].toString(), lineNum: k.currLine}, false)
		if err != nil {
			return nil, fmt.Errorf("Parsing argument \"%s\" of %s failed with: %s. Line: %d", fnArgs[i].toString(), errFnName(), err, k.currLine)

		}

		if len(arg) == 1 {
			minArgs++
		}
	}

	cmdObj := &command{names: []string{fnName}, minArgs: minArgs, maxArgs: len(fnArgs), id: k.nextFnId, fn: callFn,
		args: fnArgs, body: args[bodyIdx]}
	k.nextFnId++

	if fnName != "" {
		replaced := false
		for i := range k.commands[fnName] {
			if k.commands[fnName][i].minArgs == cmdObj.minArgs &&
				k.commands[fnName][i].maxArgs == cmdObj.maxArgs {
				k.commands[fnName][i] = cmdObj
				replaced = true
				break
			}
		}
		if !replaced {
			k.commands[fnName] = append([]*command{cmdObj}, k.commands[fnName]...)
		}
	}

	return &obj{valType: valTypeFn, valFn: cmdObj}, nil
}

func cmdIf(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {

	ifarg, err := k.parse(&codeBlock{code: args[0].toString(), lineNum: k.currLine}, false)
	if err != nil {
		return nil, err
	}

	res, err := exprJoin(ifarg)

	if err != nil {
		return nil, fmt.Errorf("%s failed with: %v on line: %d", cmd, err, k.currLine)
	}

	k.currFrame.ifTaken = res.isTrue()

	if k.currFrame.ifTaken {
		res, _, err := k.executeCore(&codeBlock{code: args[1].toString(), lineNum: k.currLine}, true)
		return res, err
	}

	return nil, nil
}

func cmdIncDec(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {

	o, present := k.currFrame.objects[args[0].toString()]

	if !present {
		return nil, fmt.Errorf("%s: No such variable: %s. Line %d", cmd, args[0].toString(), k.currLine)
	}

	if o.valType != valTypeInt && o.valType != valTypeFloat {
		return nil, fmt.Errorf("First variable isn't a number. Line %d", k.currLine)
	}

	df := 1.0
	d := 1

	if cmdID == CMD_DEC {
		df = -df
		d = -d
	}

	if len(args) == 2 {
		switch args[1].valType {
		case valTypeInt:
			if o.valType != valTypeInt {
				return nil, fmt.Errorf("%s Mismatching types. Line %d", cmd, k.currLine)
			}

			d = d * args[1].valInt
		case valTypeFloat:
			if o.valType != valTypeFloat {
				return nil, fmt.Errorf("%s: Mismatching types. Line %d", cmd, k.currLine)
			}

			df = df * args[1].valFloat
		case valTypeStr:
			if v, err := strconv.ParseInt(args[1].toString(), 0, 64); err == nil {
				if o.valType == valTypeFloat {
					return nil, fmt.Errorf("%s converted to int can't be added to float. Line %d", cmd, k.currLine)
				}
				d = int(v)
			} else if v, err := strconv.ParseFloat(args[1].toString(), 64); err == nil {
				if o.valType == valTypeInt {
					return nil, fmt.Errorf("%s converted to float can't be added to int. Line %d", cmd, k.currLine)
				}
				df = float64(v)
			} else {
				return nil, fmt.Errorf("first argument to %s isn't a number. Line %d", cmd, k.currLine)
			}
		case valTypeBool:
			return nil, fmt.Errorf("Can't do `%s` with boolean. Line %d", cmd, k.currLine)
		}
	}

	switch o.valType {
	case valTypeInt:
		o.valInt += d
		return o, nil
	case valTypeFloat:
		o.valFloat += df
		return o, nil
	}
	return nil, fmt.Errorf("%s: Variable %s is not a number. Line %d", cmd, args[0].toString(), k.currLine)
}

func cmdInt(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {
	switch args[0].valType {
	case valTypeInt:
		return args[0], nil
	case valTypeFloat:
		args[0].valType = valTypeInt
		args[0].valInt = int(args[0].valFloat)
		return args[0], nil
	case valTypeBool:
		return nil, fmt.Errorf("Can't convert boolean to integer. Line %d", k.currLine)
	default:
		if v, err := strconv.ParseInt(string(args[0].valStr), 0, 64); err == nil {
			args[0].valType = valTypeInt
			args[0].valInt = int(v)
			return args[0], nil
		}
		if v, err := strconv.ParseFloat(string(args[0].valStr), 64); err == nil {
			args[0].valType = valTypeInt
			args[0].valInt = int(v)
			return args[0], nil
		}
	}
	return nil, fmt.Errorf("Can't convert string to integer. Line %d", k.currLine)
}

func cmdLoop(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {
	return cmdWhile(k, cmdID, cmd, args)
}

func cmdPrint(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {
	msg := args[0].toBytes()
	fmt.Println(string(msg))
	return &obj{valType: valTypeStr, valStr: msg}, nil
}

func cmdReturn(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {

	if len(args) == 0 {
		return &obj{}, nil
	}
	if res, err := k.parse(&codeBlock{code: args[0].toString(), lineNum: k.currLine}, false); err == nil {
		k.isReturn = true
		if len(res) == 1 {
			return res[0], nil
		} else {
			return nil, fmt.Errorf("Too many objects to return. Line: %d", k.currLine)
		}
	} else {
		return nil, fmt.Errorf("Failed return given object: %v", err)
	}
}

func cmdUnknown(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {
	return nil, fmt.Errorf("Unknown command: %s. Line: %d", cmd, k.currLine)
}

func cmdVar(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {
	varName := args[0].toString()
	switch len(args) {
	case 0:
		return nil, fmt.Errorf("%s command must be followed with one or two arguments. Line: %d", cmd, k.currLine)
	case 1:
		if v, present := k.currFrame.objects[varName]; present {
			return v, nil
		} else {
			return nil, fmt.Errorf("%s: no such variable: %s. Line: %d", cmd, varName, k.currLine)
		}
	case 2:
		k.currFrame.objects[varName] = args[1].optimize()
		return k.currFrame.objects[varName], nil
	default:
		return nil, fmt.Errorf("%s command must be followed with at most two argument. Line: %d", cmd, k.currLine)
	}
}

func cmdWhile(k *Kittla, cmdID CmdID, cmd string, args []*obj) (*obj, error) {

	var res *obj

	loopBodyIdx := 1
	if cmdID == CMD_LOOP {
		loopBodyIdx = 0
	}

	for {
		var err error
		executeBody := true

		if cmdID == CMD_WHILE {
			whileArg, err := k.parse(&codeBlock{code: args[0].toString(), lineNum: k.currLine}, false)

			if err != nil {
				return nil, err
			}

			w, err := exprJoin(whileArg)

			if err != nil {
				return nil, fmt.Errorf("%s failed with: %v on line: %d", cmd, err, k.currLine)
			}
			executeBody = w.isTrue()
		}

		if executeBody {
			res, _, err = k.executeCore(&codeBlock{code: args[loopBodyIdx].toString(), lineNum: k.currLine}, true)
			if err != nil {
				return nil, err
			}
			if k.isBreak {
				k.isBreak = false
				break
			}
			if k.isContinue {
				k.isContinue = false
			}

		} else {
			break
		}
	}
	return res, nil
}

func getCmdMap() map[string][]*command {

	cmdMap := make(map[string][]*command)

	for i := range builtinCommands {
		for j := range builtinCommands[i].names {
			cmdMap[builtinCommands[i].names[j]] = append(cmdMap[builtinCommands[i].names[j]], &builtinCommands[i])
		}

	}
	return cmdMap
}
