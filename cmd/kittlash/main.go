package main

import (
	"fmt"
	"kittla"
	"log"
	"os"
	"path/filepath"

	"github.com/OpenPeeDeeP/xdg"
	"github.com/peterh/liner"
)

func createDir(dir string) (err error) {

	if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
		err = os.MkdirAll(dir, 0755)
	}
	return
}

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

	for {
		if cmd, err := line.Prompt("% "); err == nil {
			line.AppendHistory(cmd)

			if res, err := k.Execute(&kittla.CodeBlock{Code: cmd}); err == nil {
				fmt.Println(string(res))
			} else {
				fmt.Printf("execute error: %v\n", err)
			}

		} else if err == liner.ErrPromptAborted {
			fmt.Println("Aborted")
			break
		} else {
			fmt.Println("Error reading line: ", err)
			break
		}
	}
}
