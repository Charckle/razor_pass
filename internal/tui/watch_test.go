package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"rpass/internal/kdbx"
)

func TestFileIdentSame(t *testing.T) {
	t0 := time.Unix(1_700_000_000, 0)
	a := fileIdent{size: 10, modTime: t0}
	if !a.same(fileIdent{size: 10, modTime: t0}) {
		t.Fatal("identical idents should match")
	}
	if a.same(fileIdent{size: 11, modTime: t0}) {
		t.Fatal("size change should not match")
	}
	if a.same(fileIdent{size: 10, modTime: t0.Add(time.Second)}) {
		t.Fatal("mtime change should not match")
	}
	missing := fileIdent{missing: true}
	if a.same(missing) {
		t.Fatal("missing vs present should not match")
	}
	if !missing.same(fileIdent{missing: true}) {
		t.Fatal("both missing should match")
	}
}

func TestSnapshotFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.kdbx")
	if !snapshotFile(path).missing {
		t.Fatal("absent file should be missing")
	}
	if err := os.WriteFile(path, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := snapshotFile(path)
	if a.missing || a.size != 3 {
		t.Fatalf("got %#v", a)
	}
	if !a.same(snapshotFile(path)) {
		t.Fatal("unchanged file should match")
	}
	if err := os.WriteFile(path, []byte("abcd"), 0o600); err != nil {
		t.Fatal(err)
	}
	if a.same(snapshotFile(path)) {
		t.Fatal("rewritten file should not match")
	}
}

func TestHandleFileTickDetectsChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.kdbx")
	if err := os.WriteFile(path, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := Model{dbPath: path, db: &kdbx.DB{}}
	m.rememberFile()
	if cmd := m.handleFileTick(); cmd == nil {
		t.Fatal("expected tick to continue")
	}
	if m.fileStale || m.status != "" {
		t.Fatalf("unchanged: stale=%v status=%q", m.fileStale, m.status)
	}

	if err := os.WriteFile(path, []byte("abcd"), 0o600); err != nil {
		t.Fatal(err)
	}
	m.handleFileTick()
	if !m.fileStale || !m.fileFlash || m.status != fileChangedStatus {
		t.Fatalf("changed: stale=%v flash=%v status=%q", m.fileStale, m.fileFlash, m.status)
	}

	m.status = "copied password"
	m.handleFileTick()
	if m.status != "copied password" {
		t.Fatalf("later tick clobbered status: %q", m.status)
	}

	m.status = ""
	m.handleFileTick()
	if m.status != fileChangedStatus {
		t.Fatalf("empty status should restore prompt: %q", m.status)
	}

	m.db = nil
	if cmd := m.handleFileTick(); cmd != nil {
		t.Fatal("locked session should stop watching")
	}
}

func TestHandleFileFlashToggles(t *testing.T) {
	m := Model{fileStale: true, fileFlash: true}
	if cmd := m.handleFileFlash(); cmd == nil || m.fileFlash {
		t.Fatalf("expected toggle off, flash=%v cmd=%v", m.fileFlash, cmd != nil)
	}
	m.handleFileFlash()
	if !m.fileFlash {
		t.Fatal("expected toggle on")
	}
	m.fileStale = false
	if cmd := m.handleFileFlash(); cmd != nil || m.fileFlash {
		t.Fatal("flash should stop when not stale")
	}
}

func TestStartReloadClearsVault(t *testing.T) {
	m := New(nil, "vault.kdbx")
	m.db = &kdbx.DB{}
	m.fileStale = true
	m.status = "x"
	m.mode = modeTree
	m.startReload()
	if m.db != nil || m.mode != modePassword || !m.reloadPrompt || m.fileStale || m.status != "" {
		t.Fatalf("mode=%v db=%v reload=%v stale=%v status=%q", m.mode, m.db != nil, m.reloadPrompt, m.fileStale, m.status)
	}
}
