package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"kittla"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenPeeDeeP/xdg"
	"github.com/peterh/liner"
)

const defaultPrompt = "% "

func interactive() {

	xdgh := xdg.New("gmelchett", "kittlash")

	if stat, err := os.Stat(xdgh.ConfigHome()); err != nil || !stat.IsDir() {
		err = os.MkdirAll(xdgh.ConfigHome(), 0755)
		if err != nil {
			log.Fatal("Failed creating config directory", err)
		}
	}

	historyFile := filepath.Join(xdgh.ConfigHome(), "history.txt")

	line := liner.NewLiner()
	defer line.Close()

	if f, err := os.Open(historyFile); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	defer func() {
		if f, err := os.Create(historyFile); err == nil {
			line.WriteHistory(f)
			f.Close()
		} else {
			log.Fatal("Error writing history file: ", err)
		}
	}()

	line.SetCtrlCAborts(true)

	k := kittla.New()

	// TODO: struct with shell commands and functions.
	shcmds := []string{"/help", "/quit", "/reset"}

	line.SetCompleter(func(line string) (c []string) {
		for _, n := range append(k.Names(), shcmds...) {
			if strings.HasPrefix(n, strings.ToLower(line)) {
				c = append(c, n)
			}
		}
		return
	})

	prompt := defaultPrompt

	var prog strings.Builder
mainloop:
	for {
		if cmd, err := line.Prompt(prompt); err == nil {
			line.AppendHistory(cmd)

			switch cmd {
			case "/help":
				fmt.Println("help wanted!")
				prog.Reset()
				continue mainloop
			case "/list":
				fmt.Println("Commands and variables:")
				fmt.Println(k.Names())
				continue mainloop
			case "/reset":
				fmt.Println(" -- Reset kittla instance")
				k = kittla.New()
				prog.Reset()
				continue mainloop
			case "/quit":
				break mainloop
			case "":
				fmt.Println("/help for help")
				continue mainloop
			default:
			}
			prog.WriteString(cmd)

			if depth := k.GetNumUnclosed(prog.String()); depth == 0 {
				prog.WriteString(";")
				if res, lastFunc, err := k.Execute(prog.String()); err == nil {
					if lastFunc != kittla.CMD_PRINT {
						fmt.Println(string(res))
					}
				} else {
					fmt.Printf(" -- Execute error: %v\n", err)
				}
				prog.Reset()
				prompt = defaultPrompt
			} else if depth < 0 {
				fmt.Println("To many closing } and/or ]")
				prog.Reset()
				prompt = defaultPrompt
			} else {
				prog.WriteString(" ")
				prompt = "... " + strings.Repeat("    ", depth)
			}

		} else if err == liner.ErrPromptAborted {
			fmt.Println("Aborted")
			break
		} else {
			fmt.Println("\n -- Reset input")
			prompt = defaultPrompt
			prog.Reset()
		}
	}

}

func execute(prog string) {
	if res, lastFunc, err := kittla.New().Execute(prog); err == nil {
		if lastFunc != kittla.CMD_PRINT {
			fmt.Println(string(res))
		}
	} else {
		fmt.Println("Script failed with:", err)
		os.Exit(1)
	}
}

func main() {

	var prog string

	flag.StringVar(&prog, "e", "", "kittla code to execute and then quit. Prints the final result.")

	flag.Parse()

	if len(prog) > 0 {
		execute(prog)
	} else if len(flag.Args()) == 0 {
		interactive()
	} else if len(flag.Args()) == 1 {
		if d, err := ioutil.ReadFile(flag.Args()[0]); err == nil {
			execute(string(d))
		} else {
			fmt.Println("Failed to read given file:", err)
			os.Exit(1)
		}

	} else {
		fmt.Println("Too many arguments.")
		os.Exit(1)
	}

}
