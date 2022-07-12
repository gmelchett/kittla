package kittla

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tidwall/expr"
)

type function struct {
	names   []string
	minArgs int
	maxArgs int
	fn      func(*Kittla, string, [][]byte) ([]byte, error)
}

var builtinFunctions = []function{
	function{
		names:   []string{"else"},
		minArgs: 1,
		maxArgs: 1,
		fn:      funcElse,
	},
	function{
		names:   []string{"expr", "eval"},
		minArgs: 1,
		maxArgs: -1,
		fn:      funcExpr,
	},
	function{
		names:   []string{"if"},
		minArgs: 2,
		maxArgs: 2,
		fn:      funcIf,
	},
	function{
		names:   []string{"inc", "incr"},
		minArgs: 1,
		maxArgs: 2,
		fn:      funcInc,
	},
	function{
		names:   []string{"puts", "print"},
		minArgs: 0,
		maxArgs: 1,
		fn:      funcPrint,
	},
	function{
		names:   []string{"set", "var"},
		minArgs: 1,
		maxArgs: 2,
		fn:      funcSet,
	},
	function{
		names:   []string{"unknown"},
		minArgs: -1,
		maxArgs: -1,
		fn:      funcUnknown,
	},
	function{
		names:   []string{"while"},
		minArgs: 2,
		maxArgs: 2,
		fn:      funcWhile,
	},
}

func funcElse(k *Kittla, cmd string, args [][]byte) ([]byte, error) {

	return nil, nil

}

func funcExpr(k *Kittla, cmd string, args [][]byte) ([]byte, error) {
	var s strings.Builder
	for i := range args {
		s.Write(args[i])
		s.WriteString(" ")
	}
	if res, err := expr.Eval(s.String(), nil); err == nil {
		return []byte(res.String()), nil
	} else {
		return nil, fmt.Errorf("%s failed with: %v on line: %d", cmd, err, k.currLine)
	}
}

func funcIf(k *Kittla, cmd string, args [][]byte) ([]byte, error) {

	ifarg, err := k.Parse(&CodeBlock{Code: string(args[0]), LineNum: k.currLine}, false)

	if err != nil {
		return nil, err
	}

	res, err := expr.Eval(string(ifarg[len(ifarg)-1]), nil)

	if err != nil {
		return nil, fmt.Errorf("%s failed with: %v on line: %d", cmd, err, k.currLine)
	}

	if res.Bool() {
		if res, err := k.Execute(&CodeBlock{Code: string(args[1]), LineNum: k.currLine}); err == nil {
			return res, nil
		} else {
			return nil, err
		}
	}

	return []byte(""), nil
}

func funcInc(k *Kittla, cmd string, args [][]byte) ([]byte, error) {
	incVal := 1
	if len(args) == 2 {
		if v, err := strconv.ParseInt(string(args[1]), 0, 64); err == nil {
			incVal = int(v)
		} else {
			return nil, fmt.Errorf("%s failed with %v. Line %d", cmd, err, k.currLine)
		}
	}

	if v, present := k.objects[string(args[0])]; present {
		if vv, err := strconv.ParseInt(string(v), 0, 64); err == nil {
			s := []byte(fmt.Sprintf("%d", int(vv)+incVal))
			k.objects[string(args[0])] = s
			return s, nil
		} else {
			return nil, fmt.Errorf("%s: Variable %s does not contain a number:  %v. Line %d", cmd, string(args[0]), err, k.currLine)
		}
	} else {
		return nil, fmt.Errorf("%s: No such variable: %s. Line %d", cmd, string(args[0]), k.currLine)
	}
}

func funcPrint(k *Kittla, cmd string, args [][]byte) ([]byte, error) {
	fmt.Println(string(args[0]))
	return args[0], nil
}

func funcSet(k *Kittla, cmd string, args [][]byte) ([]byte, error) {
	var result []byte
	switch len(args) {
	case 0:
		return nil, fmt.Errorf("%s command must be followed with one or two arguments. Line: %d", cmd, k.currLine)
	case 1:
		if v, present := k.objects[string(args[0])]; present {
			result = v
		} else {
			return nil, fmt.Errorf("%s: no such variable: %s. Line: %d", cmd, string(args[0]), k.currLine)
		}
	case 2:
		k.objects[string(args[0])] = args[1]
		result = args[1]
	default:
		return nil, fmt.Errorf("%s command must be followed with at most two argument. Line: %d", cmd, k.currLine)
	}
	return result, nil
}

func funcUnknown(k *Kittla, cmd string, args [][]byte) ([]byte, error) {
	return nil, fmt.Errorf("Unknown command: %s. Line: %d", cmd, k.currLine)
}

func funcWhile(k *Kittla, cmd string, args [][]byte) ([]byte, error) {

	var res []byte

	for {

		whileArg, err := k.Parse(&CodeBlock{Code: string(args[0]), LineNum: k.currLine}, false)

		if err != nil {
			return nil, err
		}

		w, err := expr.Eval(string(whileArg[len(whileArg)-1]), nil)

		if err != nil {
			return nil, fmt.Errorf("%s failed with: %v on line: %d", cmd, err, k.currLine)
		}

		if w.Bool() {
			res, err = k.Execute(&CodeBlock{Code: string(args[1]), LineNum: k.currLine})
			if err != nil {
				return nil, err
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
