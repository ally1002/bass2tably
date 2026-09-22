package tui

import "unicode"

func lowerRunes(s string) []rune {
	out := []rune(s)
	for i, r := range out {
		out[i] = unicode.ToLower(r)
	}
	return out
}

func fuzzy(pattern, s string) (score int, pos []int, ok bool) {
	if pattern == "" {
		return 0, nil, true
	}
	score, pos, ok = fuzzyScore(pattern, s)
	if !ok {
		return 0, nil, false
	}
	return score - len([]rune(s))/8, pos, true
}

func fuzzyScore(pattern, s string) (score int, pos []int, ok bool) {
	p, t := lowerRunes(pattern), lowerRunes(s)
	if len(p) == 0 {
		return 0, nil, true
	}
	for start, r := range t {
		if r != p[0] {
			continue
		}
		sc, ps, matched := greedyFrom(p, t, start)
		if matched && (!ok || sc > score) {
			score, pos, ok = sc, ps, true
		}
	}
	return score, pos, ok
}

func greedyFrom(p, t []rune, start int) (score int, pos []int, ok bool) {
	pi, prev := 0, -2
	for ti := start; ti < len(t); ti++ {
		r := t[ti]
		if pi == len(p) || r != p[pi] {
			continue
		}
		switch {
		case ti == prev+1:
			score += 3
		case ti == 0 || isBoundary(t[ti-1]):
			score += 2
		default:
			score++
		}
		pos = append(pos, ti)
		prev, pi = ti, pi+1
	}
	return score, pos, pi == len(p)
}

func isBoundary(r rune) bool {
	return r == '/' || r == '_' || r == '-' || r == '.' || r == ' '
}
