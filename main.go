package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"rpass/internal/recents"
	"rpass/internal/tui"
)

const usage = `rpass - read-only KeePass TUI

Usage:
  rpass              Open the last used database (or pick from recents)
  rpass <file.kdbx>  Open a database file

If no recent databases exist, a path is required.
`

func main() {
	arg, hasArg := "", len(os.Args) > 1
	if hasArg {
		arg = os.Args[1]
		switch arg {
		case "-h", "--help", "help":
			fmt.Fprint(os.Stdout, usage)
			return
		}
	}

	recentsPath, err := recents.DefaultPath()
	if err != nil {
		fatal(err)
	}
	store, err := recents.Load(recentsPath)
	if err != nil {
		fatal(err)
	}
	store.PruneMissing()

	var startPath string
	switch {
	case hasArg:
		p, err := recents.Resolve(arg)
		if err != nil {
			fatal(err)
		}
		startPath = p
	case len(store.Paths) == 0:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	case len(store.Paths) == 1:
		startPath = store.Paths[0]
	}

	m := tui.New(store, startPath)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "rpass: %v\n", err)
	os.Exit(1)
}
