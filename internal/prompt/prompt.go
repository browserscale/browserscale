// Package prompt provides tiny stdlib-only interactive prompts so the CLI
// stays dependency-free.
package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var reader = bufio.NewReader(os.Stdin)

// Choice is a single selectable option in a menu.
type Choice struct {
	// Value is returned when the option is picked.
	Value string
	// Label is shown to the user.
	Label string
}

// String asks for free text, returning def when the user just hits enter.
func String(label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// Select renders a numbered menu and returns the chosen option's Value.
// defValue selects the default entry (matched against Choice.Value).
func Select(label string, choices []Choice, defValue string) string {
	defIdx := 0
	for i, c := range choices {
		if c.Value == defValue {
			defIdx = i
		}
	}

	fmt.Println(label)
	for i, c := range choices {
		marker := " "
		if i == defIdx {
			marker = "*"
		}
		fmt.Printf("  %s %d) %s\n", marker, i+1, c.Label)
	}

	for {
		fmt.Printf("Select [%d]: ", defIdx+1)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return choices[defIdx].Value
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(choices) {
			fmt.Println("  please enter a number from the list")
			continue
		}
		return choices[n-1].Value
	}
}

// Bool asks a yes/no question.
func Bool(label string, def bool) bool {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	for {
		fmt.Printf("%s [%s]: ", label, hint)
		line, _ := reader.ReadString('\n')
		line = strings.ToLower(strings.TrimSpace(line))
		switch line {
		case "":
			return def
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("  please answer y or n")
		}
	}
}

// Int asks for an integer, returning def on empty input.
func Int(label string, def int) int {
	for {
		fmt.Printf("%s [%d]: ", label, def)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			fmt.Println("  please enter a whole number")
			continue
		}
		return n
	}
}
