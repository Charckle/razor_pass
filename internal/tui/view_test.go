package tui

import (
	"strings"
	"testing"

	"rpass/internal/kdbx"
	"rpass/internal/recents"
)

func TestHardWrap(t *testing.T) {
	got := hardWrap("abcdefghij", 4)
	want := []string{"abcd", "efgh", "ij"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("hardWrap = %q, want %q", got, want)
	}
	if got := hardWrap("short", 10); len(got) != 1 || got[0] != "short" {
		t.Fatalf("short string should stay one line: %q", got)
	}
}

func TestPasswordReveal(t *testing.T) {
	m := New(&recents.Store{}, "")
	m.resize(40, 24)
	secret := "correct-horse-battery"
	e := &kdbx.Entry{Title: "Mail", Password: secret}

	m.openDetail(e, modeTree)
	joined := strings.Join(m.detailLines, "\n")
	if strings.Contains(joined, secret) {
		t.Fatal("password should be masked until revealed")
	}
	if !strings.Contains(joined, "********") {
		t.Fatal("masked password should show asterisks")
	}

	m.showPassword = true
	m.buildDetailLines()
	joined = stripANSI(strings.Join(m.detailLines, "\n"))
	if !strings.Contains(joined, secret) {
		t.Fatalf("revealed password missing from detail:\n%s", joined)
	}

	m.clearDetail()
	if m.showPassword {
		t.Fatal("leaving the entry should hide the password")
	}
	m.openDetail(e, modeTree)
	if m.showPassword {
		t.Fatal("opening an entry should start masked")
	}
}

func TestPasswordRevealWraps(t *testing.T) {
	m := New(&recents.Store{}, "")
	m.resize(24, 24) // content width 22; password budget is 11
	secret := "abcdefghijkLMNOP"
	m.showPassword = true
	lines := m.passwordLines(&kdbx.Entry{Password: secret})
	if len(lines) < 2 {
		t.Fatalf("long password should wrap, got %d line(s)", len(lines))
	}
	var got strings.Builder
	for i, ln := range lines {
		plain := stripANSI(ln)
		if i == 0 {
			plain = strings.TrimPrefix(plain, "  password ")
		} else {
			plain = strings.TrimLeft(plain, " ")
		}
		got.WriteString(plain)
	}
	if got.String() != secret {
		t.Fatalf("wrapped password = %q, want %q", got.String(), secret)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
