package kittla

import (
	"fmt"
	"strconv"

	"github.com/tidwall/expr"
)

type funcId int

const (
	FUNC_BREAK funcId = iota
	FUNC_DEC
	FUNC_CONTINUE
	FUNC_ELIF
	FUNC_ELSE
	FUNC_EVAL
	FUNC_IF
	FUNC_INC
	FUNC_PRINT
	FUNC_UNKNOWN
	FUNC_VAR
	FUNC_WHILE
)

type function struct {
	names   []string
	minArgs int
	maxArgs int
	funcId  funcId
	fn      func(*Kittla, funcId, string, [][]byte) ([]byte, error)
}

var builtinFunctions = []function{
	function{
		names:   []string{"break"},
		minArgs: 0,
		maxArgs: 0,
		funcId:  FUNC_BREAK,
		fn:      funcBreakContinue,
	},
	function{
		names:   []string{"continue"},
		minArgs: 0,
		maxArgs: 0,
		funcId:  FUNC_CONTINUE,
		fn:      funcBreakContinue,
	},
	function{
		names:   []string{"dec", "decr"},
		minArgs: 1,
		maxArgs: 2,
		funcId:  FUNC_DEC,
		fn:      funcIncDec,
	},
	function{
		names:   []string{"elif", "elseif"},
		minArgs: 2,
		maxArgs: 2,
		funcId:  FUNC_ELIF,
		fn:      funcElIf,
	},
	function{
		names:   []string{"else"},
		minArgs: 1,
		maxArgs: 1,
		funcId:  FUNC_ELSE,
		fn:      funcElse,
	},
	function{
		names:   []string{"eval", "expr"},
		minArgs: 1,
		maxArgs: -1,
		funcId:  FUNC_EVAL,
		fn:      funcEval,
	},
	function{
		names:   []string{"if"},
		minArgs: 2,
		maxArgs: 2,
		funcId:  FUNC_IF,
		fn:      funcIf,
	},
	function{
		names:   []string{"inc", "incr"},
		minArgs: 1,
		maxArgs: 2,
		funcId:  FUNC_INC,
		fn:      funcIncDec,
	},
	function{
		names:   []string{"print", "puts"},
		minArgs: 0,
		maxArgs: 1,
		funcId:  FUNC_PRINT,
		fn:      funcPrint,
	},
	function{
		names:   []string{"unknown"},
		minArgs: -1,
		maxArgs: -1,
		funcId:  FUNC_UNKNOWN,
		fn:      funcUnknown,
	},
	function{
		names:   []string{"var", "set"},
		minArgs: 1,
		maxArgs: 2,
		funcId:  FUNC_VAR,
		fn:      funcVar,
	},
	function{
		names:   []string{"while"},
		minArgs: 2,
		maxArgs: 2,
		funcId:  FUNC_WHILE,
		fn:      funcWhile,
	},
}

func funcBreakContinue(k *Kittla, funcId funcId, cmd string, args [][]byte) ([]byte, error) {
	switch funcId {
	case FUNC_BREAK:
		k.isBreak = true
	case FUNC_CONTINUE:
		k.isContinue = true
	}
	return nil, nil
}

func funcElIf(k *Kittla, funcId funcId, cmd string, args [][]byte) ([]byte, error) {

	if k.currFrame.prevFunc != FUNC_IF && k.currFrame.prevFunc != FUNC_ELIF {
		return nil, fmt.Errorf("%s lacks if or else if. Line: %d", cmd, k.currLine)
	}

	if !k.currFrame.ifTaken {
		return funcIf(k, FUNC_IF, "if", args)
	}
	return nil, nil
}

func funcElse(k *Kittla, funcId funcId, cmd string, args [][]byte) ([]byte, error) {
	if k.currFrame.prevFunc != FUNC_IF && k.currFrame.prevFunc != FUNC_ELIF {
		return nil, fmt.Errorf("%s lacks if or else if. Line: %d", cmd, k.currLine)
	}

	if !k.currFrame.ifTaken {
		res, _, err := k.executeCore(&codeBlock{code: string(args[0]), lineNum: k.currLine})
		return res, err
	}
	return nil, nil
}

func exprJoin(args [][]byte) (expr.Value, error) {

	joined := make([]byte, 0, 256)
	for i := range args {
		joined = append(joined, args[i]...)
	}

	return expr.Eval(string(joined), nil)
}

