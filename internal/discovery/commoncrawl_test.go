package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCommonCrawlSearch(t *testing.T) {
	var cdx *httptest.Server
	cdx = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !stringsContains(r.URL.RawQuery, "example.com") {
			t.Errorf("padrão ausente: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`[{"url":"https://docs.example.com/manual","timestamp":"20260101000000","status":"200"}]`))
	}))
	defer cdx.Close()
	info := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"cdx-api":"` + cdx.URL + `"}]`))
	}))
	defer info.Close()
	cc := NewCommonCrawl(info.URL, true)
	cc.client = info.Client()
	results, err := cc.Search(context.Background(), "example.com", 5)
	if err != nil || len(results) != 1 || results[0].Source != "commoncrawl" || results[0].URL == "" {
		t.Fatalf("results=%+v err=%v", results, err)
	}
	if NewCommonCrawl(info.URL, false) != nil {
		t.Fatal("adaptador desligado")
	}
}

func stringsContains(value, part string) bool {
	return len(part) == 0 || (len(value) >= len(part) && (value == part || indexOf(value, part) >= 0))
}

func indexOf(value, part string) int {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return i
		}
	}
	return -1
}
