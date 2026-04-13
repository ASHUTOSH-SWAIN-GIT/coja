package parser

import (
	"compress/bzip2"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// document represents a single wikipedia document
type Document struct {
	ID    int
	Title string
	Text  string
}

// xmlPage maps to the <page> element in the dump
type xmlPage struct {
	Title    string      `xml:"title"`
	NS       int         `xml:"ns"`
	ID       int         `xml:"id"`
	Redirect *struct{}   `xml:"redirect"`
	Revision xmlRevision `xml:"revision"`
}

// xmlRevision maps to the <revision> element
type xmlRevision struct {
	Text xmlText `xml:"text"`
}

// xmlText maps to <text> which has attribute
type xmlText struct {
	Content string `xml:",chardata"`
}

/*
parse streams through the wikipedia dump and sends documents on the channel
it skips redirects and non article page
*/

func Parse(path string, out chan<- Document) error {
	defer close(out)

	if path == "-" {
		return ParseReader(os.Stdin, out)
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var reader io.Reader = f
	if strings.EqualFold(filepath.Ext(path), ".bz2") {
		reader = bzip2.NewReader(f)
	}

	return ParseReader(reader, out)
}

// ParseReader streams wikipedia XML from any reader and sends documents on out.
func ParseReader(r io.Reader, out chan<- Document) error {
	decoder := xml.NewDecoder(r)

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("decode token: %w", err)
		}

		startElem, ok := token.(xml.StartElement)
		if !ok || startElem.Name.Local != "page" {
			continue
		}

		var page xmlPage
		if err := decoder.DecodeElement(&page, &startElem); err != nil {
			continue
		}

		if page.NS != 0 {
			continue
		}

		if page.Redirect != nil {
			continue
		}

		trimmed := strings.TrimSpace(page.Revision.Text.Content)
		if strings.HasPrefix(strings.ToUpper(trimmed), "#REDIRECT") {
			continue
		}

		out <- Document{
			ID:    page.ID,
			Title: page.Title,
			Text:  page.Revision.Text.Content,
		}
	}
}