func funcEval(k *Kittla, funcId funcId, cmd string, args [][]byte) ([]byte, error) {
	if res, err := exprJoin(args); err == nil {
		return []byte(res.String()), nil
	} else {
		return nil, fmt.Errorf("%s failed with: %v on line: %d", cmd, err, k.currLine)
	}
}

func funcIf(k *Kittla, funcId funcId, cmd string, args [][]byte) ([]byte, error) {

	ifarg, err := k.parse(&codeBlock{code: string(args[0]), lineNum: k.currLine}, false)
	if err != nil {
		return nil, err
	}

	res, err := exprJoin(ifarg)

	if err != nil {
		return nil, fmt.Errorf("%s failed with: %v on line: %d", cmd, err, k.currLine)
	}

	k.currFrame.ifTaken = res.Bool()

	if k.currFrame.ifTaken {
		res, _, err := k.executeCore(&codeBlock{code: string(args[1]), lineNum: k.currLine})
		return res, err
	}

	return []byte(""), nil
}

func funcIncDec(k *Kittla, funcId funcId, cmd string, args [][]byte) ([]byte, error) {
	d := 1
	if len(args) == 2 {
		if v, err := strconv.ParseInt(string(args[1]), 0, 64); err == nil {
			d = int(v)
		} else {
			return nil, fmt.Errorf("%s failed with %v. Line %d", cmd, err, k.currLine)
		}
	}

	if v, present := k.currFrame.objects[string(args[0])]; present {
		if vv, err := strconv.ParseInt(string(v), 0, 64); err == nil {
			var s []byte
			if funcId == FUNC_INC {
				s = []byte(fmt.Sprintf("%d", int(vv)+d))
			} else {
				s = []byte(fmt.Sprintf("%d", int(vv)-d))
			}
			k.currFrame.objects[string(args[0])] = s
			return s, nil
		} else {
			return nil, fmt.Errorf("%s: Variable %s does not contain a number:  %v. Line %d", cmd, string(args[0]), err, k.currLine)
		}

	}
	return nil, fmt.Errorf("%s: No such variable: %s. Line %d", cmd, string(args[0]), k.currLine)
}

func funcPrint(k *Kittla, funcId funcId, cmd string, args [][]byte) ([]byte, error) {
	fmt.Println(string(args[0]))
	return args[0], nil
}

func funcVar(k *Kittla, funcId funcId, cmd string, args [][]byte) ([]byte, error) {
	var result []byte
	varName := string(args[0])
	switch len(args) {
	case 0:
		return nil, fmt.Errorf("%s command must be followed with one or two arguments. Line: %d", cmd, k.currLine)
	case 1:
		if v, present := k.currFrame.objects[varName]; present {
			result = v
		} else {
			return nil, fmt.Errorf("%s: no such variable: %s. Line: %d", cmd, varName, k.currLine)
		}
	case 2:
		k.currFrame.objects[varName] = args[1]
		result = args[1]
	default:
		return nil, fmt.Errorf("%s command must be followed with at most two argument. Line: %d", cmd, k.currLine)
	}
	return result, nil
}

func funcUnknown(k *Kittla, funcId funcId, cmd string, args [][]byte) ([]byte, error) {
	return nil, fmt.Errorf("Unknown command: %s. Line: %d", cmd, k.currLine)
}

func funcWhile(k *Kittla, funcId funcId, cmd string, args [][]byte) ([]byte, error) {

	var res []byte

	for {
		whileArg, err := k.parse(&codeBlock{code: string(args[0]), lineNum: k.currLine}, false)

		if err != nil {
			return nil, err
		}

		w, err := exprJoin(whileArg)

		if err != nil {
			return nil, fmt.Errorf("%s failed with: %v on line: %d", cmd, err, k.currLine)
		}

		if w.Bool() {
			res, _, err = k.executeCore(&codeBlock{code: string(args[1]), lineNum: k.currLine})
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

func getFuncMap() map[string]*function {

	funcMap := make(map[string]*function)

	for i := range builtinFunctions {
		for j := range builtinFunctions[i].names {
			funcMap[builtinFunctions[i].names[j]] = &builtinFunctions[i]
		}

	}
	return funcMap
}
