package bytebase

import (
	"math/rand"
	"sort"
)

func FindString(strings []string, search string) int {
	sort.Strings(strings)
	i := sort.SearchStrings(strings, search)
	if i == len(strings) {
		return -1
	}
	return i
}

var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
