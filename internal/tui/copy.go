package tui

import (
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"rpass/internal/kdbx"
)

const clipboardTTL = 20 * time.Second

type copyKind string

const (
	copyPassword copyKind = "password"
	copyUsername copyKind = "username"
	copyURL      copyKind = "url"
)

type copiedMsg struct {
	kind copyKind
	seq  int
	err  error
}

type clearClipMsg struct {
	seq int
}

func (m *Model) currentEntry() *kdbx.Entry {
	switch m.mode {
	case modeDetail:
		return m.detail
	case modeSearch:
		if !m.searchFocusQuery && m.search.cursor >= 0 && m.search.cursor < len(m.hits) {
			return m.hits[m.search.cursor]
		}
	case modeTree:
		if m.tree.cursor >= 0 && m.tree.cursor < len(m.items) {
			return m.items[m.tree.cursor].entry
		}
	}
	return nil
}

func (m *Model) copyField(kind copyKind) tea.Cmd {
	e := m.currentEntry()
	if e == nil {
		m.status = "select an entry"
		return nil
	}
	val := e.Password
	label := "password"
	switch kind {
	case copyUsername:
		val = e.Username
		label = "username"
	case copyURL:
		val = e.URL
		label = "url"
	}
	if val == "" {
		m.status = "no " + label
		return nil
	}
	m.clipSeq++
	seq := m.clipSeq
	text := val
	return tea.Batch(
		func() tea.Msg {
			if err := clipboard.WriteAll(text); err != nil {
				return copiedMsg{kind: kind, seq: seq, err: err}
			}
			return copiedMsg{kind: kind, seq: seq}
		},
		tea.Tick(clipboardTTL, func(time.Time) tea.Msg {
			return clearClipMsg{seq: seq}
		}),
	)
}

func (m *Model) handleCopied(msg copiedMsg) {
	if msg.seq != m.clipSeq {
		return
	}
	if msg.err != nil {
		m.status = "clipboard: " + msg.err.Error()
		m.clipOwned = false
		return
	}
	m.clipOwned = true
	m.status = "copied " + string(msg.kind) + ", clipboard clears in 20s"
}

func (m *Model) handleClearClip(msg clearClipMsg) {
	if msg.seq != m.clipSeq || !m.clipOwned {
		return
	}
	_ = clipboard.WriteAll("")
	m.clipOwned = false
	if m.fileStale {
		m.status = fileChangedStatus
	} else if m.status != "" {
		m.status = "clipboard cleared"
	}
}

func (m *Model) wipeClipboard() {
	if m.clipOwned {
		_ = clipboard.WriteAll("")
		m.clipOwned = false
	}
}
