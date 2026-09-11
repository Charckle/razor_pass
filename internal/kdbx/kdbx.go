package kdbx

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/tobischo/gokeepasslib/v3"
)

var skipSearchKeys = map[string]struct{}{
	"Title":    {},
	"UserName": {},
	"Password": {},
	"URL":      {},
	"Notes":    {},
}

// DB is an in-memory, read-only KeePass tree. Everything is derived once in
// Open — the tree never changes afterwards, so nothing here is recomputed.
type DB struct {
	Root *Group

	// Entries is every entry in the tree, flattened in display order.
	Entries []*Entry
}

// Group is a KeePass group (folder). Groups and Entries are sorted A–Z at
// load time, with the Recycle Bin pinned last at the top level. Both slices
// are shared with the tree; callers must not mutate them.
type Group struct {
	Name         string
	IsRecycleBin bool
	Groups       []*Group
	Entries      []*Entry

	parent    *Group
	nameLower string
}

// Entry is a KeePass password entry.
type Entry struct {
	Title    string
	Username string
	Password string
	URL      string
	Notes    string
	Tags     string
	Path     string // group path, e.g. "Work / GitHub"
	Extra    []Field

	// Haystack is every field except the password, concatenated and
	// lowercased for search. TitleLower backs the search tiebreak. Both are
	// precomputed so that matching never re-lowercases the whole tree.
	Haystack   string
	TitleLower string

	group *Group
}

// Field is a non-standard string field.
type Field struct {
	Key, Value string
}

// Open decrypts path with a master password and returns a read-only tree.
func Open(path, password string) (*DB, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	raw := gokeepasslib.NewDatabase()
	raw.Credentials = gokeepasslib.NewPasswordCredentials(password)
	if err := gokeepasslib.NewDecoder(f).Decode(raw); err != nil {
		return nil, fmt.Errorf("invalid master password or file")
	}
	if err := raw.UnlockProtectedEntries(); err != nil {
		return nil, fmt.Errorf("invalid master password or file")
	}
	if raw.Content == nil || raw.Content.Root == nil || len(raw.Content.Root.Groups) == 0 {
		return nil, fmt.Errorf("empty database")
	}

	var recycle gokeepasslib.UUID
	if raw.Content.Meta != nil {
		recycle = raw.Content.Meta.RecycleBinUUID
	}

	root := convertGroup(&raw.Content.Root.Groups[0], nil, "", recycle, true)
	return &DB{Root: root, Entries: flatten(root, nil)}, nil
}

func convertGroup(g *gokeepasslib.Group, parent *Group, parentPath string, recycle gokeepasslib.UUID, isRoot bool) *Group {
	out := &Group{
		Name:         g.Name,
		IsRecycleBin: !recycle.IsZero() && g.UUID.Compare(recycle),
		parent:       parent,
		nameLower:    strings.ToLower(g.Name),
	}

	// KeePass has a single root group (often named "Root" or the DB name).
	// Omit it from paths — every entry lives under it.
	path := parentPath
	if !isRoot && g.Name != "" {
		if parentPath == "" {
			path = g.Name
		} else {
			path = parentPath + " / " + g.Name
		}
	}

	out.Groups = make([]*Group, 0, len(g.Groups))
	for i := range g.Groups {
		out.Groups = append(out.Groups, convertGroup(&g.Groups[i], out, path, recycle, false))
	}
	out.Entries = make([]*Entry, 0, len(g.Entries))
	for i := range g.Entries {
		out.Entries = append(out.Entries, convertEntry(&g.Entries[i], out, path))
	}

	slices.SortFunc(out.Groups, func(a, b *Group) int {
		return cmp.Compare(a.nameLower, b.nameLower)
	})
	slices.SortFunc(out.Entries, func(a, b *Entry) int {
		return cmp.Compare(a.TitleLower, b.TitleLower)
	})

	// The Recycle Bin is always a child of the root group; keep it last so it
	// stays out of the way.
	if isRoot {
		if i := slices.IndexFunc(out.Groups, func(g *Group) bool { return g.IsRecycleBin }); i >= 0 {
			bin := out.Groups[i]
			out.Groups = append(slices.Delete(out.Groups, i, i+1), bin)
		}
	}
	return out
}

func convertEntry(e *gokeepasslib.Entry, group *Group, path string) *Entry {
	ent := &Entry{
		Title:    e.GetTitle(),
		Username: e.GetContent("UserName"),
		Password: e.GetPassword(),
		URL:      e.GetContent("URL"),
		Notes:    e.GetContent("Notes"),
		Tags:     e.Tags,
		Path:     path,
		group:    group,
	}
	ent.TitleLower = strings.ToLower(ent.Title)

	parts := make([]string, 0, 6+len(e.Values))
	parts = append(parts, ent.Title, ent.Username, ent.URL, ent.Path, ent.Notes, ent.Tags)
	for i := range e.Values {
		k := e.Values[i].Key
		if _, skip := skipSearchKeys[k]; skip {
			continue
		}
		v := e.Values[i].Value.Content
		if v == "" {
			continue
		}
		ent.Extra = append(ent.Extra, Field{Key: k, Value: v})
		parts = append(parts, v)
	}
	ent.Haystack = strings.ToLower(strings.Join(parts, " "))
	return ent
}

func flatten(g *Group, out []*Entry) []*Entry {
	out = append(out, g.Entries...)
	for _, c := range g.Groups {
		out = flatten(c, out)
	}
	return out
}

// Locate returns the group chain from the database root to the folder that
// contains e. Last element is that folder. Nil if e is not in this tree.
func (d *DB) Locate(e *Entry) []*Group {
	if d == nil || d.Root == nil || e == nil || e.group == nil {
		return nil
	}
	depth := 0
	for g := e.group; g != nil; g = g.parent {
		depth++
	}
	chain := make([]*Group, depth)
	for g := e.group; g != nil; g = g.parent {
		depth--
		chain[depth] = g
	}
	if chain[0] != d.Root {
		return nil
	}
	return chain
}

// Clear drops secrets from memory as well as Go allows.
func (d *DB) Clear() {
	if d == nil {
		return
	}
	for _, e := range d.Entries {
		e.Password = ""
		e.Notes = ""
		e.Haystack = ""
		e.Extra = nil
		e.group = nil
	}
	d.Entries = nil
	if d.Root != nil {
		clearGroup(d.Root)
		d.Root = nil
	}
}

func clearGroup(g *Group) {
	for _, c := range g.Groups {
		clearGroup(c)
	}
	g.Groups = nil
	g.Entries = nil
	g.parent = nil
}
