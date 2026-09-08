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
	colFolder     = lipgloss.Color("214")
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
			Foreground(colFolder).
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
)

const (
	footerText           = " ? help   / search   y copy   u user   w url   o folder   esc back   q quit"
	passwordTileMaxWidth = 54
)

func passwordTileWidth(termW int) int {
	w := passwordTileMaxWidth
	if max := termW - 4; max > 0 && max < w {
		w = max
	}
	if w < 22 {
		w = termW
		if w < 18 {
			w = 18
		}
	}
	return w
}

func passwordTileMetrics(termW int) (styleW, inputW int) {
	styleW = passwordTileWidth(termW) - 2
	if styleW < 16 {
		styleW = 16
	}
	inputW = styleW - 4
	if inputW < 8 {
		inputW = 8
	}
	return styleW, inputW
}
