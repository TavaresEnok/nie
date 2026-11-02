package extract

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html/charset"
)

// Tipos de aresta de descoberta produzidos por documentos XML.
const (
	LinkKindSitemap = "sitemap"
	LinkKindFeed    = "feed"
)

// MaxFeedLinks limita quantas URLs um único sitemap/feed pode enfileirar.
const MaxFeedLinks = 5000

// IsFeedCandidate indica se o recurso deve ser tratado como sitemap ou feed
// RSS/Atom, pelo Content-Type ou pela extensão do caminho.
func IsFeedCandidate(contentType, path string) bool {
	ct := strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	switch ct {
	case "application/rss+xml", "application/atom+xml", "application/xml", "text/xml":
		return true
	case "text/html", "application/xhtml+xml", "application/pdf":
		return false
	}
	return strings.HasSuffix(strings.ToLower(path), ".xml")
}

// Feed extrai URLs de sitemaps (urlset/sitemapindex), RSS 2.0 e Atom.
// O texto resultante lista títulos e URLs, para que o documento seja citável.
func Feed(base *url.URL, body []byte) (Result, error) {
	dec := xml.NewDecoder(bytes.NewReader(body))
	dec.CharsetReader = charset.NewReaderLabel
	dec.Strict = false

	res := Result{MIME: "application/xml"}
	seen := make(map[string]struct{})
	var text strings.Builder
	var stack []string
	var root string

	add := func(raw string) {
		if len(res.Links) >= MaxFeedLinks {
			return
		}
		u, ok := resolveLink(base, strings.TrimSpace(raw))
		if !ok {
			return
		}
		if _, dup := seen[u]; dup {
			return
		}
		seen[u] = struct{}{}
		res.Links = append(res.Links, u)
		text.WriteString(u)
		text.WriteString("\n")
	}

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Result{}, fmt.Errorf("parse XML: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := strings.ToLower(t.Name.Local)
			if root == "" {
				root = name
			}
			stack = append(stack, name)
			// Atom: <link href="..."/> (ignora rel="self" e afins).
			if name == "link" && root == "feed" {
				rel, href := "", ""
				for _, a := range t.Attr {
					switch strings.ToLower(a.Name.Local) {
					case "rel":
						rel = strings.ToLower(a.Value)
					case "href":
						href = a.Value
					}
				}
				if href != "" && (rel == "" || rel == "alternate") && parentIs(stack, "entry") {
					add(href)
				}
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			cur := stack[len(stack)-1]
			value := strings.TrimSpace(string(t))
			if value == "" {
				continue
			}
			switch {
			case cur == "loc" && (root == "urlset" || root == "sitemapindex"):
				add(value)
			case cur == "link" && root == "rss" && parentIs(stack, "item"):
				add(value)
			case cur == "title" && (parentIs(stack, "item") || parentIs(stack, "entry")):
				text.WriteString(value)
				text.WriteString("\n")
			}
		}
	}

	switch root {
	case "urlset", "sitemapindex":
		res.LinkKind = LinkKindSitemap
	case "rss", "feed":
		res.LinkKind = LinkKindFeed
	default:
		return Result{}, fmt.Errorf("XML não é sitemap nem feed (raiz %q)", root)
	}
	res.Text = NormalizeText(text.String())
	if len(res.Text) > MaxTextChars {
		res.Text = res.Text[:MaxTextChars]
	}
	res.Title = root
	res.Passages = MakePassages(res.Text, nil, DefaultPassageChars)
	return res, nil
}

// parentIs informa se o elemento atual está diretamente dentro de tag.
func parentIs(stack []string, tag string) bool {
	return len(stack) >= 2 && stack[len(stack)-2] == tag
}
