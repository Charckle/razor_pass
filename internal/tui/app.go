package tui

import (
	"cmp"
	"path/filepath"
	"slices"
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

type scoredEntry struct {
	entry *kdbx.Entry
	score int
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
	ranked           []scoredEntry
	search           cursorList

	detail       *kdbx.Entry
	detailFrom   mode
	detailOff    int
	detailLines  []string
	showPassword bool

	status    string
	clipSeq   int
	clipOwned bool

	file         fileIdent
	fileStale    bool
	fileFlash    bool
	reloadPrompt bool
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
		store:       store,
		pwInput:     pw,
		searchInput: se,
		spin:        sp,
	}
	m.resize(80, 24)
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
		m.resize(msg.Width, msg.Height)
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
			m.pwInput.SetValue("")
			m.pwErr = msg.err.Error()
			m.pwInput.Focus()
			return textinput.Blink
		}
		m.db = msg.db
		m.resetPasswordPrompt()
		m.store.Touch(m.dbPath)
		_ = m.store.Save()
		m.stack = []*kdbx.Group{m.db.Root}
		m.rebuildTree()
		m.mode = modeTree
		m.rememberFile()
		return tickFile()

	case fileTickMsg:
		return m.handleFileTick()

	case fileFlashMsg:
		return m.handleFileFlash()

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

// resize recomputes everything that depends on the terminal size. New calls it
// too, so the first frame is laid out correctly even before the first
// WindowSizeMsg arrives.
func (m *Model) resize(width, height int) {
	m.width = width
	m.height = height
	cw := m.contentWidth()
	_, inner := tileMetrics(cw, passwordTileMaxWidth)
	m.pwInput.Width = inner
	m.searchInput.Width = max(cw-4, 8)
	m.buildDetailLines()
}

func (m *Model) typing() bool {
	if m.mode == modePassword {
		return m.pwInput.Value() != ""
	}
	return m.mode == modeSearch && m.searchFocusQuery
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
	m.clearVault()
	m.resetPasswordPrompt()
	m.status = ""
	m.store.PruneMissing()
	m.mode = modeRecents
	m.recents.cursor = 0
	m.recents.offset = 0
	return nil
}

// resetPasswordPrompt clears the master-password input along with any error or
// reload banner left over from the last prompt.
func (m *Model) resetPasswordPrompt() {
	m.pwInput.SetValue("")
	m.pwErr = ""
	m.reloadPrompt = false
}

func (m *Model) clearVault() {
	if m.db != nil {
		m.db.Clear()
		m.db = nil
	}
	m.stack = nil
	m.items = nil
	m.hits = nil
	m.ranked = nil
	m.clearDetail()
	m.searchInput.Blur()
	m.searchInput.SetValue("")
	m.file = fileIdent{}
	m.fileStale = false
	m.fileFlash = false
}

func (m *Model) rememberFile() {
	m.file = snapshotFile(m.dbPath)
	m.fileStale = false
	m.fileFlash = false
}

func (m *Model) handleFileTick() tea.Cmd {
	if m.db == nil || m.dbPath == "" {
		return nil
	}
	now := snapshotFile(m.dbPath)
	if !m.file.same(now) {
		first := !m.fileStale
		m.fileStale = true
		if first || m.status == "" {
			m.status = fileChangedStatus
		}
		if first {
			m.fileFlash = true
			return tea.Batch(tickFile(), tickFlash())
		}
	}
	return tickFile()
}

func (m *Model) handleFileFlash() tea.Cmd {
	if !m.fileStale {
		m.fileFlash = false
		return nil
	}
	m.fileFlash = !m.fileFlash
	return tickFlash()
}

func (m *Model) startReload() tea.Cmd {
	m.clearVault()
	m.resetPasswordPrompt()
	m.status = ""
	m.reloadPrompt = true
	m.mode = modePassword
	m.pwInput.Focus()
	return textinput.Blink
}

func (m *Model) passwordBack(quitIfNoRecents bool) tea.Cmd {
	m.store.PruneMissing()
	if len(m.store.Paths) == 0 {
		if quitIfNoRecents {
			return m.quit()
		}
		return nil
	}
	m.resetPasswordPrompt()
	m.pwInput.Blur()
	m.mode = modeRecents
	return nil
}

func (m *Model) treeBack() tea.Cmd {
	if len(m.stack) > 1 {
		m.goUp()
		return nil
	}
	return m.lock()
}

func (m *Model) goRoot() tea.Cmd {
	if m.db == nil {
		return nil
	}
	m.searchInput.Blur()
	m.searchInput.SetValue("")
	m.clearDetail()
	m.stack = []*kdbx.Group{m.db.Root}
	m.tree.cursor = 0
	m.tree.offset = 0
	m.rebuildTree()
	m.mode = modeTree
	return nil
}

func (m *Model) closeSearch() tea.Cmd {
	m.searchInput.Blur()
	m.searchInput.SetValue("")
	m.mode = modeTree
	return nil
}

func (m *Model) rebuildTree() {
	if len(m.stack) == 0 {
		m.items = nil
		m.tree.setLen(0)
		return
	}
	cur := m.stack[len(m.stack)-1]
	m.items = m.items[:0]
	for _, g := range cur.Groups {
		m.items = append(m.items, row{group: g})
	}
	for _, e := range cur.Entries {
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
	m.clearDetail()
	m.mode = modeSearch
	m.searchFocusQuery = true
	m.searchInput.SetValue("")
	m.searchInput.Focus()
	m.refreshSearch()
	m.search.cursor = 0
	m.search.offset = 0
	return textinput.Blink
}

// refreshSearch runs on every keystroke, so it leans entirely on values
// precomputed at load time: the flattened entry list, the lowercased
// haystacks, and the lowercased titles used for the tiebreak.
func (m *Model) refreshSearch() {
	q := fuzzy.Compile(m.searchInput.Value())
	m.ranked = m.ranked[:0]
	if m.db != nil {
		for _, e := range m.db.Entries {
			if ok, score := q.Match(e.Haystack); ok {
				m.ranked = append(m.ranked, scoredEntry{entry: e, score: score})
			}
		}
	}
	slices.SortStableFunc(m.ranked, func(a, b scoredEntry) int {
		if a.score != b.score {
			return cmp.Compare(b.score, a.score)
		}
		return cmp.Compare(a.entry.TitleLower, b.entry.TitleLower)
	})
	m.hits = m.hits[:0]
	for _, r := range m.ranked {
		m.hits = append(m.hits, r.entry)
	}
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
	m.clearDetail()
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
	m.showPassword = false
	m.mode = modeDetail
	m.buildDetailLines()
}

func (m *Model) clearDetail() {
	m.detail = nil
	m.detailLines = nil
	m.detailOff = 0
	m.showPassword = false
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
	m.resetPasswordPrompt()
	m.pwInput.Focus()
	return textinput.Blink
}

func dbLabel(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}
