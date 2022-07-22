package kittla

import "sort"

// Helper functions for the kittla shell

// Counts the number of unclosed [] and {} pairs and make sure that
// there is an even number of ". Escaped characters are taken into account.
func (k *Kittla) GetNumUnclosed(prog string) int {

	cb := &codeBlock{code: prog, lineNum: 1}

	stringOpen := false
	preDepth := 0
	postDepth := 0

	for {
		c := cb.next()
		if c == '\\' {
			cb.next()
			if cb.eof {
				break
			}
			continue
		}

		if stringOpen {
			if c == '"' {
				stringOpen = false
			}
			continue
		}

		switch c {
		case '"':
			stringOpen = true
		case '[':
			preDepth++
		case ']':
			preDepth--
		case '{':
			postDepth++
		case '}':
			postDepth--
		default:
		}

		if cb.eof {
			break
		}
	}
	totDepth := preDepth + postDepth
	if stringOpen {
		totDepth++
	}
	return totDepth
}

// Fetches a list of all commands plus variables. Returned alphabetically sorted.
// Useful for shell tab completion
func (k *Kittla) Names() []string {

	names := make([]string, 0, 1024)

	for i := range k.commands {
		for j := range k.commands[i].names {
			names = append(names, k.commands[i].names[j])
		}
	}

	// FIXME: Consider what to add..
	for i := range k.currFrame.objects {
		names = append(names, i)
	}

	sort.Strings(names)
	return names
}
