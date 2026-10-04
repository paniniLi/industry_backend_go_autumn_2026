package main

import (
	"strings"
	"unicode/utf8"
)

func rotateRunes(s string, shift int) string {
	var runesCount = utf8.RuneCountInString(s)
	if runesCount == 0 {
		return s
	}

	var runesArray = []rune(s)
	shift = shift % runesCount
	if shift == 0 {
		return string(runesArray)
	}

	var resultStringBuilder strings.Builder
	for i := range runesCount {
		var index = (runesCount + (i+shift)%runesCount) % runesCount
		resultStringBuilder.WriteRune(runesArray[index])
	}

	return resultStringBuilder.String()
}
