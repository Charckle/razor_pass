package fuzzy

import "strings"

// Match reports whether query is a fuzzy (subsequence) match of text.
// Multiple space-separated terms are AND-ed. Empty query matches everything.
// Score is higher for tighter, earlier matches.
func Match(query, text string) (ok bool, score int) {
	text = strings.ToLower(text)
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return true, 0
	}
	total := 0
	for _, term := range terms {
		matched, s := subsequence(term, text)
		if !matched {
			return false, 0
		}
		total += s
	}
	return true, total
}

func subsequence(query, text string) (bool, int) {
	if query == "" {
		return true, 0
	}
	q := []rune(query)
	t := []rune(text)
	ti := 0
	score := 0
	consecutive := 0
	first := -1
	for _, qr := range q {
		found := false
		for ti < len(t) {
			if t[ti] == qr {
				if first < 0 {
					first = ti
				}
				consecutive++
				score += 10 + consecutive*4
				ti++
				found = true
				break
			}
			consecutive = 0
			ti++
		}
		if !found {
			return false, 0
		}
	}
	if first >= 0 {
		score -= first
	}
	return true, score
}
