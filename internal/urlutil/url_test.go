package urlutil

import (
	"errors"
	"net/url"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"https://Example.COM:443/a/../b/?utm_source=x&b=2&a=1#frag", "https://example.com/b/?a=1&b=2"},
		{"http://example.com", "http://example.com/"},
		{"https://example.com:8443/a", "https://example.com:8443/a"},
		{"https://example.com/path?b=2&a=1", "https://example.com/path?a=1&b=2"},
		{"https://example.com/%7Euser/", "https://example.com/~user/"},
	}
	for _, tc := range tests {
		got, err := Normalize(tc.in)
		if err != nil {
			t.Errorf("Normalize(%q) erro: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Normalize(%q) = %q, quer %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeRejects(t *testing.T) {
	tests := []struct {
		in  string
		err error
	}{
		{"", ErrInvalidURL},
		{"ftp://example.com/file", ErrUnsupported},
		{"file:///etc/passwd", ErrUnsupported},
		{"https://user:pass@example.com/", ErrInvalidURL},
		{"https://localhost/", ErrPrivateHost},
		{"https://foo.local/", ErrPrivateHost},
		{"https://foo.internal/", ErrPrivateHost},
	}
	for _, tc := range tests {
		_, err := Normalize(tc.in)
		if !errors.Is(err, tc.err) {
			t.Errorf("Normalize(%q) erro = %v, quer %v", tc.in, err, tc.err)
		}
	}
}

func TestResolve(t *testing.T) {
	base, err := url.Parse("https://example.org/docs/page")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := Resolve(base, "../guide?b=2&a=1")
	if !ok {
		t.Fatal("Resolve deveria aceitar URL relativa")
	}
	if want := "https://example.org/guide?a=1&b=2"; got != want {
		t.Fatalf("Resolve = %q, quer %q", got, want)
	}
	for _, href := range []string{"#section", "mailto:x@example.org", "javascript:alert(1)", "data:text/plain,x", ""} {
		if _, ok := Resolve(base, href); ok {
			t.Errorf("Resolve(%q) deveria ser rejeitado", href)
		}
	}
}

func TestValidateCrawlURL(t *testing.T) {
	blocked := []string{
		"http://127.0.0.1/",
		"http://10.0.0.1/",
		"http://192.168.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://[::1]/",
		"http://[fd00:ec2::254]/",
		"http://localhost/",
	}
	for _, raw := range blocked {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateCrawlURL(u); !errors.Is(err, ErrPrivateHost) {
			t.Errorf("ValidateCrawlURL(%q) = %v, quer ErrPrivateHost", raw, err)
		}
	}

	u, _ := url.Parse("https://8.8.8.8/dns")
	if err := ValidateCrawlURL(u); err != nil {
		t.Errorf("IP público deveria ser aceito: %v", err)
	}

	u, _ = url.Parse("https://example.com:8443/")
	if err := ValidateCrawlURL(u); !errors.Is(err, ErrUnsupported) {
		t.Errorf("porta não suportada deveria falhar: %v", err)
	}
}

func TestHashStable(t *testing.T) {
	if Hash("https://example.com/") != Hash("https://example.com/") {
		t.Fatal("hash deveria ser estável")
	}
	if Hash("https://example.com/a") == Hash("https://example.com/b") {
		t.Fatal("URLs diferentes não deveriam colidir")
	}
}
