package util

import (
	"iter"
	"strings"
)

func SplitStringAny(s string, chars string) []string {
	// TODO: Handle non-ASCII
	var res []string
	i := 0
	for {
		j := strings.IndexAny(s[i:], chars)
		if j == -1 {
			res = append(res, s[i:])
			break
		}
		res = append(res, s[i:i+j])
		i += j + 1
	}
	return res
}

func SplitStringAnySeq(s string, chars string) iter.Seq[string] {
	// TODO: Handle non-ASCII
	return func(yield func(string) bool) {
		i := 0
		for {
			j := strings.IndexAny(s[i:], chars)
			if j == -1 {
				if !yield(s[i:]) {
					return
				}
				break
			}
			if !yield(s[i : i+j]) {
				return
			}
			i += j + 1
		}
	}
}

func SplitStringAfterAny(s string, chars string) []string {
	// TODO: Handle non-ASCII
	var res []string
	i := 0
	for {
		j := strings.IndexAny(s[i:], chars)
		if j == -1 {
			res = append(res, s[i:])
			break
		}
		res = append(res, s[i:i+j+1])
		i += j + 1
	}
	return res
}

func SplitStringAfterAnySeq(s string, chars string) iter.Seq[string] {
	// TODO: Handle non-ASCII
	return func(yield func(string) bool) {
		i := 0
		for {
			j := strings.IndexAny(s[i:], chars)
			if j == -1 {
				if !yield(s[i:]) {
					return
				}
				break
			}
			if !yield(s[i : i+j+1]) {
				return
			}
			i += j + 1
		}
	}
}
