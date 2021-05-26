package bytebase

import "sort"

func FindString(strings []string, search string) string {
	sort.Strings(strings)
	i := sort.SearchStrings(strings, search)
	if i == len(strings) {
		return ""
	}
	return strings[i]
}
