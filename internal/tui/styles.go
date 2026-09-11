package tui

import "github.com/charmbracelet/lipgloss"

const (
	iconFolder = "▸"
	iconEntry  = "•"
)

var (
	colHeaderBg   = lipgloss.Color("166") // burnt orange
	colFooterBg   = lipgloss.Color("130")
	colCream      = lipgloss.Color("230")
	colAmber      = lipgloss.Color("214")
	colEntry      = lipgloss.Color("223")
	colSelectedBg = lipgloss.Color("173")
	colSelectedFg = lipgloss.Color("235")
	colDim        = lipgloss.Color("138")
	colBorder     = lipgloss.Color("172")
	colRecycle    = lipgloss.Color("180")
	colRed        = lipgloss.Color("167")
	colGreen      = lipgloss.Color("150")

	headerStyle = lipgloss.NewStyle().
			Background(colHeaderBg).
			Foreground(colCream).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Background(colFooterBg).
			Foreground(colCream)

	selectedStyle = lipgloss.NewStyle().
			Background(colSelectedBg).
			Foreground(colSelectedFg).
			Bold(true)

	groupStyle = lipgloss.NewStyle().
			Foreground(colAmber).
			Bold(true)

	folderIconStyle = lipgloss.NewStyle().
			Foreground(colAmber).
			Bold(true)

	recycleStyle = lipgloss.NewStyle().
			Foreground(colRecycle).
			Bold(true)

	entryStyle = lipgloss.NewStyle().
			Foreground(colEntry)

	entryIconStyle = lipgloss.NewStyle().
			Foreground(colDim)

	dimStyle = lipgloss.NewStyle().
			Foreground(colDim)

	errStyle = lipgloss.NewStyle().
			Foreground(colRed)

	okStyle = lipgloss.NewStyle().
		Foreground(colGreen)

	warnStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colSelectedFg).
			Background(colAmber)

	warnFlashStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colCream).
			Background(colRed)

	titleStyle = lipgloss.NewStyle().
			Foreground(colCream).
			Bold(true)

	tileStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colAmber).
			Padding(1, 2)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder)

	// wrapStyle is unstyled; callers set a width on a copy of it to word-wrap.
	wrapStyle = lipgloss.NewStyle()
)

const (
	footerText           = " ? help   / search   y copy   u user   w url   r reload   o folder   Q root   esc back   q back/quit"
	recentsFooterText    = " enter open    j/k move    q quit    ? help"
	passwordTileMaxWidth = 54
	recentsTileMaxWidth  = 72
)

func tileWidth(termW, maxW int) int {
	w := maxW
	if avail := termW - 4; avail > 0 && avail < w {
		w = avail
	}
	if w < 22 {
		w = termW
		if w < 18 {
			w = 18
		}
	}
	return w
}

func tileMetrics(termW, maxW int) (styleW, innerW int) {
	styleW = max(tileWidth(termW, maxW)-2, 16)
	innerW = max(styleW-4, 8)
	return styleW, innerW
}
