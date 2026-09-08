package fuzzy

import "testing"

func TestMatch(t *testing.T) {
	ok, _ := Match("ghb", "Work / GitHub")
	if !ok {
		t.Fatal("expected subsequence match")
	}
	ok, _ = Match("zzz", "GitHub")
	if ok {
		t.Fatal("expected miss")
	}
	ok, _ = Match("git work", "Work / GitHub personal")
	if !ok {
		t.Fatal("expected AND of terms")
	}
	ok, scoreEmpty := Match("", "anything")
	if !ok || scoreEmpty != 0 {
		t.Fatalf("empty query should match, score=0, got %v %d", ok, scoreEmpty)
	}
}

func TestScorePrefersTighter(t *testing.T) {
	_, a := Match("git", "git")
	_, b := Match("git", "g x i x t")
	if a <= b {
		t.Fatalf("consecutive match should score higher: %d vs %d", a, b)
	}
}
