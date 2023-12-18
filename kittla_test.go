package kittla

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

type testParseData struct {
	code  string
	args  []string
	fails bool
}

func TestParse(t *testing.T) {

	tests := []testParseData{
		{"{}", []string{""}, false},
		{"5;", []string{"5"}, false},
		{"hej5;", []string{"hej5"}, false},
		{"hej", []string{"hej"}, false},
		{"  hej  ", []string{"hej"}, false},
		{"  hej  ;", []string{"hej"}, false},
		{"  hej  ;\n", []string{"hej"}, false},
		{"  hej  \n", []string{"hej"}, false},

		{"hej  hopp", []string{"hej", "hopp"}, false},
		{"hej  hopp;", []string{"hej", "hopp"}, false},
		{"  hej hopp\n", []string{"hej", "hopp"}, false},
		{"hej  hopp  hipp", []string{"hej", "hopp", "hipp"}, false},
		{"hej hopp hipp  ;", []string{"hej", "hopp", "hipp"}, false},
		{"hej hopp hipp  \n", []string{"hej", "hopp", "hipp"}, false},

		{"hej {hopp hipp}\n", []string{"hej", "hopp hipp"}, false},
		{"hej {hopp hipp};\n", []string{"hej", "hopp hipp"}, false},

		{"hej {hopp hipp}", []string{"hej", "hopp hipp"}, false},
		{"hej \"hopp hipp\"", []string{"hej", "hopp hipp"}, false},
		{"if {1 == 2} {puts a}", []string{"if", "1 == 2", "puts a"}, false},
	}

	k := New()

	for i := range tests {

		args, err := k.parse(&codeBlock{code: tests[i].code}, false)
		if (err != nil) != tests[i].fails {
			t.Logf("Expected failure: %t got: %v. Test: %d\n", tests[i].fails, err, i)
			t.Fail()
			return
		}
		if len(tests[i].args) != len(args) {
			fmt.Println(tests[i].args, args)
			t.Logf("Expected %d args got: %d. Test: %d\n", len(tests[i].args), len(args), i)
			t.Fail()
			return
		}

		for j := range tests[i].args {
			if string(tests[i].args[j]) != args[j].toString() {
				t.Logf("Expected arg(%d): *%s* got: *%s*. Test: %d\n", j, string(tests[i].args[j]), args[j].toString(), i)
				t.Fail()
				return
			}
		}
	}
}

type parserTest struct {
	program string
	expects map[string]string
	fails   bool
}

