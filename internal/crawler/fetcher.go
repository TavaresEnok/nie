package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"nie/internal/netguard"
	"nie/internal/urlutil"
)

var (
	ErrBlocked         = errors.New("destino bloqueado pela política de egress")
	ErrTooLarge        = errors.New("recurso excede o limite de tamanho")
	ErrUnsupportedType = errors.New("tipo de conteúdo não suportado")
	// ErrRobotsDisallowed: um redirecionamento levou a uma URL vetada pelo robots.txt.
	ErrRobotsDisallowed = errors.New("redirecionamento bloqueado por robots.txt")
)

type HTTPError struct {
	StatusCode int
	URL        string
	RetryAfter time.Duration
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d em %s", e.StatusCode, e.URL)
}

type Response struct {
	URL         string
	StatusCode  int
	ContentType string
	Header      http.Header
	Body        []byte
}

type Fetcher struct {
	client    *http.Client
	userAgent string
	maxHTML   int64
	maxPDF    int64
	maxText   int64

	// redirectCheck é consultado a cada redirecionamento (exceto /robots.txt)
	// para que o destino final também respeite o robots.txt do seu host.
	redirectCheck func(ctx context.Context, u *url.URL) error
}

// SetRedirectCheck registra a verificação aplicada a cada redirecionamento.
func (f *Fetcher) SetRedirectCheck(check func(ctx context.Context, u *url.URL) error) {
	f.redirectCheck = check
}

func (f *Fetcher) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return errors.New("excesso de redirecionamentos")
	}
	if err := urlutil.ValidateCrawlURL(req.URL); err != nil {
		return err
	}
	// O próprio robots.txt pode redirecionar (http→https); não consultar o
	// robots para ele, o que causaria recursão.
	if f.redirectCheck != nil && req.URL.Path != "/robots.txt" {
		return f.redirectCheck(req.Context(), req.URL)
	}
	return nil
}

func NewFetcher(userAgent string, timeout time.Duration, maxHTML, maxPDF, maxText int64) *Fetcher {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:                  nil, // não permitir proxy de ambiente contornar a validação
		ForceAttemptHTTP2:      true,
		MaxIdleConns:           32,
		MaxIdleConnsPerHost:    4,
		IdleConnTimeout:        60 * time.Second,
		TLSHandshakeTimeout:    10 * time.Second,
		ResponseHeaderTimeout:  timeout,
		MaxResponseHeaderBytes: 1 << 20,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, fmt.Errorf("%w: endereço inválido", ErrBlocked)
			}
			if port != "80" && port != "443" {
				return nil, fmt.Errorf("%w: porta %s", ErrBlocked, port)
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("resolver %s: %w", host, err)
			}
			var lastErr error
			for _, ip := range ips {
				if !IsPublicIP(ip.IP) {
					continue
				}
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, fmt.Errorf("%w: %s resolve apenas para IPs privados/reservados", ErrBlocked, host)
		},
	}
	f := &Fetcher{
		userAgent: userAgent,
		maxHTML:   maxHTML,
		maxPDF:    maxPDF,
		maxText:   maxText,
	}
	f.client = &http.Client{
		Transport:     transport,
		Timeout:       timeout,
		CheckRedirect: f.checkRedirect,
	}
	return f
}

func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (*Response, error) {
	u, _, err := urlutil.ParseNormalized(rawURL)
	if err != nil {
		return nil, err
	}
	if err := urlutil.ValidateCrawlURL(u); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, f.client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/pdf,text/plain,text/markdown;q=0.9,application/xml,text/xml,application/rss+xml,application/atom+xml;q=0.8,*/*;q=0.1")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8")
	req.Header.Set("From", "nie@localhost")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	limit := f.limitFor(contentType)
	if resp.ContentLength > 0 && resp.ContentLength > limit {
		return nil, fmt.Errorf("%w: content-length %d > %d", ErrTooLarge, resp.ContentLength, limit)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%w: corpo > %d bytes", ErrTooLarge, limit)
	}

	mediaType, _, _ := mime.ParseMediaType(contentType)
	if mediaType == "" || mediaType == "application/octet-stream" || mediaType == "binary/octet-stream" {
		detected := http.DetectContentType(body)
		mediaType, _, _ = mime.ParseMediaType(detected)
		if strings.HasPrefix(strings.TrimSpace(string(body)), "%PDF-") {
			mediaType = "application/pdf"
		}
		contentType = detected
	}
	if !supportedType(mediaType) {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, mediaType)
	}
	if mediaType == "application/pdf" && int64(len(body)) > f.maxPDF {
		return nil, ErrTooLarge
	}
	if isHTMLType(mediaType) && int64(len(body)) > f.maxHTML {
		return nil, ErrTooLarge
	}
	if isTextType(mediaType) && int64(len(body)) > f.maxText {
		return nil, ErrTooLarge
	}

	return &Response{
		URL:         resp.Request.URL.String(),
		StatusCode:  resp.StatusCode,
		ContentType: contentType,
		Header:      resp.Header,
		Body:        body,
	}, nil
}

func (f *Fetcher) limitFor(contentType string) int64 {
	mediaType, _, _ := mime.ParseMediaType(contentType)
	switch {
	case isHTMLType(mediaType):
		return f.maxHTML
	case mediaType == "application/pdf":
		return f.maxPDF
	case isTextType(mediaType):
		return f.maxText
	default:
		// Até o sniff, aceita o maior limite. O tipo final é validado depois.
		max := f.maxPDF
		if f.maxHTML > max {
			max = f.maxHTML
		}
		if f.maxText > max {
			max = f.maxText
		}
		return max
	}
}

func isHTMLType(t string) bool {
	return t == "text/html" || t == "application/xhtml+xml"
}

func isTextType(t string) bool {
	return t == "text/plain" || t == "text/markdown" || t == "text/x-markdown" || isFeedType(t)
}

// isFeedType cobre sitemaps e feeds RSS/Atom (descoberta de URLs).
func isFeedType(t string) bool {
	return t == "application/xml" || t == "text/xml" || t == "application/rss+xml" || t == "application/atom+xml"
}

func supportedType(t string) bool {
	return isHTMLType(t) || isTextType(t) || t == "application/pdf"
}

// IsPublicIP rejeita loopback, redes privadas, link-local, CGNAT, multicast e reservados.
func IsPublicIP(ip net.IP) bool {
	return netguard.IsPublicIP(ip)
}

// CheckRedirectTarget é exposto para testes do filtro de egress.
func CheckRedirectTarget(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	return urlutil.ValidateCrawlURL(u)
}
