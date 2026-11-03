package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestZoektSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "bgp peer" {
			t.Errorf("q=%s", r.URL.Query().Get("q"))
		}
		_, _ = w.Write([]byte(`{"Result":{"FileMatches":[{"FileName":"bgp.rsc","Repo":"isp/router","Matches":[{"LineNum":12,"Fragments":[{"Pre":"/routing bgp connection"}]}]}]}}`))
	}))
	defer server.Close()
	z := NewZoekt(server.URL)
	z.client = server.Client()
	hits, err := z.Search(context.Background(), "bgp peer", 5)
	if err != nil || len(hits) != 1 || hits[0].Line != 12 || hits[0].Path != "bgp.rsc" {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
	if NewZoekt("") != nil {
		t.Fatal("sem URL não há cliente")
	}
}
