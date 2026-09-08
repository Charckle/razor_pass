package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width <= 0 {
		return "rpass"
	}
	w := m.contentWidth()
	header := m.viewHeader()
	footer := footerStyle.Width(w).Render(trunc(footerText, w))
	var status string
	if m.status != "" {
		status = okStyle.Width(w).Render(" " + trunc(m.status, w-1))
	}
	used := 4 // panel border + header + footer
	if status != "" {
		used++
	}
	bodyH := m.height - used
	if bodyH < 1 {
		bodyH = 1
	}
	body := m.fill(m.viewBody(bodyH), bodyH)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(body)
	if status != "" {
		b.WriteString("\n")
		b.WriteString(status)
	}
	b.WriteString("\n")
	b.WriteString(footer)
	return panelStyle.Render(b.String())
}

func (m Model) viewHeader() string {
	left := " rpass"
	switch m.mode {
	case modeRecents:
		left += "  recent databases"
	case modePassword, modeDecrypting:
		left += "  " + m.dbPath
	case modeTree, modeSearch, modeDetail:
		left += "  " + m.breadcrumb()
	case modeHelp:
		left += "  help"
	}
	right := dbLabel(m.dbPath)
	w := m.contentWidth()
	gap := w - lipgloss.Width(left) - lipgloss.Width(right) - 1
	if gap < 1 {
		s := trunc(left, w)
		return headerStyle.Width(w).Render(s)
	}
	line := left + strings.Repeat(" ", gap) + right + " "
	return headerStyle.Width(w).Render(trunc(line, w))
}

func (m Model) viewBody(h int) string {
	switch m.mode {
	case modeRecents:
		return m.viewRecents(h)
	case modePassword:
		return m.viewPassword(h)
	case modeDecrypting:
		return m.viewDecrypting(h)
	case modeTree:
		return m.viewTree(h)
	case modeSearch:
		return m.viewSearch(h)
	case modeDetail:
		return m.viewDetail(h)
	case modeHelp:
		return m.viewHelp(h)
	}
	return ""
}

