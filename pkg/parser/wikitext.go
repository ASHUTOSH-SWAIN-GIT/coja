package parser

import (
	"regexp"
	"strings"
)

var (
	// {{template}} and {{template|arg1|arg2}} — including nested ones
	reDoubleBrace = regexp.MustCompile(`\{\{[^}]*\}\}`)

	// [[Link|Display text]] → Display text, [[Link]] → Link
	reLink = regexp.MustCompile(`\[\[([^]|]*\|)?([^]]*)\]\]`)

	// [http://example.com Display text] → Display text
	reExtLink = regexp.MustCompile(`\[https?://[^\s\]]+\s*([^\]]*)\]`)

	// <ref>...</ref> and <ref name="...">...</ref> and self-closing <ref ... />
	reRef = regexp.MustCompile(`(?s)<ref[^>]*>.*?</ref>|<ref[^/]*/\s*>`)

	// Any remaining HTML tags like <small>, <br/>, <blockquote>, etc.
	reHTMLTags = regexp.MustCompile(`<[^>]+>`)

	// == Heading == or === Subheading ===
	reHeading = regexp.MustCompile(`={2,6}\s*([^=]+?)\s*={2,6}`)

	// '''bold''' and ''italic''
	reBoldItalic = regexp.MustCompile(`'{2,3}`)

	// Category and File links [[Category:...]] [[File:...]]
	reCategoryFile = regexp.MustCompile(`\[\[(Category|File|Image):[^\]]*\]\]`)

	// Multiple whitespace → single space
	reMultiSpace = regexp.MustCompile(`\s+`)

	// Multiple newlines → single newline
	reMultiNewline = regexp.MustCompile(`\n{3,}`)
)

// StripWikitext removes MediaWiki markup and returns plain text.
func StripWikitext(raw string) string {
	s := raw

	// Remove Category/File links entirely
	s = reCategoryFile.ReplaceAllString(s, "")

	// Remove templates (run twice to catch shallow nesting)
	s = reDoubleBrace.ReplaceAllString(s, "")
	s = reDoubleBrace.ReplaceAllString(s, "")

	// Remove references
	s = reRef.ReplaceAllString(s, "")

	// Remove remaining HTML tags
	s = reHTMLTags.ReplaceAllString(s, "")

	// Convert [[Link|Display]] → Display, [[Link]] → Link
	s = reLink.ReplaceAllString(s, "$2")

	// Convert external links
	s = reExtLink.ReplaceAllString(s, "$1")

	// Convert headings: == Foo == → Foo
	s = reHeading.ReplaceAllString(s, "$1")

	// Remove bold/italic markers
	s = reBoldItalic.ReplaceAllString(s, "")

	// Remove common leftover markup
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&nbsp;", " ")

	// Clean up any HTML tags that came from entity decoding
	s = reHTMLTags.ReplaceAllString(s, "")

	// Normalize whitespace
	s = reMultiSpace.ReplaceAllString(s, " ")
	s = reMultiNewline.ReplaceAllString(s, "\n")
	s = strings.TrimSpace(s)

	return s
}