package main

// sort a map's keys in descending order of its values.

import (
	"sort"
)

type sortedMap struct {
	m map[rune]int
	s []rune
}

func (sm *sortedMap) Len() int {
	return len(sm.m)
}

func (sm *sortedMap) Less(i, j int) bool {
	a, b := sm.m[sm.s[i]], sm.m[sm.s[j]]
	if a != b {
		// Order by decreasing value.
		return a > b
	} else {
		// Otherwise, alphabetical order.
		return sm.s[j] > sm.s[i]
	}
}

func (sm *sortedMap) Swap(i, j int) {
	sm.s[i], sm.s[j] = sm.s[j], sm.s[i]
}

func sortedKeys(m map[rune]int) []rune {
	sm := new(sortedMap)
	sm.m = m
	sm.s = make([]rune, len(m))
	i := 0
	for key, _ := range m {
		sm.s[i] = key
		i++
	}
	sort.Sort(sm)
	return sm.s
}
