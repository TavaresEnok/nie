package extract

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

const (
	DefaultPassageChars = 1800
	MaxTextChars        = 8 << 20
)

type Passage struct {
	Ordinal int
	Page    *int
	Text    string
}

type Result struct {
	Title    string
	Text     string
	MIME     string
	Links    []string
	Passages []Passage
}

var spaceRE = regexp.MustCompile(`[ \t\f\v\x{00a0}]+`)

func HTML(base *url.URL, contentType string, body []byte) (Result, error) {
	reader, err := charset.NewReader(bytes.NewReader(body), contentType)
	if err != nil {
		reader = bytes.NewReader(body)
	}
	doc, err := html.Parse(reader)
	if err != nil {
		return Result{}, fmt.Errorf("parse HTML: %w", err)
	}

	res := Result{MIME: "text/html"}
	res.Title = strings.TrimSpace(collapse(nodeText(findElement(doc, "title"))))

	var text strings.Builder
	links := make(map[string]struct{})

	var walk func(*html.Node, bool)
	walk = func(n *html.Node, skip bool) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			switch tag {
			case "script", "style", "noscript", "template", "svg", "canvas", "iframe", "object", "embed":
				skip = true
			case "a":
				if !skip {
					if href := attr(n, "href"); href != "" {
						if u, ok := resolveLink(base, href); ok {
							links[u] = struct{}{}
						}
					}
				}
			}
			if isBlock(tag) {
				text.WriteString("\n")
			}
		}
		if n.Type == html.TextNode && !skip {
			text.WriteString(n.Data)
			text.WriteString(" ")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, skip)
		}
		if n.Type == html.ElementNode && isBlock(strings.ToLower(n.Data)) {
			text.WriteString("\n")
		}
	}
	if body := findElement(doc, "body"); body != nil {
		walk(body, false)
	} else {
		walk(doc, false)
	}

	res.Text = NormalizeText(text.String())
	if len(res.Text) > MaxTextChars {
		res.Text = res.Text[:MaxTextChars]
	}
	if res.Title == "" {
		if first := firstLine(res.Text); first != "" && len(first) < 200 {
			res.Title = first
		}
	}
	for link := range links {
		if base != nil && link == base.String() {
			continue
		}
		res.Links = append(res.Links, link)
	}
	res.Passages = MakePassages(res.Text, nil, DefaultPassageChars)
	return res, nil
}

var urlRegex = regexp.MustCompile(`https?://[a-zA-Z0-9\-\._~:/\?#\[\]@!$&'\(\)\*\+,;=%]+[a-zA-Z0-9/_]`)

// ExtractURLLinks extrai URLs e citações de hiperlinks presentes no texto puro de PDFs ou documentos técnicos.
func ExtractURLLinks(text string) []string {
	matches := urlRegex.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var unique []string
	for _, m := range matches {
		cleaned := strings.TrimRight(m, ".,;:!?)'\"")
		if _, exists := seen[cleaned]; !exists && len(cleaned) > 8 {
			seen[cleaned] = struct{}{}
			unique = append(unique, cleaned)
		}
	}
	return unique
}

func Text(body []byte) (Result, error) {
	text := NormalizeText(string(body))
	if len(text) > MaxTextChars {
		text = text[:MaxTextChars]
	}
	res := Result{MIME: "text/plain", Text: text}
	if first := firstLine(text); len(first) < 200 {
		res.Title = first
	}
	res.Passages = MakePassages(text, nil, DefaultPassageChars)
	res.Links = ExtractURLLinks(text)
	return res, nil
}

func PDF(body []byte) (res Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("PDF inválido ou não suportado: %v", r)
			res = Result{}
		}
	}()
	reader, err := pdf.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return Result{}, fmt.Errorf("abrir PDF: %w", err)
	}
	res = Result{MIME: "application/pdf"}
	var all strings.Builder
	for page := 1; page <= reader.NumPage(); page++ {
		p := reader.Page(page)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		clean := NormalizeText(text)
		if clean == "" {
			continue
		}
		pn := page
		all.WriteString(clean)
		all.WriteString("\n\n")
		res.Passages = append(res.Passages, MakePassages(clean, &pn, DefaultPassageChars)...)
	}
	res.Text = strings.TrimSpace(all.String())
	if len(res.Text) > MaxTextChars {
		res.Text = res.Text[:MaxTextChars]
	}
	if len(res.Passages) == 0 {
		return Result{}, fmt.Errorf("PDF sem texto extraível (OCR não habilitado)")
	}
	res.Title = firstLine(res.Text)
	res.Links = ExtractURLLinks(res.Text)
	return res, nil
}

func MakePassages(text string, page *int, max int) []Passage {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if max < 200 {
		max = DefaultPassageChars
	}
	var out []Passage
	var b strings.Builder
	flush := func() {
		chunk := strings.TrimSpace(b.String())
		if chunk != "" {
			out = append(out, Passage{Ordinal: len(out), Page: page, Text: chunk})
		}
		b.Reset()
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if b.Len() > 0 && b.Len()+len(line)+1 > max {
			flush()
		}
		for len(line) > max {
			cut := max
			if idx := strings.LastIndexAny(line[:max], " \t"); idx > max/2 {
				cut = idx
			}
			if b.Len() > 0 {
				flush()
			}
			out = append(out, Passage{Ordinal: len(out), Page: page, Text: strings.TrimSpace(line[:cut])})
			line = strings.TrimSpace(line[cut:])
		}
		if line == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	flush()
	return out
}

func NormalizeText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	blank := 0
	for _, line := range lines {
		line = strings.TrimSpace(spaceRE.ReplaceAllString(line, " "))
		if line == "" {
			blank++
			if blank > 1 {
				continue
			}
		} else {
			blank = 0
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func isBlock(tag string) bool {
	switch tag {
	case "address", "article", "aside", "blockquote", "br", "dd", "div", "dl", "dt",
		"fieldset", "figcaption", "figure", "footer", "form", "h1", "h2", "h3", "h4",
		"h5", "h6", "header", "hr", "li", "main", "nav", "ol", "p", "pre", "section",
		"table", "tbody", "td", "tfoot", "th", "thead", "tr", "ul":
		return true
	}
	return false
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func findElement(n *html.Node, tag string) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode && strings.EqualFold(n.Data, tag) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func nodeText(n *html.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func resolveLink(base *url.URL, href string) (string, bool) {
	if base == nil {
		return "", false
	}
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") {
		return "", false
	}
	ref, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	switch strings.ToLower(ref.Scheme) {
	case "mailto", "tel", "javascript", "data", "ftp", "file":
		return "", false
	}
	u := base.ResolveReference(ref)
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}
	if u.Host == "" || u.User != nil {
		return "", false
	}
	u.Fragment = ""
	return u.String(), true
}
