package kdbx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

func mkValue(key, value string) gokeepasslib.ValueData {
	return gokeepasslib.ValueData{Key: key, Value: gokeepasslib.V{Content: value}}
}

func mkProtected(key, value string) gokeepasslib.ValueData {
	return gokeepasslib.ValueData{
		Key:   key,
		Value: gokeepasslib.V{Content: value, Protected: w.NewBoolWrapper(true)},
	}
}

func TestOpenRoundTripAndRecycleLast(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.kdbx")
	const pw = "test-master"

	recycle := gokeepasslib.NewGroup()
	recycle.Name = "Recycle Bin"
	recycle.UUID = gokeepasslib.NewUUID()

	alpha := gokeepasslib.NewGroup()
	alpha.Name = "Zebra"

	work := gokeepasslib.NewGroup()
	work.Name = "Alpha"

	entry := gokeepasslib.NewEntry()
	entry.Values = []gokeepasslib.ValueData{
		mkValue("Title", "GitHub"),
		mkValue("UserName", "ada"),
		mkProtected("Password", "s3cret"),
		mkValue("URL", "https://github.com"),
		mkValue("Notes", "work account"),
		mkValue("PIN", "1234"),
	}
	work.Entries = append(work.Entries, entry)

	root := gokeepasslib.NewGroup()
	root.Name = "Root"
	root.Groups = []gokeepasslib.Group{recycle, alpha, work}

	meta := gokeepasslib.NewMetaData()
	meta.RecycleBinUUID = recycle.UUID
	meta.RecycleBinEnabled = w.NewBoolWrapper(true)

	db := &gokeepasslib.Database{
		Header:      gokeepasslib.NewHeader(),
		Credentials: gokeepasslib.NewPasswordCredentials(pw),
		Content: &gokeepasslib.DBContent{
			Meta: meta,
			Root: &gokeepasslib.RootData{Groups: []gokeepasslib.Group{root}},
		},
	}
	if err := db.LockProtectedEntries(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := gokeepasslib.NewEncoder(f).Encode(db); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	opened, err := Open(path, pw)
	if err != nil {
		t.Fatal(err)
	}
	groups, entries := opened.Root.Children(true)
	if len(entries) != 0 {
		t.Fatalf("root entries: %d", len(entries))
	}
	if len(groups) != 3 {
		t.Fatalf("groups: %#v", names(groups))
	}
	if groups[0].Name != "Alpha" || groups[1].Name != "Zebra" || groups[2].Name != "Recycle Bin" {
		t.Fatalf("order: %#v", names(groups))
	}
	if !groups[2].IsRecycleBin {
		t.Fatal("expected recycle bin flag")
	}

	all := opened.AllEntries()
	if len(all) != 1 || all[0].Title != "GitHub" || all[0].Password != "s3cret" {
		t.Fatalf("entries: %+v", all)
	}
	if all[0].Path != "Alpha" {
		t.Fatalf("path should omit root group, got %q", all[0].Path)
	}
	loc := opened.Locate(all[0])
	if len(loc) != 2 || loc[0].Name != "Root" || loc[1].Name != "Alpha" {
		t.Fatalf("locate: %#v", names(loc))
	}
	if all[0].Haystack == "" || containsPassword(all[0].Haystack, "s3cret") {
		t.Fatalf("haystack should omit password: %q", all[0].Haystack)
	}
	if all[0].Extra[0].Key != "PIN" {
		t.Fatalf("extra: %+v", all[0].Extra)
	}

	if _, err := Open(path, "wrong"); err == nil {
		t.Fatal("wrong password should fail")
	}
}

func names(groups []*Group) []string {
	out := make([]string, len(groups))
	for i, g := range groups {
		out[i] = g.Name
	}
	return out
}

func containsPassword(haystack, pw string) bool {
	return len(pw) > 0 && (haystack == pw || (len(haystack) >= len(pw) && indexOf(haystack, pw) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
