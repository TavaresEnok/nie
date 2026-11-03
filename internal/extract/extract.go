package extract

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

const (
	DefaultPassageChars = 1800
	MaxTextChars        = 8 << 20
)

type Passage struct {
	Ordinal     int
	Page        *int
	Text        string
	StartOffset int
	EndOffset   int
}

type Result struct {
	Title         string
	Text          string
	MIME          string
	Links         []string
	Passages      []Passage
	Parser        string
	ParserVersion string
	// LinkKind é o tipo de aresta para Links ("" = link HTML comum; ver feed.go).
	LinkKind string
	// FeedLinks são feeds RSS/Atom anunciados por <link rel="alternate"> no HTML.
	FeedLinks []string
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

	res := Result{MIME: "text/html", Parser: "nie-html", ParserVersion: "1.2.0"}
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
	res.FeedLinks = feedAutodiscovery(base, findElement(doc, "head"))
	res.Passages = MakePassages(res.Text, nil, DefaultPassageChars)
	return res, nil
}

// feedAutodiscovery coleta <link rel="alternate" type="application/rss+xml|atom+xml">.
func feedAutodiscovery(base *url.URL, head *html.Node) []string {
	if head == nil {
		return nil
	}
	var feeds []string
	for c := head.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || !strings.EqualFold(c.Data, "link") {
			continue
		}
		typ := strings.ToLower(attr(c, "type"))
		if !strings.Contains(strings.ToLower(attr(c, "rel")), "alternate") ||
			(typ != "application/rss+xml" && typ != "application/atom+xml") {
			continue
		}
		if u, ok := resolveLink(base, attr(c, "href")); ok {
			feeds = append(feeds, u)
		}
	}
	return feeds
}

var (
	urlRegex      = regexp.MustCompile(`https?://[a-zA-Z0-9\-\._~:/\?#\[\]@!$&'\(\)\*\+,;=%]+[a-zA-Z0-9/_]`)
	pdfURIAnnotRE = regexp.MustCompile(`/URI\s*\((https?://[^)\r\n]+)\)`)
)

// ExtractPDFAnnotationLinks extrai links de anotações interativas (/Annot /Link /URI (...)) embutidas na estrutura PDF.
func ExtractPDFAnnotationLinks(body []byte) []string {
	matches := pdfURIAnnotRE.FindAllSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		raw := strings.TrimSpace(string(m[1]))
		// Remove escapes básicos de string literal PDF: \( \) \\
		raw = strings.ReplaceAll(raw, `\(`, "(")
		raw = strings.ReplaceAll(raw, `\)`, ")")
		raw = strings.ReplaceAll(raw, `\\`, `\`)
		cleaned := strings.TrimRight(raw, ".,;:!?)'\"")
		if _, ok := seen[cleaned]; !ok && len(cleaned) > 8 {
			seen[cleaned] = struct{}{}
			out = append(out, cleaned)
		}
	}
	return out
}

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
	res := Result{MIME: "text/plain", Text: text, Parser: "nie-text", ParserVersion: "1.2.0"}
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
	annotLinks := ExtractPDFAnnotationLinks(body)
	reader, err := pdf.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return Result{}, fmt.Errorf("abrir PDF: %w", err)
	}
	res = Result{MIME: "application/pdf", Parser: "nie-pdf", ParserVersion: "1.2.0"}
	var all strings.Builder
	baseOffset := 0
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
		pagePassages := MakePassages(clean, &pn, DefaultPassageChars)
		for i := range pagePassages {
			pagePassages[i].Ordinal = len(res.Passages) + i
			pagePassages[i].StartOffset += baseOffset
			pagePassages[i].EndOffset += baseOffset
		}
		res.Passages = append(res.Passages, pagePassages...)
		all.WriteString(clean)
		all.WriteString("\n\n")
		baseOffset = all.Len()
	}
	res.Text = strings.TrimSpace(all.String())
	if len(res.Text) > MaxTextChars {
		res.Text = res.Text[:MaxTextChars]
	}
	if len(res.Passages) == 0 {
		return Result{}, fmt.Errorf("PDF sem texto extraível (OCR não habilitado)")
	}
	res.Title = firstLine(res.Text)
	seenLinks := make(map[string]struct{})
	for _, l := range ExtractURLLinks(res.Text) {
		seenLinks[l] = struct{}{}
		res.Links = append(res.Links, l)
	}
	for _, l := range annotLinks {
		if _, ok := seenLinks[l]; !ok {
			seenLinks[l] = struct{}{}
			res.Links = append(res.Links, l)
		}
	}
	return res, nil
}

func MakePassages(text string, page *int, max int) []Passage {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if max < 200 {
		max = DefaultPassageChars
	}
	var out []Passage
	for start := 0; start < len(text); {
		for start < len(text) {
			r, n := utf8.DecodeRuneInString(text[start:])
			if !unicode.IsSpace(r) {
				break
			}
			start += n
		}
		if start >= len(text) {
			break
		}
		end := min(start+max, len(text))
		for end < len(text) && !utf8.RuneStart(text[end]) {
			end--
		}
		if end < len(text) {
			if cut := strings.LastIndexAny(text[start:end], " \t\n"); cut > max/2 {
				end = start + cut
			}
		}
		chunk := strings.TrimSpace(text[start:end])
		if chunk != "" {
			out = append(out, Passage{Ordinal: len(out), Page: page, Text: chunk, StartOffset: start, EndOffset: start + len(chunk)})
		}
		start = end
	}
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
