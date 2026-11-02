package crawler

import (
	"net/url"
	"testing"
	"time"

	"github.com/temoto/robotstxt"
)

func TestRobotsPolicy(t *testing.T) {
	data, err := robotstxt.FromBytes([]byte("User-agent: *\nDisallow: /private\nAllow: /private/public\n"))
	if err != nil {
		t.Fatal(err)
	}
	r := &Robots{userAgent: "NIEBot/0.1"}
	entry := robotsEntry{rules: data, expiresAt: time.Now().Add(time.Hour)}

	allowed := []string{"https://example.org/", "https://example.org/docs", "https://example.org/private/public/x"}
	for _, raw := range allowed {
		u, _ := url.Parse(raw)
		if !r.test(entry, u) {
			t.Errorf("%s deveria ser permitida", raw)
		}
	}
	blocked := []string{"https://example.org/private", "https://example.org/private/x"}
	for _, raw := range blocked {
		u, _ := url.Parse(raw)
		if r.test(entry, u) {
			t.Errorf("%s deveria ser bloqueada", raw)
		}
	}

	if !r.test(robotsEntry{allowAll: true}, mustURL(t, "https://example.org/private")) {
		t.Error("allowAll deveria permitir")
	}
	if r.test(robotsEntry{denyAll: true}, mustURL(t, "https://example.org/")) {
		t.Error("denyAll deveria bloquear")
	}
}

func TestRobotToken(t *testing.T) {
	tests := map[string]string{
		"NIEBot/0.1 (+https://localhost)": "NIEBot",
		"NIEBot":                          "NIEBot",
		"":                                "",
	}
	for in, want := range tests {
		if got := robotToken(in); got != want {
			t.Errorf("robotToken(%q) = %q, quer %q", in, got, want)
		}
	}
}

func TestRobotsStatusPolicy(t *testing.T) {
	body := []byte("User-agent: *\nDisallow: /privado\n")
	tests := []struct {
		status    int
		wantAllow bool
		wantDeny  bool
	}{
		{200, false, false},
		{404, true, false},
		{410, true, false},
		{401, false, true},
		{403, false, true},
		{429, false, true},
		{500, false, true},
		{503, false, true},
	}
	for _, tc := range tests {
		entry := robotsEntryFromResponse(tc.status, body, time.Hour)
		if entry.allowAll != tc.wantAllow {
			t.Errorf("status %d: allowAll=%v, quer %v", tc.status, entry.allowAll, tc.wantAllow)
		}
		if entry.denyAll != tc.wantDeny {
			t.Errorf("status %d: denyAll=%v, quer %v", tc.status, entry.denyAll, tc.wantDeny)
		}
		if tc.status == 404 && !entry.expiresAt.After(time.Now().Add(59*time.Minute)) {
			t.Errorf("404 deveria usar o TTL longo")
		}
		if tc.status == 429 && entry.expiresAt.After(time.Now().Add(6*time.Minute)) {
			t.Errorf("429 deveria usar bloqueio curto")
		}
	}
	entry := robotsEntryFromResponse(200, body, time.Hour)
	if entry.rules == nil {
		t.Fatal("200 deveria carregar regras")
	}
	group := entry.rules.FindGroup("NIEBot")
	if group.Test("/privado") {
		t.Error("/privado deveria ser bloqueado pelas regras")
	}
}

func TestCheckRedirectTarget(t *testing.T) {
	if err := CheckRedirectTarget("https://example.com/"); err != nil {
		t.Errorf("URL pública deveria ser aceita: %v", err)
	}
	if err := CheckRedirectTarget("http://127.0.0.1/admin"); err == nil {
		t.Error("redirect para loopback deveria ser bloqueado")
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