func (m Model) viewRecents(h int) string {
	w := m.contentWidth()
	if len(m.store.Paths) == 0 {
		return "  no databases"
	}
	lines := make([]string, 0, h)
	end := m.recents.offset + h
	for i := m.recents.offset; i < len(m.store.Paths) && i < end; i++ {
		p := m.store.Paths[i]
		plain := " " + iconEntry + " " + p
		if i == m.recents.cursor {
			lines = append(lines, selectedStyle.Width(w).Render(trunc(plain, w)))
		} else {
			lines = append(lines, trunc(plain, w))
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewPassword(h int) string {
	_, inputW := passwordTileMetrics(m.contentWidth())
	m.pwInput.Width = inputW
	err := " "
	if m.pwErr != "" {
		err = errStyle.Render(trunc(m.pwErr, inputW))
	}
	body := strings.Join([]string{
		m.pwInput.View(),
		"",
		err,
	}, "\n")
	return m.centerTile(h, "Unlock", body, "enter unlock    esc back")
}

func (m Model) viewDecrypting(h int) string {
	body := strings.Join([]string{
		m.spin.View() + " decrypting…",
		"",
		" ",
	}, "\n")
	return m.centerTile(h, "Unlock", body, "")
}

func (m Model) centerTile(h int, title, body, hint string) string {
	styleW, inner := passwordTileMetrics(m.contentWidth())
	file := dbLabel(m.dbPath)
	if file == "" {
		file = m.dbPath
	}
	header := titleStyle.Render(title) + "\n" + dimStyle.Render(trunc(file, inner))
	parts := []string{header, "", body}
	if hint != "" {
		parts = append(parts, "", dimStyle.Render(hint))
	}
	tile := tileStyle.Width(styleW).Render(strings.Join(parts, "\n"))
	return lipgloss.Place(m.contentWidth(), h, lipgloss.Center, lipgloss.Center, tile)
}

func (m Model) viewTree(h int) string {
	if len(m.items) == 0 {
		return dimStyle.Render("  (empty group)")
	}
	lines := make([]string, 0, h)
	end := m.tree.offset + h
	for i := m.tree.offset; i < len(m.items) && i < end; i++ {
		lines = append(lines, m.renderTreeRow(i, i == m.tree.cursor))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderTreeRow(i int, selected bool) string {
	r := m.items[i]
	if r.group != nil {
		label := r.group.Name
		style := groupStyle
		iconSt := folderIconStyle
		if r.group.IsRecycleBin {
			style = recycleStyle
			iconSt = recycleStyle
		}
		n := len(r.group.Groups) + len(r.group.Entries)
		extra := fmt.Sprintf("(%d)", n)
		return m.rowLine(iconFolder, label, extra, iconSt, style, selected)
	}
	label := r.entry.Title
	if label == "" {
		label = "(untitled)"
	}
	return m.rowLine(iconEntry, label, r.entry.Username, entryIconStyle, entryStyle, selected)
}

func (m Model) viewSearch(h int) string {
	query := m.searchInput.View()
	if !m.searchFocusQuery {
		query = dimStyle.Render(query)
	}
	rest := h - 2
	if rest < 1 {
		rest = 1
	}
	var body string
	if len(m.hits) == 0 {
		body = dimStyle.Render("  no matches")
	} else {
		lines := make([]string, 0, rest)
		end := m.search.offset + rest
		for i := m.search.offset; i < len(m.hits) && i < end; i++ {
			e := m.hits[i]
			label := e.Title
			if e.Path != "" {
				label = e.Path + " / " + e.Title
			}
			sel := !m.searchFocusQuery && i == m.search.cursor
			lines = append(lines, m.rowLine(iconEntry, label, e.Username, entryIconStyle, entryStyle, sel))
		}
		body = strings.Join(lines, "\n")
	}
	return query + "\n" + dimStyle.Render(strings.Repeat("─", m.contentWidth())) + "\n" + body
}

func (m Model) viewDetail(h int) string {
	e := m.detail
	if e == nil {
		return ""
	}
	mask := "(empty)"
	if e.Password != "" {
		mask = "********"
	}
	lines := []string{
		"  " + titleStyle.Render(orDash(e.Title)),
		"  " + dimStyle.Render("group    ") + e.Path,
		"  " + dimStyle.Render("user     ") + orDash(e.Username),
		"  " + dimStyle.Render("password ") + mask,
		"  " + dimStyle.Render("url      ") + orDash(e.URL),
	}
	for _, f := range e.Extra {
		lines = append(lines, "  "+dimStyle.Render(padKey(f.Key))+" "+f.Value)
	}
	if e.Notes != "" {
		lines = append(lines, "  "+dimStyle.Render("notes"))
		w := m.contentWidth() - 4
		if w < 8 {
			w = 8
		}
		wrapped := lipgloss.NewStyle().Width(w).Render(e.Notes)
		for _, ln := range strings.Split(wrapped, "\n") {
			lines = append(lines, "  "+ln)
		}
	}
	start := m.detailOff
	if start < 0 {
		start = 0
	}
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + h
	if end > len(lines) {
		end = len(lines)
	}
	out := lines[start:end]
	for i := range out {
		out[i] = trunc(out[i], m.contentWidth())
	}
	return strings.Join(out, "\n")
}

func (m Model) viewHelp(h int) string {
	text := strings.TrimSpace(`
  navigation
    j k  ↑ ↓     move
    enter  l     open group or entry
    h  ←         parent group
    g  G         top / bottom

  find & copy
    /  ctrl+f    search
    type         filter as you go
    ↓  tab       jump to results
    ↑            back to query
    y            copy password
    u            copy username
    w            copy url
    o            open that entry's folder in the tree
    esc          close search / back
    q            quit (not while typing)
    ?            this help

  recents
    enter        open selected database
    rpass FILE   add a database (saved as an absolute path)
`)
	lines := strings.Split(text, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for i := range lines {
		lines[i] = trunc(lines[i], m.contentWidth())
	}
	return strings.Join(lines, "\n")
}

func (m Model) rowLine(icon, label, extra string, iconSt, labelSt lipgloss.Style, selected bool) string {
	w := m.contentWidth()
	prefix := " " + icon + " "
	extraPart := ""
	if extra != "" {
		extraPart = "  " + extra
	}
	budget := w - lipgloss.Width(prefix) - lipgloss.Width(extraPart)
	if budget < 1 {
		extraPart = ""
		budget = w - lipgloss.Width(prefix)
	}
	if budget < 1 {
		budget = 1
	}
	label = trunc(label, budget)
	plain := prefix + label + extraPart
	if selected {
		return selectedStyle.Width(w).Render(trunc(plain, w))
	}
	line := iconSt.Render(prefix) + labelSt.Render(label)
	if extraPart != "" {
		line += dimStyle.Render(extraPart)
	}
	pad := w - lipgloss.Width(plain)
	if pad > 0 {
		line += strings.Repeat(" ", pad)
	}
	return line
}

func (m Model) fill(s string, h int) string {
	width := m.contentWidth()
	lines := strings.Split(s, "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	for i, ln := range lines {
		lw := lipgloss.Width(ln)
		if lw < width {
			lines[i] = ln + strings.Repeat(" ", width-lw)
		}
	}
	return strings.Join(lines, "\n")
}

func trunc(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	r := []rune(s)
	low, high := 0, len(r)
	for low < high {
		mid := (low + high + 1) / 2
		if lipgloss.Width(string(r[:mid])+"…") <= w {
			low = mid
		} else {
			high = mid - 1
		}
	}
	if low <= 0 {
		return "…"
	}
	return string(r[:low]) + "…"
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func padKey(k string) string {
	if len(k) >= 8 {
		return k
	}
	return k + strings.Repeat(" ", 8-len(k))
}
