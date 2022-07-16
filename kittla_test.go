package kittla

import (
	"fmt"
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

		args, err := k.Parse(&CodeBlock{Code: tests[i].code}, false)
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
			if string(tests[i].args[j]) != string(args[j]) {
				t.Logf("Expected arg(%d): *%s* got: *%s*. Test: %d\n", j, string(tests[i].args[j]), string(args[j]), i)
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
}

func TestParser(t *testing.T) {

	for i, te := range parserTests {
		k := New()
		_, err := k.Execute(&CodeBlock{Code: te.program})
		if (err != nil) != te.fails {
			t.Logf("test %d - Failure not matching Excepted failure: %t got: %v\n", i, te.fails, err)
			t.Fail()
			return
		}

		if len(k.objects) != len(te.expects) {
			t.Logf("Objects mismatch, got: %d wanted: %d\n", len(k.objects), len(te.expects))
			spew.Dump(k.objects)
			spew.Dump(te.expects)
			t.Fail()
			return
		}
		for k, v := range k.objects {
			if ev, present := te.expects[string(k)]; present && string(v) != ev {
				t.Logf("Content of \"%s\" mismatch. Got \"%s\" wanted %s\n",
					string(k), string(v), ev)
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
