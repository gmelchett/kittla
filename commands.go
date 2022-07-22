package kittla

import (
	"fmt"
	"strconv"

	"github.com/tidwall/expr"
)

type cmdId int

const (
	CMD_BREAK cmdId = iota
	CMD_DEC
	CMD_CONTINUE
	CMD_ELIF
	CMD_ELSE
	CMD_EVAL
	CMD_IF
	CMD_INC
	CMD_PRINT
	CMD_UNKNOWN
	CMD_VAR
	CMD_WHILE
)

type command struct {
	names   []string
	minArgs int
	maxArgs int
	id      cmdId
	fn      func(*Kittla, cmdId, string, [][]byte) ([]byte, error)
}

var builtinCommands = []command{
	command{
		names:   []string{"break"},
		minArgs: 0,
		maxArgs: 0,
		id:      CMD_BREAK,
		fn:      cmdBreakContinue,
	},
	command{
		names:   []string{"continue"},
		minArgs: 0,
		maxArgs: 0,
		id:      CMD_CONTINUE,
		fn:      cmdBreakContinue,
	},
	command{
		names:   []string{"dec", "decr"},
		minArgs: 1,
		maxArgs: 2,
		id:      CMD_DEC,
		fn:      cmdIncDec,
	},
	command{
		names:   []string{"elif", "elseif"},
		minArgs: 2,
		maxArgs: 2,
		id:      CMD_ELIF,
		fn:      cmdElIf,
	},
	command{
		names:   []string{"else"},
		minArgs: 1,
		maxArgs: 1,
		id:      CMD_ELSE,
		fn:      cmdElse,
	},
	command{
		names:   []string{"eval", "expr"},
		minArgs: 1,
		maxArgs: -1,
		id:      CMD_EVAL,
		fn:      cmdEval,
	},
	command{
		names:   []string{"if"},
		minArgs: 2,
		maxArgs: 2,
		id:      CMD_IF,
		fn:      cmdIf,
	},
	command{
		names:   []string{"inc", "incr"},
		minArgs: 1,
		maxArgs: 2,
		id:      CMD_INC,
		fn:      cmdIncDec,
	},
	command{
		names:   []string{"print", "puts"},
		minArgs: 0,
		maxArgs: 1,
		id:      CMD_PRINT,
		fn:      cmdPrint,
	},
	command{
		names:   []string{"unknown"},
		minArgs: -1,
		maxArgs: -1,
		id:      CMD_UNKNOWN,
		fn:      cmdUnknown,
	},
	command{
		names:   []string{"var", "set"},
		minArgs: 1,
		maxArgs: 2,
		id:      CMD_VAR,
		fn:      cmdVar,
	},
	command{
		names:   []string{"while"},
		minArgs: 2,
		maxArgs: 2,
		id:      CMD_WHILE,
		fn:      cmdWhile,
	},
}

func cmdBreakContinue(k *Kittla, cmdId cmdId, cmd string, args [][]byte) ([]byte, error) {
	switch cmdId {
	case CMD_BREAK:
		k.isBreak = true
	case CMD_CONTINUE:
		k.isContinue = true
	}
	return nil, nil
}

func cmdElIf(k *Kittla, cmdId cmdId, cmd string, args [][]byte) ([]byte, error) {

	if k.currFrame.prevCmd != CMD_IF && k.currFrame.prevCmd != CMD_ELIF {
		return nil, fmt.Errorf("%s lacks if or else if. Line: %d", cmd, k.currLine)
	}

	if !k.currFrame.ifTaken {
		return cmdIf(k, CMD_IF, "if", args)
	}
	return nil, nil
}

func cmdElse(k *Kittla, cmdId cmdId, cmd string, args [][]byte) ([]byte, error) {
	if k.currFrame.prevCmd != CMD_IF && k.currFrame.prevCmd != CMD_ELIF {
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

func cmdEval(k *Kittla, cmdId cmdId, cmd string, args [][]byte) ([]byte, error) {
	if res, err := exprJoin(args); err == nil {
		return []byte(res.String()), nil
	} else {
		return nil, fmt.Errorf("%s failed with: %v on line: %d", cmd, err, k.currLine)
	}
}

func cmdIf(k *Kittla, cmdId cmdId, cmd string, args [][]byte) ([]byte, error) {

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

func cmdIncDec(k *Kittla, cmdId cmdId, cmd string, args [][]byte) ([]byte, error) {
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
			if cmdId == CMD_INC {
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

func cmdPrint(k *Kittla, cmdId cmdId, cmd string, args [][]byte) ([]byte, error) {
	fmt.Println(string(args[0]))
	return args[0], nil
}

func cmdVar(k *Kittla, cmdId cmdId, cmd string, args [][]byte) ([]byte, error) {
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

func cmdUnknown(k *Kittla, cmdId cmdId, cmd string, args [][]byte) ([]byte, error) {
	return nil, fmt.Errorf("Unknown command: %s. Line: %d", cmd, k.currLine)
}

func cmdWhile(k *Kittla, cmdId cmdId, cmd string, args [][]byte) ([]byte, error) {

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

func getCmdMap() map[string]*command {

	cmdMap := make(map[string]*command)

	for i := range builtinCommands {
		for j := range builtinCommands[i].names {
			cmdMap[builtinCommands[i].names[j]] = &builtinCommands[i]
		}

	}
	return cmdMap
}
