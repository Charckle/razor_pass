package recents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAndTouch(t *testing.T) {
	dir := t.TempDir()
	rel := filepath.Join(dir, "vault.kdbx")
	if err := os.WriteFile(rel, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	abs, err := Resolve(rel)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(abs) {
		t.Fatalf("expected absolute path, got %q", abs)
	}

	storePath := filepath.Join(dir, "recents.json")
	s, err := Load(storePath)
	if err != nil {
		t.Fatal(err)
	}
	s.Touch(abs)
	s.Touch(filepath.Join(dir, "missing.kdbx"))
	s.PruneMissing()
	if len(s.Paths) != 1 || s.Paths[0] != abs {
		t.Fatalf("pruned store = %#v", s.Paths)
	}

	s.Touch(abs)
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	s2, err := Load(storePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(s2.Paths) != 1 || s2.Paths[0] != abs {
		t.Fatalf("reloaded = %#v", s2.Paths)
	}
}

func TestResolveMissing(t *testing.T) {
	_, err := Resolve(filepath.Join(t.TempDir(), "nope.kdbx"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTouchOrder(t *testing.T) {
	s := &Store{}
	s.Touch("a")
	s.Touch("b")
	s.Touch("a")
	if len(s.Paths) != 2 || s.Paths[0] != "a" || s.Paths[1] != "b" {
		t.Fatalf("got %#v", s.Paths)
	}
}
