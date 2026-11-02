package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMwmblSearch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "bgp evpn" {
			t.Errorf("query inesperada: %s", r.URL.Query().Get("q"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"results": [
				{"url": "https://example.com/bgp-evpn", "title": "BGP EVPN Guide", "snippet": "Technical overview"}
			]
		}`))
	}))
	defer ts.Close()

	m := NewMwmbl(ts.URL)
	results, err := m.Search(context.Background(), "bgp evpn", 5)
	if err != nil {
		t.Fatalf("erro no Mwmbl search: %v", err)
	}
	if len(results) != 1 || results[0].URL != "https://example.com/bgp-evpn" {
		t.Fatalf("resultado inesperado: %+v", results)
	}
	if results[0].Source != "mwmbl" {
		t.Errorf("fonte esperada mwmbl, obteve %s", results[0].Source)
	}
}

func TestWaybackMachineAvailability(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"archived_snapshots": {
				"closest": {
					"available": true,
					"url": "https://web.archive.org/web/20210101000000/https://example.com/legacy-doc.pdf",
					"timestamp": "20210101000000",
					"status": "200"
				}
			}
		}`))
	}))
	defer ts.Close()

	wb := NewWaybackMachine(ts.URL)
	snap, err := wb.CheckAvailability(context.Background(), "https://example.com/legacy-doc.pdf")
	if err != nil {
		t.Fatalf("erro no Wayback check: %v", err)
	}
	if snap == nil || !snap.Available {
		t.Fatalf("esperava snapshot disponível")
	}
	if snap.SnapshotURL != "https://web.archive.org/web/20210101000000/https://example.com/legacy-doc.pdf" {
		t.Errorf("snapshot URL inesperada: %s", snap.SnapshotURL)
	}
}

func TestSoftwareHeritageSearch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"url": "https://github.com/FRRouting/frr", "type": "git"}
		]`))
	}))
	defer ts.Close()

	sh := NewSoftwareHeritage(ts.URL)
	results, err := sh.SearchOrigins(context.Background(), "frr", 5)
	if err != nil {
		t.Fatalf("erro no SoftwareHeritage: %v", err)
	}
	if len(results) != 1 || results[0].URL != "https://github.com/FRRouting/frr" {
		t.Fatalf("resultado inesperado: %+v", results)
	}
	if results[0].Source != "software_heritage" {
		t.Errorf("fonte esperada software_heritage, obteve %s", results[0].Source)
	}
}
