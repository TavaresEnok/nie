package discovery

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResearcherReportIsExternalText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/report" {
			t.Errorf("path=%s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) == "" {
			t.Fatal("corpo vazio")
		}
		_, _ = w.Write([]byte(`{"report":"relatório não certificado"}`))
	}))
	defer server.Close()
	client := NewResearcher(server.URL)
	client.client = server.Client()
	report, err := client.Report(context.Background(), "Huawei NE8000 PPPoE")
	if err != nil || report != "relatório não certificado" {
		t.Fatalf("report=%q err=%v", report, err)
	}
}
