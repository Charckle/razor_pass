package fuzzy

import (
	"strings"
	"unicode/utf8"
)

// Query is a compiled query: space-separated terms that are AND-ed together.
// Compile once per query, then match it against every candidate.
type Query struct {
	terms []string
}

// Compile lowercases query and splits it into terms.
func Compile(query string) Query {
	return Query{terms: strings.Fields(strings.ToLower(query))}
}

// Match reports whether every term in q is a subsequence of text, which must
// already be lowercased. An empty query matches everything with score 0.
// Score is higher for tighter, earlier matches.
func (q Query) Match(text string) (ok bool, score int) {
	total := 0
	for _, term := range q.terms {
		matched, s := subsequence(term, text)
		if !matched {
			return false, 0
		}
		total += s
	}
	return true, total
}

// subsequence scans text in place: ranging over query yields runes without
// allocating, and IndexRune walks the remaining bytes of text directly.
func subsequence(query, text string) (bool, int) {
	score, consecutive, first, pos := 0, 0, -1, 0
	for _, qr := range query {
		i := strings.IndexRune(text[pos:], qr)
		if i < 0 {
			return false, 0
		}
		if i > 0 {
			consecutive = 0
		}
		consecutive++
		score += 10 + consecutive*4
		if first < 0 {
			first = pos + i
		}
		pos += i + utf8.RuneLen(qr)
	}
	if first > 0 {
		score -= first
	}
	return true, score
}
