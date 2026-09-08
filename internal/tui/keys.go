package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	s := msg.String()
	if s == "ctrl+c" {
		return m.quit()
	}

	if m.mode == modeHelp {
		switch s {
		case "esc":
			m.mode = m.helpFrom
			return nil
		case "q":
			return m.quit()
		}
		return nil
	}

	if s == "q" && !m.typing() {
		return m.quit()
	}

	if s == "?" && !m.typing() {
		m.helpFrom = m.mode
		m.mode = modeHelp
		return nil
	}

	switch m.mode {
	case modeRecents:
		return m.keysRecents(s)
	case modePassword:
		return m.keysPassword(msg, s)
	case modeDecrypting:
		return nil
	case modeTree:
		return m.keysTree(s)
	case modeSearch:
		return m.keysSearch(msg, s)
	case modeDetail:
		return m.keysDetail(s)
	}
	return nil
}

func (m *Model) keysRecents(s string) tea.Cmd {
	n := len(m.store.Paths)
	page := m.bodyHeight()
	switch s {
	case "esc":
		return m.quit()
	case "enter", "l", "right":
		return m.selectRecent()
	case "up", "k":
		m.recents.move(-1, n, page)
	case "down", "j":
		m.recents.move(1, n, page)
	case "pgup":
		m.recents.move(-page, n, page)
	case "pgdown":
		m.recents.move(page, n, page)
	case "home", "g":
		m.recents.move(-n, n, page)
	case "end", "G":
		m.recents.move(n, n, page)
	}
	return nil
}

func (m *Model) keysPassword(msg tea.KeyMsg, s string) tea.Cmd {
	switch s {
	case "esc":
		m.pwInput.SetValue("")
		m.pwErr = ""
		m.store.PruneMissing()
		if len(m.store.Paths) == 0 {
			return m.quit()
		}
		m.mode = modeRecents
		return nil
	case "enter":
		return m.startUnlock()
	}
	var cmd tea.Cmd
	m.pwInput, cmd = m.pwInput.Update(msg)
	return cmd
}

func (m *Model) keysTree(s string) tea.Cmd {
	n := len(m.items)
	page := m.bodyHeight()
	switch s {
	case "esc":
		return m.lock()
	case "/", "ctrl+f":
		return m.openSearch()
	case "y":
		return m.copyField(copyPassword)
	case "u":
		return m.copyField(copyUsername)
	case "w":
		return m.copyField(copyURL)
	case "up", "k":
		m.tree.move(-1, n, page)
	case "down", "j":
		m.tree.move(1, n, page)
	case "pgup":
		m.tree.move(-page, n, page)
	case "pgdown":
		m.tree.move(page, n, page)
	case "home", "g":
		m.tree.move(-n, n, page)
	case "end", "G":
		m.tree.move(n, n, page)
	case "left", "h":
		m.goUp()
	case "enter", "l", "right":
		m.openRow()
	}
	return nil
}

func (m *Model) goUp() {
	if len(m.stack) <= 1 {
		return
	}
	m.stack = m.stack[:len(m.stack)-1]
	m.tree.cursor = 0
	m.tree.offset = 0
	m.rebuildTree()
}

func (m *Model) openRow() {
	if m.tree.cursor < 0 || m.tree.cursor >= len(m.items) {
		return
	}
	r := m.items[m.tree.cursor]
	if r.group != nil {
		m.stack = append(m.stack, r.group)
		m.tree.cursor = 0
		m.tree.offset = 0
		m.rebuildTree()
		return
	}
	if r.entry != nil {
		m.openDetail(r.entry, modeTree)
	}
}

func (m *Model) keysSearch(msg tea.KeyMsg, s string) tea.Cmd {
	page := m.bodyHeight() - 2
	if page < 1 {
		page = 1
	}
	n := len(m.hits)

	if m.searchFocusQuery {
		switch s {
		case "esc":
			m.searchInput.Blur()
			m.searchInput.SetValue("")
			m.mode = modeTree
			return nil
		case "down", "ctrl+n", "tab":
			if n > 0 {
				m.searchFocusQuery = false
				m.searchInput.Blur()
			}
			return nil
		case "enter":
			if n > 0 {
				m.searchFocusQuery = false
				m.searchInput.Blur()
			}
			return nil
		}
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.refreshSearch()
		return cmd
	}

	switch s {
	case "esc":
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.mode = modeTree
		return nil
	case "/", "ctrl+f":
		m.searchFocusQuery = true
		m.searchInput.Focus()
		return nil
	case "up", "ctrl+p", "k":
		if m.search.cursor == 0 {
			m.searchFocusQuery = true
			m.searchInput.Focus()
			return nil
		}
		m.search.move(-1, n, page)
	case "down", "ctrl+n", "j":
		m.search.move(1, n, page)
	case "pgup":
		m.search.move(-page, n, page)
	case "pgdown":
		m.search.move(page, n, page)
	case "y":
		return m.copyField(copyPassword)
	case "u":
		return m.copyField(copyUsername)
	case "w":
		return m.copyField(copyURL)
	case "o":
		if n > 0 {
			m.revealInTree(m.hits[m.search.cursor])
		}
	case "enter", "l", "right":
		if n > 0 {
			m.openDetail(m.hits[m.search.cursor], modeSearch)
		}
	}
	return nil
}

func (m *Model) keysDetail(s string) tea.Cmd {
	switch s {
	case "esc", "h", "left":
		m.mode = m.detailFrom
		m.detail = nil
		if m.mode == modeSearch && m.searchFocusQuery {
			m.searchInput.Focus()
		}
		return nil
	case "/", "ctrl+f":
		return m.openSearch()
	case "y":
		return m.copyField(copyPassword)
	case "u":
		return m.copyField(copyUsername)
	case "w":
		return m.copyField(copyURL)
	case "o":
		if m.detail != nil {
			m.revealInTree(m.detail)
		}
		return nil
	case "up", "k":
		if m.detailOff > 0 {
			m.detailOff--
		}
	case "down", "j":
		m.detailOff++
		if m.detailOff > 1000 {
			m.detailOff = 1000
		}
	}
	return nil
}

func (m Model) bodyHeight() int {
	h := m.height - 4 // panel border + header + footer
	if m.status != "" {
		h--
	}
	if h < 1 {
		return 1
	}
	return h
}

func (m Model) contentWidth() int {
	w := m.width - 2
	if w < 1 {
		return 1
	}
	return w
}
