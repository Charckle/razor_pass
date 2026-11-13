package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	s := msg.String()
	if s == "ctrl+c" {
		return m.quit()
	}

	if s == "Q" && m.db != nil && !m.typing() {
		return m.goRoot()
	}

	if m.mode == modeHelp {
		if s == "esc" || s == "q" {
			m.mode = m.helpFrom
		}
		return nil
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

// moveList applies the list-navigation bindings shared by every scrolling
// pane. It reports false if s is not one of them.
func moveList(l *cursorList, s string, n, page int) bool {
	switch s {
	case "up", "k":
		l.move(-1, n, page)
	case "down", "j":
		l.move(1, n, page)
	case "pgup":
		l.move(-page, n, page)
	case "pgdown":
		l.move(page, n, page)
	case "home", "g":
		l.move(-n, n, page)
	case "end", "G":
		l.move(n, n, page)
	default:
		return false
	}
	return true
}

// vaultKeys applies the bindings shared by the tree, search results and detail
// panes. It reports false if s is not one of them.
func (m *Model) vaultKeys(s string) (tea.Cmd, bool) {
	switch s {
	case "/", "ctrl+f":
		return m.openSearch(), true
	case "r":
		return m.startReload(), true
	case "y":
		return m.copyField(copyPassword), true
	case "u":
		return m.copyField(copyUsername), true
	case "w":
		return m.copyField(copyURL), true
	}
	return nil, false
}

func (m *Model) keysRecents(s string) tea.Cmd {
	switch s {
	case "esc":
		return nil
	case "q":
		return m.quit()
	case "enter", "l", "right":
		return m.selectRecent()
	}
	moveList(&m.recents, s, len(m.store.Paths), m.recentsPage())
	return nil
}

func (m *Model) keysPassword(msg tea.KeyMsg, s string) tea.Cmd {
	switch s {
	case "esc":
		return m.passwordBack(false)
	case "q":
		if m.pwInput.Value() == "" {
			return m.passwordBack(true)
		}
	case "enter":
		return m.startUnlock()
	}
	var cmd tea.Cmd
	m.pwInput, cmd = m.pwInput.Update(msg)
	return cmd
}

func (m *Model) keysTree(s string) tea.Cmd {
	switch s {
	case "esc", "q":
		return m.treeBack()
	case "left", "h":
		m.goUp()
		return nil
	case "enter", "l", "right":
		m.openRow()
		return nil
	}
	if moveList(&m.tree, s, len(m.items), m.bodyHeight()) {
		return nil
	}
	cmd, _ := m.vaultKeys(s)
	return cmd
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
	page := max(m.bodyHeight()-2, 1)
	n := len(m.hits)

	if m.searchFocusQuery {
		switch s {
		case "esc":
			return m.closeSearch()
		case "down", "ctrl+n", "tab", "enter":
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
	case "esc", "q":
		return m.closeSearch()
	case "/", "ctrl+f":
		m.searchFocusQuery = true
		m.searchInput.Focus()
		return nil
	case "up", "ctrl+p", "k":
		// Moving off the top of the results returns to the query.
		if m.search.cursor == 0 {
			m.searchFocusQuery = true
			m.searchInput.Focus()
			return nil
		}
		m.search.move(-1, n, page)
		return nil
	case "ctrl+n":
		m.search.move(1, n, page)
		return nil
	case "o":
		if n > 0 {
			m.revealInTree(m.hits[m.search.cursor])
		}
		return nil
	case "enter", "l", "right":
		if n > 0 {
			m.openDetail(m.hits[m.search.cursor], modeSearch)
		}
		return nil
	}
	if moveList(&m.search, s, n, page) {
		return nil
	}
	cmd, _ := m.vaultKeys(s)
	return cmd
}

func (m *Model) keysDetail(s string) tea.Cmd {
	switch s {
	case "esc", "q", "h", "left":
		m.mode = m.detailFrom
		m.clearDetail()
		if m.mode == modeSearch && m.searchFocusQuery {
			m.searchInput.Focus()
		}
		return nil
	case "o":
		if m.detail != nil {
			m.revealInTree(m.detail)
		}
		return nil
	case "s":
		if m.detail == nil {
			return nil
		}
		m.showPassword = !m.showPassword
		m.buildDetailLines()
		return nil
	case "up", "k":
		if m.detailOff > 0 {
			m.detailOff--
		}
		return nil
	case "down", "j":
		if m.detailOff < m.detailMaxOff(m.bodyHeight()) {
			m.detailOff++
		}
		return nil
	}
	cmd, _ := m.vaultKeys(s)
	return cmd
}

func (m Model) recentsPage() int {
	page := m.bodyHeight() - 10
	if page < 1 {
		page = 1
	}
	if page > 12 {
		page = 12
	}
	return page
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
