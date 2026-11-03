package discovery

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenSearchQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nie/_search" {
			t.Errorf("path=%s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !stringsHas(string(body), "display bgp") {
			t.Errorf("body=%s", body)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "nie" || pass != "segredo" {
			t.Fatalf("auth %s %s %v", user, pass, ok)
		}
		_, _ = w.Write([]byte(`{"hits":{"hits":[{"_id":"1","_score":1.5,"_source":{"title":"manual"}}]}}`))
	}))
	defer server.Close()
	client := NewOpenSearch(server.URL, "nie", "nie", "segredo")
	client.client = server.Client()
	hits, err := client.Search(context.Background(), "display bgp", 5)
	if err != nil || len(hits) != 1 || hits[0].ID != "1" {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
}

func stringsHas(value, part string) bool {
	return len(value) >= len(part) && (value == part || indexHas(value, part))
}

func indexHas(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
