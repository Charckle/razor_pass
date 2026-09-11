package fuzzy

import (
	"strings"
	"testing"
)

// match mirrors how the TUI calls the matcher: compile the query once, then
// test it against an already-lowercased haystack.
func match(query, text string) (bool, int) {
	return Compile(query).Match(strings.ToLower(text))
}

func TestMatch(t *testing.T) {
	ok, _ := match("ghb", "Work / GitHub")
	if !ok {
		t.Fatal("expected subsequence match")
	}
	ok, _ = match("zzz", "GitHub")
	if ok {
		t.Fatal("expected miss")
	}
	ok, _ = match("git work", "Work / GitHub personal")
	if !ok {
		t.Fatal("expected AND of terms")
	}
	ok, scoreEmpty := match("", "anything")
	if !ok || scoreEmpty != 0 {
		t.Fatalf("empty query should match, score=0, got %v %d", ok, scoreEmpty)
	}
}

func TestScorePrefersTighter(t *testing.T) {
	_, a := match("git", "git")
	_, b := match("git", "g x i x t")
	if a <= b {
		t.Fatalf("consecutive match should score higher: %d vs %d", a, b)
	}
}

func TestMatchHandlesMultiByteRunes(t *testing.T) {
	if ok, _ := match("ü", "München"); !ok {
		t.Fatal("expected multi-byte rune match")
	}
	if ok, _ := match("mn", "München"); !ok {
		t.Fatal("expected subsequence across a multi-byte rune")
	}
	if ok, _ := match("nü", "München"); ok {
		t.Fatal("subsequence must respect order")
	}
}
