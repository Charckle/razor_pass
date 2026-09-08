package kdbx

import (
	"fmt"
	"os"
	"sort"
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

// DB is an in-memory, read-only KeePass tree.
type DB struct {
	Root *Group
}

// Group is a KeePass group (folder).
type Group struct {
	Name         string
	IsRecycleBin bool
	Groups       []*Group
	Entries      []*Entry
}

// Entry is a KeePass password entry. Password is never included in Haystack.
type Entry struct {
	Title    string
	Username string
	Password string
	URL      string
	Notes    string
	Tags     string
	Path     string // group path, e.g. "Work / GitHub"
	Extra    []Field
	Haystack string
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

	root := convertGroup(&raw.Content.Root.Groups[0], "", recycle, true)
	return &DB{Root: root}, nil
}

func convertGroup(g *gokeepasslib.Group, parentPath string, recycle gokeepasslib.UUID, isRoot bool) *Group {
	out := &Group{
		Name:         g.Name,
		IsRecycleBin: !recycle.IsZero() && g.UUID.Compare(recycle),
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

	for i := range g.Groups {
		out.Groups = append(out.Groups, convertGroup(&g.Groups[i], path, recycle, false))
	}
	for i := range g.Entries {
		out.Entries = append(out.Entries, convertEntry(&g.Entries[i], path))
	}
	return out
}

func convertEntry(e *gokeepasslib.Entry, path string) *Entry {
	ent := &Entry{
		Title:    e.GetTitle(),
		Username: e.GetContent("UserName"),
		Password: e.GetPassword(),
		URL:      e.GetContent("URL"),
		Notes:    e.GetContent("Notes"),
		Tags:     e.Tags,
		Path:     path,
	}
	var extraParts []string
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
		extraParts = append(extraParts, v)
	}
	parts := []string{ent.Title, ent.Username, ent.URL, ent.Path, ent.Notes, ent.Tags}
	parts = append(parts, extraParts...)
	ent.Haystack = strings.Join(parts, " ")
	return ent
}

// Children returns groups then entries, both A–Z.
// At root (recycleLast), the Recycle Bin group is forced to the end.
func (g *Group) Children(recycleLast bool) (groups []*Group, entries []*Entry) {
	var recycle *Group
	for _, c := range g.Groups {
		if recycleLast && c.IsRecycleBin {
			recycle = c
			continue
		}
		groups = append(groups, c)
	}
	sort.Slice(groups, func(i, j int) bool {
		return strings.ToLower(groups[i].Name) < strings.ToLower(groups[j].Name)
	})
	if recycle != nil {
		groups = append(groups, recycle)
	}

	entries = append(entries, g.Entries...)
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Title) < strings.ToLower(entries[j].Title)
	})
	return groups, entries
}

// AllEntries walks the tree in display order.
func (d *DB) AllEntries() []*Entry {
	if d == nil || d.Root == nil {
		return nil
	}
	var out []*Entry
	var walk func(*Group, bool)
	walk = func(g *Group, atRoot bool) {
		groups, entries := g.Children(atRoot)
		out = append(out, entries...)
		for _, c := range groups {
			walk(c, false)
		}
	}
	walk(d.Root, true)
	return out
}

// Locate returns the group chain from the database root to the folder
// that contains e. Last element is that folder. Nil if not found.
func (d *DB) Locate(e *Entry) []*Group {
	if d == nil || d.Root == nil || e == nil {
		return nil
	}
	var found []*Group
	var walk func(*Group, []*Group) bool
	walk = func(g *Group, chain []*Group) bool {
		next := make([]*Group, len(chain)+1)
		copy(next, chain)
		next[len(chain)] = g
		for _, ent := range g.Entries {
			if ent == e {
				found = next
				return true
			}
		}
		for _, c := range g.Groups {
			if walk(c, next) {
				return true
			}
		}
		return false
	}
	if walk(d.Root, nil) {
		return found
	}
	return nil
}

// Clear drops secrets from memory as well as Go allows.
func (d *DB) Clear() {
	if d == nil || d.Root == nil {
		return
	}
	var walk func(*Group)
	walk = func(g *Group) {
		for _, e := range g.Entries {
			e.Password = ""
			e.Notes = ""
			e.Haystack = ""
			e.Extra = nil
		}
		for _, c := range g.Groups {
			walk(c)
		}
		g.Entries = nil
		g.Groups = nil
	}
	walk(d.Root)
	d.Root = nil
}
