package tokenizer

import "strings"

// Stem applies basic suffix stripping to reduce words to their root form.
// This is a simplified Porter stemmer — not perfect but good enough
// for search relevance.
func Stem(word string) string {
	// Step 1: plurals and past participles
	if strings.HasSuffix(word, "ies") && len(word) > 4 {
		word = word[:len(word)-3] + "i"
	} else if strings.HasSuffix(word, "sses") {
		word = word[:len(word)-2]
	} else if strings.HasSuffix(word, "ness") && len(word) > 5 {
		word = word[:len(word)-4]
	} else if strings.HasSuffix(word, "ment") && len(word) > 5 {
		word = word[:len(word)-4]
	} else if strings.HasSuffix(word, "ting") && len(word) > 5 {
		word = word[:len(word)-3]
	} else if strings.HasSuffix(word, "ing") && len(word) > 5 {
		word = word[:len(word)-3]
	} else if strings.HasSuffix(word, "tion") && len(word) > 5 {
		word = word[:len(word)-4]
	} else if strings.HasSuffix(word, "sion") && len(word) > 5 {
		word = word[:len(word)-4]
	} else if strings.HasSuffix(word, "ous") && len(word) > 5 {
		word = word[:len(word)-3]
	} else if strings.HasSuffix(word, "ive") && len(word) > 5 {
		word = word[:len(word)-3]
	} else if strings.HasSuffix(word, "ful") && len(word) > 5 {
		word = word[:len(word)-3]
	} else if strings.HasSuffix(word, "able") && len(word) > 5 {
		word = word[:len(word)-4]
	} else if strings.HasSuffix(word, "ible") && len(word) > 5 {
		word = word[:len(word)-4]
	} else if strings.HasSuffix(word, "ally") && len(word) > 5 {
		word = word[:len(word)-4]
	} else if strings.HasSuffix(word, "ly") && len(word) > 4 {
		word = word[:len(word)-2]
	} else if strings.HasSuffix(word, "ed") && len(word) > 4 {
		word = word[:len(word)-2]
	} else if strings.HasSuffix(word, "er") && len(word) > 4 {
		word = word[:len(word)-2]
	} else if strings.HasSuffix(word, "es") && len(word) > 3 {
		word = word[:len(word)-2]
	} else if strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") && len(word) > 3 {
		word = word[:len(word)-1]
	}

	return word
}