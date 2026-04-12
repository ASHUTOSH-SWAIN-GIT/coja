package parser

import (
	"compress/bzip2"
	"encoding/xml"
	"os"
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
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	defer f.Close()

	//bzip2 compressor reads from the file
	bzReader := bzip2.NewReader(f)
	decoder := xml.NewDecoder(bzReader)

	for {
		//read the new xml token
		token, err := decoder.Token()
		if err != nil {
			break
		}

		//we only care about <page> start elements
		startElem, ok := token.(xml.StartElement)
		if !ok || startElem.Name.Local != "page" {
			continue
		}

		//decode the full <page> into our struct
		var page xmlPage
		if err := decoder.DecodeElement(&page, &startElem); err != nil {
			continue
		}

		//skip non articale name spaces
		if page.NS != 0 {
			continue
		}

		//skip redirects
		if page.Redirect != nil {
			continue
		}

		// Skip text-based redirects
		if strings.HasPrefix(strings.TrimSpace(page.Revision.Text.Content), "#REDIRECT") {
			continue
		}

		out <- Document{
			ID:    page.ID,
			Title: page.Title,
			Text:  page.Revision.Text.Content,
		}
	}
	close(out)
	return nil
}
