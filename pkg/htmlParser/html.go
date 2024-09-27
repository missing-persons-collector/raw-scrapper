package htmlParser

import (
	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"strings"
)

func Parse(body string) (*html.Node, error) {
	doc, err := html.Parse(strings.NewReader(body))

	if err != nil {
		return nil, err
	}

	return doc, nil
}

func Query(pageHtml *html.Node, query string) ([]*html.Node, error) {
	sel, err := cascadia.Parse(query)
	if err != nil {
		return nil, err
	}

	node := cascadia.QueryAll(pageHtml, sel)

	return node, nil
}

func Find(pageHtml *html.Node, query string) (*html.Node, error) {
	sel, err := cascadia.Parse(query)
	if err != nil {
		return nil, err
	}

	node := cascadia.Query(pageHtml, sel)

	return node, nil
}

func Attr(attr string, attributes []html.Attribute) string {
	for _, a := range attributes {
		if a.Key == attr {
			return a.Val
		}
	}

	return ""
}
