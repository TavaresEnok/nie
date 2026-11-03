package discovery

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubFileAtRef(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte("display bgp peer"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ref") != "abc123" {
			t.Errorf("ref=%s", r.URL.Query().Get("ref"))
		}
		_, _ = w.Write([]byte(`{"sha":"blob","html_url":"https://example/blob","encoding":"base64","content":"` + payload + `"}`))
	}))
	defer server.Close()
	gh := NewGitHub("token", true)
	gh.base = server.URL
	gh.client = server.Client()
	file, err := gh.File(context.Background(), "isp/router", "bgp.rsc", "abc123")
	if err != nil || file.SHA != "blob" || file.Content != "display bgp peer" || file.Ref != "abc123" {
		t.Fatalf("file=%+v err=%v", file, err)
	}
}

func TestGitLabFileAtRef(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte("set protocols bgp"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"blob_id":"glblob","encoding":"base64","content":"` + payload + `"}`))
	}))
	defer server.Close()
	gl := NewGitLab(server.URL, "tok", true)
	gl.client = server.Client()
	file, err := gl.File(context.Background(), "isp/router", "bgp.conf", "v7")
	if err != nil || file.SHA != "glblob" || file.Content != "set protocols bgp" {
		t.Fatalf("file=%+v err=%v", file, err)
	}
}
