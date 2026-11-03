package archive

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestAddRejectsPrivateURLBeforeCommand(t *testing.T) {
	err := Add(context.Background(), t.TempDir(), "http://127.0.0.1:9", "http://127.0.0.1/admin")
	if err == nil {
		t.Fatal("URL privada não pode chegar ao ArchiveBox")
	}
}

func TestAddRequiresEgressProxy(t *testing.T) {
	err := Add(context.Background(), t.TempDir(), " ", "https://example.com/manual")
	if err == nil || !strings.Contains(err.Error(), "NIE_ARCHIVEBOX_PROXY") {
		t.Fatalf("sem proxy de saída: %v", err)
	}
}

func TestAddRequiresDirectory(t *testing.T) {
	err := Add(context.Background(), " ", "http://127.0.0.1:9", "https://example.com/")
	if err == nil {
		t.Fatal("sem diretório o comando não deve existir")
	}
}

func TestVetPublicURLRejectsRebinding(t *testing.T) {
	previous := lookupIP
	lookupIP = func(context.Context, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("10.1.1.1")}, nil
	}
	t.Cleanup(func() { lookupIP = previous })
	if err := VetPublicURL(context.Background(), "https://rebind.example/a"); err == nil {
		t.Fatal("um endereço privado na resposta DNS deveria bloquear o nome")
	}
}

func TestFollowRejectsRedirectToPrivateHost(t *testing.T) {
	previous := lookupIP
	lookupIP = func(context.Context, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("8.8.8.8")}, nil
	}
	t.Cleanup(func() { lookupIP = previous })
	client := &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		header := make(http.Header)
		header.Set("Location", "http://127.0.0.1/secret")
		return &http.Response{StatusCode: http.StatusFound, Header: header, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	})}
	if _, err := Follow(context.Background(), "https://docs.example/manual", client); err == nil {
		t.Fatal("redirecionamento para host privado deveria ser recusado")
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (fn roundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }
