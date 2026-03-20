package repositories

import (
	"regexp"
	"strings"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

func formatPrefixTSQuery(query string) string {
	words := strings.Fields(query)
	var validWords []string
	for _, w := range words {
		cleanW := nonAlphanumericRegex.ReplaceAllString(w, "")
		if cleanW != "" {
			validWords = append(validWords, cleanW+":*")
		}
	}
	if len(validWords) > 0 {
		return strings.Join(validWords, " & ")
	}
	return ""
}