var parserTests = []parserTest{
	{
		program: "set a 4;",
		expects: map[string]string{
			"a": "4",
		},
	},

	{
		program: "set a 4;set b 5;",
		expects: map[string]string{
			"a": "4",
			"b": "5",
		},
	},

	{
		program: "set a 4;set b $a;",
		expects: map[string]string{
			"a": "4",
			"b": "4",
		},
	},

	{
		program: "set a [set b 7];",
		expects: map[string]string{
			"a": "7",
			"b": "7",
		},
	},
	{
		program: "set a [set b 7];",
		expects: map[string]string{
			"a": "7",
			"b": "7",
		},
	},
	{
		program: "set a [set b [set c 7]];",
		expects: map[string]string{
			"a": "7",
			"b": "7",
			"c": "7",
		},
	},
	{
		program: "set a [set b 7][set c 99];",
		expects: map[string]string{
			"a": "799",
			"b": "7",
			"c": "99",
		},
	},
	{
		program: "if {1} {set b 2}",
		expects: map[string]string{
			"b": "2",
		},
	},
	{
		program: "if {5 == 6} {set b 2}",
		expects: map[string]string{},
	},
	{
		program: "if {77 == 77} {set b 2;set c 3;}",
		expects: map[string]string{
			"b": "2",
			"c": "3",
		},
	},
	{
		program: "set ahej 77; if {$ahej == 77} {set b 2;set c 3;}",
		expects: map[string]string{
			"ahej": "77",
			"b":    "2",
			"c":    "3",
		},
	},
	{
		program: "set a 78; if {$a == 77} {set b 2;set c 3;}",
		expects: map[string]string{
			"a": "78",
		},
	},
	{
		program: "set a 1; inc a; set b 66; inc b 1",
		expects: map[string]string{
			"a": "2",
			"b": "67",
		},
	},
	{
		program: "set a 1; inc a 5; set b 66; inc b $a",
		expects: map[string]string{
			"a": "6",
			"b": "72",
		},
	},

	{
		program: "set ii 1; set b 66; while {$ii < 10} {inc ii; inc b 1}",
		expects: map[string]string{
			"ii": "10",
			"b":  "75",
		},
	},

	{
		program: "set ii 1; set b 66; while {$b < $ii} {inc ii; inc b 1}",
		expects: map[string]string{
			"ii": "1",
			"b":  "66",
		},
	},

	{
		program: "set res 0; set input 5; if {$res == 0} {inc input}; else {dec input}",
		expects: map[string]string{
			"res":   "0",
			"input": "6",
		},
	},
	{
		program: "set res 1; set input 5; if {$res == 0} {inc input}; else {dec input}",
		expects: map[string]string{
			"res":   "1",
			"input": "4",
		},
	},

	{
		program: "set res 2; set input 5; if {$res == 0} {inc input}; elseif {$res == 2} {inc input 2; inc res}; else {dec input 2};",
		expects: map[string]string{
			"res":   "3",
			"input": "7",
		},
	},
	{
		program: "set res 3; set input 5; if {$res == 0} {inc input}; elseif {$res == 2} {inc input 2; inc res}; elseif {$res == 3} {inc input 4; inc res 4}; else {dec input 2};",
		expects: map[string]string{
			"res":   "7",
			"input": "9",
		},
	},
	{
		program: "set i 0; while {$i < 10} { inc i }",
		expects: map[string]string{
			"i": "10",
		},
	},
	{
		program: "set i 0; while {$i < 10} { inc i; if {$i == 5} { break } }",
		expects: map[string]string{
			"i": "5",
		},
	},
	{
		program: "set j 0; set i 0; while {$i < 10} { inc i; if {$i == 5} { continue }; inc j }",
		expects: map[string]string{
			"i": "10",
			"j": "9",
		},
	},
	{
		program: "set tot 0; set i 0; while {$i < 10} { inc i; set j 0; while {$j < 10} { inc j; inc tot }}",
		expects: map[string]string{
			"i":   "10",
			"j":   "10",
			"tot": "100",
		},
	},
	{
		program: "set tot 0; set i 0; while {$i < 10} { inc i; set j 0; while {$j < 10} { inc j; if {$i > 5} { break }; inc tot}}",
		expects: map[string]string{
			"i":   "10",
			"j":   "1",
			"tot": "50",
		},
	},
	{
		program: "set l 0; ; loop { inc l; if {$l > 5} { break }}",
		expects: map[string]string{
			"l": "6",
		},
	},

	{
		program: "set l 0; ; inc l 100;",
		expects: map[string]string{
			"l": "100",
		},
	},

	{
		program: "set l 0; ; inc l",
		expects: map[string]string{
			"l": "1",
		},
	},
	{
		program: "set l 0.0; ; inc l 1.0",
		expects: map[string]string{
			"l": "1.000000",
		},
	},
	{
		program: "set l 1.2; ; inc l",
		expects: map[string]string{
			"l": "2.200000",
		},
	},
	{
		program: "set l 1.2; ; inc l 1;",
		fails:   true,
		expects: map[string]string{
			"l": "1.200000",
		},
	},
	{
		program: "set l 1; ; inc l 0.1;",
		fails:   true,
		expects: map[string]string{
			"l": "1",
		},
	},
	{
		program: "set l [int 7.5]",
		expects: map[string]string{
			"l": "7",
		},
	},
	{
		program: "set l [int \"7.5\"]",
		expects: map[string]string{
			"l": "7",
		},
	},
	{
		program: "set l [int 7]",
		expects: map[string]string{
			"l": "7",
		},
	},
	{
		program: "set l [float 7]",
		expects: map[string]string{
			"l": "7.000000",
		},
	},
	{
		program: "set l [float 7.5]",
		expects: map[string]string{
			"l": "7.500000",
		},
	},
	{
		program: "set l [float \"7.5\"]",
		expects: map[string]string{
			"l": "7.500000",
		},
	},
	{
		program: "fn test {} {set b 1;}; set a [test];",
		expects: map[string]string{
			"a": "1",
		},
	},
	{
		program: "fn test {a} {set b $a;}; set a [test 5];",
		expects: map[string]string{
			"a": "5",
		},
	},
	{
		program: "fn test {b} {set b 7;}; set a [test 5];",
		expects: map[string]string{
			"a": "7",
		},
	},

	{
		program: "fn test {a {c 4}} {inc a $c;}; set a [test 5];",
		expects: map[string]string{
			"a": "9",
		},
	},
	{
		program: "fn test {a {c 4}} {inc a $c;}; set a [test 5 3];",
		expects: map[string]string{
			"a": "8",
		},
	},
	{
		program: "fn test {a {c 4}} {inc a $c;}; set a [test];",
		fails:   true,
	},
	{
		program: "fn test {a {c 4}} {inc a $c;}; set a [test 1 2 3];",
		fails:   true,
	},

	{
		program: "fn test {} {return 1;}; set a [test];",
		expects: map[string]string{
			"a": "1",
		},
	},
	{
		program: "fn test {a} {set b $a;return 2;}; set a [test 77];",
		expects: map[string]string{
			"a": "2",
		},
	},

	{
		program: "fn test {a} {set b $a;return 2;}; fn test {a b} {set b $a;return 3;}; set a [test 77 8];",
		expects: map[string]string{
			"a": "3",
		},
	},
	{
		program: "fn test {a} {set b $a;return 2;}; fn test {a b} {set b $a;return 3;}; set a [test 77];",
		expects: map[string]string{
			"a": "2",
		},
	},

	{
		program: "fn test {} {return 2;return 3;return 4}; set a [test];",
		expects: map[string]string{
			"a": "2",
		},
	},
	{
		program: "set hello [fn {} {return 2;}]; set a [hello];",
		expects: map[string]string{
			"a":     "2",
			"hello": "return 2;",
		},
	},
	{
		program: "set a [list 1 2 3];set b [len a];",
		expects: map[string]string{
			"a": "(1, 2, 3)",
			"b": "3",
		},
	},
	{
		program: "set b [width 1234];",
		expects: map[string]string{
			"b": "4",
		},
	},

	{
		program: "set a [list];set b [len a];",
		expects: map[string]string{
			"a": "()",
			"b": "0",
		},
	},
	{
		program: "set a [list 1]; append a 2 3;set b [len a];",
		expects: map[string]string{
			"a": "(1, 2, 3)",
			"b": "3",
		},
	},
	{
		program: "set a hej; set a [concat $a \" hopp\"];set b [width $a];",
		expects: map[string]string{
			"a": "hej hopp",
			"b": "8",
		},
	},
	{
		program: "set a 1; set a [concat $a 2];set b [width $a];",
		expects: map[string]string{
			"a": "12",
			"b": "2",
		},
	},

	{
		program: "set a [width abcdef];",
		expects: map[string]string{
			"a": "6",
		},
	},
	{
		program: "set a [list 1 2 3]; set b [last a]",
		expects: map[string]string{
			"a": "(1, 2, 3)",
			"b": "3",
		},
	},
	{
		program: "set k 3; goadd k 3",
		expects: map[string]string{
			"k": "6",
		},
	},
}

