package tokenizer

import (
	"strings"
	"unicode"
)

// stop words common words that add noise to search
var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true,
	"but": true, "in": true, "on": true, "at": true, "to": true,
	"for": true, "of": true, "with": true, "by": true, "from": true,
	"is": true, "it": true, "as": true, "was": true, "are": true,
	"be": true, "been": true, "has": true, "had": true, "have": true,
	"do": true, "does": true, "did": true, "will": true, "would": true,
	"could": true, "should": true, "may": true, "might": true,
	"that": true, "this": true, "these": true, "those": true,
	"not": true, "no": true, "nor": true, "so": true, "if": true,
	"then": true, "than": true, "too": true, "very": true,
	"can": true, "just": true, "also": true, "into": true,
	"about": true, "up": true, "out": true, "some": true,
	"its": true, "his": true, "her": true, "he": true, "she": true,
	"they": true, "we": true, "you": true, "i": true, "my": true,
	"your": true, "their": true, "our": true, "me": true, "him": true,
	"who": true, "which": true, "what": true, "where": true,
	"when": true, "how": true, "all": true, "each": true,
	"were": true, "there": true, "such": true,
	"more": true, "other": true, "only": true, "over": true,
	"after": true, "between": true, "through": true, "during": true,
	"before": true, "under": true, "above": true, "both": true,
	"same": true, "being": true, "most": true, "while": true,
}

// token represents a single token with its position in the document
type Token struct {
	Term     string
	Position int
}

/*
Token takes raw text and returns a slice of tokens
it lowercases, splits, on non alphanumeric chars,
removes stop words and stems each term
*/
func Tokenize(text string) []Token {
	var tokens []Token
	pos := 0

	//split text into words by non alphanumeric boundries
	words := splitWords(text)

	for _, word := range words {
		//lowercse
		word = strings.ToLower(word)

		//skip short words
		if len(word) < 2 {
			continue
		}

		//skip stop words
		if stopWords[word] {
			continue
		}

		//skip if its all digits
		if isAllDigits(word) {
			continue
		}

		//stem the word
		stemmed := Stem(word)
		if len(stemmed) < 2 {
			continue
		}

		tokens = append(tokens, Token{
			Term:     stemmed,
			Position: pos,
		})
		pos++
	}
	return tokens
}

// splitWords splits text on any non-alphanumeric character
func splitWords(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
