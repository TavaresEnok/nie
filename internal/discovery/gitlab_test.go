package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitLabSearchReadmes(t *testing.T) {
	var sawToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawToken = r.Header.Get("PRIVATE-TOKEN")
		if !strings.Contains(r.URL.RawQuery, "search=bgp") {
			t.Errorf("consulta ausente: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":7,"path_with_namespace":"isp/pppoe","web_url":"https://gitlab.example/isp/pppoe","description":"notas","default_branch":"v7"}]`))
	}))
	defer server.Close()
	gl := NewGitLab(server.URL, "token-teste", true)
	gl.client = server.Client()
	results, err := gl.SearchReadmes(context.Background(), "bgp", 5)
	if err != nil {
		t.Fatal(err)
	}
	if sawToken != "token-teste" || len(results) != 1 || results[0].URL != "https://gitlab.example/isp/pppoe/-/raw/v7/README.md" || results[0].Source != "gitlab" {
		t.Fatalf("results=%+v token=%s", results, sawToken)
	}
	if NewGitLab(server.URL, "", false) != nil {
		t.Fatal("adaptador desligado não deveria existir")
	}
}
