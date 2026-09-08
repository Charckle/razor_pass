package tui

type cursorList struct {
	cursor int
	offset int
}

func (l *cursorList) setLen(n int) {
	if n <= 0 {
		l.cursor = 0
		l.offset = 0
		return
	}
	if l.cursor >= n {
		l.cursor = n - 1
	}
	if l.cursor < 0 {
		l.cursor = 0
	}
	l.reveal(n, 1)
}

func (l *cursorList) move(delta, n, page int) {
	if n <= 0 {
		l.cursor = 0
		l.offset = 0
		return
	}
	l.cursor += delta
	if l.cursor < 0 {
		l.cursor = 0
	}
	if l.cursor >= n {
		l.cursor = n - 1
	}
	if page < 1 {
		page = 1
	}
	l.reveal(n, page)
}

func (l *cursorList) reveal(n, page int) {
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+page {
		l.offset = l.cursor - page + 1
	}
	if l.offset < 0 {
		l.offset = 0
	}
	maxOff := n - page
	if maxOff < 0 {
		maxOff = 0
	}
	if l.offset > maxOff {
		l.offset = maxOff
	}
}