func testAdd(k *Kittla, args []string) (string, error) {
	var v int
	if v_str, present := k.GetVar(args[0]); present {
		var err error
		v, err = strconv.Atoi(v_str)
		if err != nil {
			return "", fmt.Errorf("'%s' does not contain a number.", args[0])
		}
	}

	v2, err := strconv.Atoi(args[1])
	if err != nil {
		return "", fmt.Errorf("Second argument '%s' is not a  number.", args[1])
	}
	k.SetVar(args[0], strconv.Itoa(v+v2))
	return "", nil
}

func TestParser(t *testing.T) {

	for i, te := range parserTests {
		k := New()
		k.AddFunction("goadd", 2, 2, testAdd)

		_, _, err := k.Execute(te.program)
		if (err != nil) != te.fails {
			t.Logf("test %d - Failure not matching Excepted failure: %t got: %v\n", i, te.fails, err)
			t.Fail()
			return
		}

		if len(k.currFrame.objects) != len(te.expects) {
			t.Logf("Test: %d Objects mismatch, got: %d wanted: %d\n", i, len(k.currFrame.objects), len(te.expects))
			spew.Dump(k.currFrame.objects)
			spew.Dump(te.expects)
			t.Fail()
			return
		}
		for k, v := range k.currFrame.objects {
			if ev, present := te.expects[string(k)]; present && v.toString() != ev {
				t.Logf("Test: %d Content of \"%s\" mismatch. Got \"%s\" wanted %s\n",
					i,
					string(k), v.toString(), ev)
				t.Logf("Test: %s\n", te.program)
				t.Fail()
				return
			} else if !present {
				t.Logf("Object with name: %s is missing\n", string(k))
				t.Fail()
				return
			}
		}
	}
}
