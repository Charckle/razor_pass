package tui

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"rpass/internal/fuzzy"
	"rpass/internal/kdbx"
	"rpass/internal/recents"
)

type mode int

const (
	modeRecents mode = iota
	modePassword
	modeDecrypting
	modeTree
	modeSearch
	modeDetail
	modeHelp
)

type row struct {
	group *kdbx.Group
	entry *kdbx.Entry
}

type decryptResult struct {
	db  *kdbx.DB
	err error
}

// Model is the rpass TUI.
type Model struct {
	width, height int
	mode          mode
	helpFrom      mode

	store  *recents.Store
	dbPath string

	recents cursorList

	pwInput textinput.Model
	pwErr   string
	spin    spinner.Model

	db    *kdbx.DB
	stack []*kdbx.Group
	items []row
	tree  cursorList

	searchInput      textinput.Model
	searchFocusQuery bool
	hits             []*kdbx.Entry
	search           cursorList

	detail     *kdbx.Entry
	detailFrom mode
	detailOff  int

	status    string
	clipSeq   int
	clipOwned bool
}

// New starts on recents, or the password prompt if startPath is set.
func New(store *recents.Store, startPath string) Model {
	pw := textinput.New()
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'
	pw.Prompt = ""
	pw.Placeholder = "master password"
	pw.CharLimit = 1024

	se := textinput.New()
	se.Prompt = " / "
	se.Placeholder = "fuzzy search"
	se.CharLimit = 256

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = folderIconStyle

	m := Model{
		width:       80,
		height:      24,
		store:       store,
		pwInput:     pw,
		searchInput: se,
		spin:        sp,
	}
	if startPath != "" {
		m.dbPath = startPath
		m.mode = modePassword
		m.pwInput.Focus()
	} else {
		m.mode = modeRecents
	}
	return m
}

func (m Model) Init() tea.Cmd {
	if m.mode == modePassword {
		return textinput.Blink
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.update(msg)
	return m, cmd
}

func (m *Model) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		cw := msg.Width - 2
		if cw < 1 {
			cw = 1
		}
		_, inner := passwordTileMetrics(cw)
		m.pwInput.Width = inner
		w := cw - 4
		if w < 8 {
			w = 8
		}
		m.searchInput.Width = w
		return nil

	case spinner.TickMsg:
		if m.mode == modeDecrypting {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return cmd
		}

	case decryptResult:
		if msg.err != nil {
			m.mode = modePassword
			m.pwErr = msg.err.Error()
			m.pwInput.SetValue("")
			m.pwInput.Focus()
			return textinput.Blink
		}
		m.db = msg.db
		m.pwErr = ""
		m.pwInput.SetValue("")
		m.store.Touch(m.dbPath)
		_ = m.store.Save()
		m.stack = []*kdbx.Group{m.db.Root}
		m.rebuildTree()
		m.mode = modeTree
		return nil

	case copiedMsg:
		m.handleCopied(msg)
		return nil

	case clearClipMsg:
		m.handleClearClip(msg)
		return nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return nil
}

func (m *Model) typing() bool {
	return m.mode == modePassword || (m.mode == modeSearch && m.searchFocusQuery)
}

func (m *Model) quit() tea.Cmd {
	m.wipeClipboard()
	if m.db != nil {
		m.db.Clear()
		m.db = nil
	}
	return tea.Quit
}

func (m *Model) lock() tea.Cmd {
	if m.db != nil {
		m.db.Clear()
		m.db = nil
	}
	m.stack = nil
	m.items = nil
	m.hits = nil
	m.detail = nil
	m.pwInput.SetValue("")
	m.pwErr = ""
	m.status = ""
	m.store.PruneMissing()
	if len(m.store.Paths) == 0 {
		return m.quit()
	}
	m.mode = modeRecents
	m.recents.cursor = 0
	m.recents.offset = 0
	return nil
}

func (m *Model) rebuildTree() {
	if len(m.stack) == 0 {
		m.items = nil
		m.tree.setLen(0)
		return
	}
	cur := m.stack[len(m.stack)-1]
	groups, entries := cur.Children(len(m.stack) == 1)
	m.items = m.items[:0]
	for _, g := range groups {
		m.items = append(m.items, row{group: g})
	}
	for _, e := range entries {
		m.items = append(m.items, row{entry: e})
	}
	m.tree.setLen(len(m.items))
}

func (m *Model) breadcrumb() string {
	if len(m.stack) <= 1 {
		return "/"
	}
	parts := make([]string, 0, len(m.stack)-1)
	for _, g := range m.stack[1:] {
		parts = append(parts, g.Name)
	}
	return "/" + strings.Join(parts, "/")
}

func (m *Model) openSearch() tea.Cmd {
	m.detail = nil
	m.mode = modeSearch
	m.searchFocusQuery = true
	m.searchInput.SetValue("")
	m.searchInput.Focus()
	m.refreshSearch()
	m.search.cursor = 0
	m.search.offset = 0
	return textinput.Blink
}

func (m *Model) refreshSearch() {
	q := m.searchInput.Value()
	var hits []*kdbx.Entry
	type scored struct {
		e     *kdbx.Entry
		score int
	}
	var ranked []scored
	if m.db != nil {
		for _, e := range m.db.AllEntries() {
			ok, score := fuzzy.Match(q, e.Haystack)
			if ok {
				ranked = append(ranked, scored{e, score})
			}
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return strings.ToLower(ranked[i].e.Title) < strings.ToLower(ranked[j].e.Title)
	})
	hits = make([]*kdbx.Entry, len(ranked))
	for i, r := range ranked {
		hits[i] = r.e
	}
	m.hits = hits
	m.search.setLen(len(m.hits))
}

func (m *Model) revealInTree(e *kdbx.Entry) {
	if m.db == nil || e == nil {
		m.status = "select an entry"
		return
	}
	chain := m.db.Locate(e)
	if len(chain) == 0 {
		m.status = "folder not found"
		return
	}
	m.searchInput.Blur()
	m.searchInput.SetValue("")
	m.detail = nil
	m.stack = chain
	m.rebuildTree()
	for i, item := range m.items {
		if item.entry == e {
			m.tree.cursor = i
			break
		}
	}
	m.tree.reveal(len(m.items), m.bodyHeight())
	m.mode = modeTree
}

func (m *Model) openDetail(e *kdbx.Entry, from mode) {
	m.detail = e
	m.detailFrom = from
	m.detailOff = 0
	m.mode = modeDetail
}

func (m *Model) startUnlock() tea.Cmd {
	pw := m.pwInput.Value()
	if pw == "" {
		m.pwErr = "enter the master password"
		return nil
	}
	m.pwInput.SetValue("")
	m.mode = modeDecrypting
	m.pwErr = ""
	path := m.dbPath
	return tea.Batch(m.spin.Tick, func() tea.Msg {
		db, err := kdbx.Open(path, pw)
		return decryptResult{db: db, err: err}
	})
}

func (m *Model) selectRecent() tea.Cmd {
	if len(m.store.Paths) == 0 {
		return m.quit()
	}
	i := m.recents.cursor
	if i < 0 || i >= len(m.store.Paths) {
		i = 0
	}
	m.dbPath = m.store.Paths[i]
	m.mode = modePassword
	m.pwInput.SetValue("")
	m.pwErr = ""
	m.pwInput.Focus()
	return textinput.Blink
}

func dbLabel(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}
