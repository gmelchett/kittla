package main

import (
	"fmt"
	"kittla"

	"github.com/peterh/liner"
)

func main() {

	line := liner.NewLiner()
	defer line.Close()

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
