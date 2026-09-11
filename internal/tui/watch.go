package tui

import (
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	filePollInterval  = time.Second
	fileFlashInterval = 400 * time.Millisecond
	fileChangedStatus = "vault file changed — press r to reload"
)

type fileIdent struct {
	size    int64
	modTime time.Time
	missing bool
}

type fileTickMsg struct{}

type fileFlashMsg struct{}

func snapshotFile(path string) fileIdent {
	fi, err := os.Stat(path)
	if err != nil {
		return fileIdent{missing: true}
	}
	return fileIdent{size: fi.Size(), modTime: fi.ModTime()}
}

func (a fileIdent) same(b fileIdent) bool {
	if a.missing || b.missing {
		return a.missing && b.missing
	}
	return a.size == b.size && a.modTime.Equal(b.modTime)
}

func tickFile() tea.Cmd {
	return tea.Tick(filePollInterval, func(time.Time) tea.Msg {
		return fileTickMsg{}
	})
}

func tickFlash() tea.Cmd {
	return tea.Tick(fileFlashInterval, func(time.Time) tea.Msg {
		return fileFlashMsg{}
	})
}
