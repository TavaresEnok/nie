package semantic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbedOpenAIShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vec := make([]float32, Dimensions)
		vec[0] = 0.5
		write := map[string]any{"data": []map[string]any{{"embedding": vec}}}
		w.Header().Set("Content-Type", "application/json")
		_ = jsonNewEncoder(w, write)
	}))
	defer srv.Close()
	got, err := New(srv.URL, "nomic-embed-text", "").Embed(context.Background(), "bgp peer")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != Dimensions || got[0] != 0.5 {
		t.Fatalf("vetor = %v", got[:1])
	}
}

func TestEmbedRejectsWrongSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = jsonNewEncoder(w, map[string]any{"embeddings": [][]float32{{1, 2, 3}}})
	}))
	defer srv.Close()
	if _, err := New(srv.URL, "m", "").Embed(context.Background(), "texto"); err == nil {
		t.Fatal("dimensão errada deveria ser recusada")
	}
}

func TestNewWithoutURL(t *testing.T) {
	if New("  ", "m", "") != nil {
		t.Fatal("sem URL não há cliente")
	}
}
