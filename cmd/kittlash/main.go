package main

import (
	"fmt"
	"kittla"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenPeeDeeP/xdg"
	"github.com/peterh/liner"
)

func createDir(dir string) (err error) {

	if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
		err = os.MkdirAll(dir, 0755)
	}
	return
}

const defaultPrompt = "% "

func main() {

	xdgh := xdg.New("gmelchett", "kittlash")

	if err := createDir(xdgh.ConfigHome()); err != nil {
		log.Fatal("Failed creating config directory", err)
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
			prog.WriteString(cmd)

			switch prog.String() {
			case "/help":
				fmt.Println("help wanted!")
				prog.Reset()
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
			prog.WriteString(";")

			if depth := k.GetNumUnclosed(prog.String()); depth == 0 {
				if res, lastFunc, err := k.Execute(prog.String()); err == nil {
					if lastFunc != kittla.FUNC_PRINT {
						fmt.Println(string(res))
					}
				} else {
					fmt.Printf(" -- Execute error: %v\n", err)
				}
				prog.Reset()
				prompt = defaultPrompt
			} else {
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
