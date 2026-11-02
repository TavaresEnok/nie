package crawler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"nie/internal/domain"
)

// newTestFetcher devolve um Fetcher real cujo transporte disca sempre para o
// servidor de teste. A política de egress (que bloqueia 127.0.0.1) é coberta
// por outros testes; aqui só interessa o tratamento do corpo.
func newTestFetcher(t *testing.T, handler http.Handler) *Fetcher {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	f := NewFetcher("NIEBot/test", 10*time.Second, 5<<20, 50<<20, 5<<20)
	addr := ts.Listener.Addr().String()
	f.client.Transport = &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
	}
	return f
}

// O executor nativo (caminho padrão) respeita o teto por tarefa, que carrega o
// saldo do orçamento da campanha, e não só o limite global do tipo.
func TestNativeExecutorAppliesTaskMaxBytes(t *testing.T) {
	body := "<html><body>" + strings.Repeat("a", 64<<10) + "</body></html>" // ~64 KiB
	f := newTestFetcher(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	exec := NewNativeExecutor(f)

	// Saldo de 16 KiB: o download de ~64 KiB deve ser recusado.
	_, err := exec.Execute(context.Background(), domain.FetchTask{URL: "http://docs.example.org/big", MaxBytes: 16 << 10})
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("download acima do saldo deveria falhar com ErrTooLarge, obteve %v", err)
	}

	// Com saldo suficiente, o mesmo recurso é baixado.
	res, err := exec.Execute(context.Background(), domain.FetchTask{URL: "http://docs.example.org/big", MaxBytes: 1 << 20})
	if err != nil || len(res.RawContent) != len(body) {
		t.Fatalf("download dentro do saldo falhou: err=%v bytes=%d", err, len(res.RawContent))
	}
}

// Regressão da auditoria V5: robots.txt que redireciona (→ /policy.txt) não pode
// fazer a verificação de robots chamar a si mesma; a política redirecionada vale.
func TestRobotsRedirectDoesNotRecurse(t *testing.T) {
	var robotsHits int
	f := newTestFetcher(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			robotsHits++
			http.Redirect(w, r, "/policy.txt", http.StatusMovedPermanently)
		case "/policy.txt":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("User-agent: *\nDisallow: /private\n"))
		default:
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html><body>ok</body></html>"))
		}
	}))
	robots := NewRobots(f, "NIEBot/test", slog.New(slog.NewTextHandler(io.Discard, nil)))
	// O mesmo encadeamento que o worker instala.
	f.SetRedirectCheck(func(ctx context.Context, u *url.URL) error {
		if !robots.Allowed(ctx, u) {
			return ErrRobotsDisallowed
		}
		return nil
	})

	done := make(chan [2]bool, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		done <- [2]bool{
			robots.Allowed(ctx, mustURL(t, "http://docs.example.org/private/x")),
			robots.Allowed(ctx, mustURL(t, "http://docs.example.org/public")),
		}
	}()
	select {
	case got := <-done:
		if got[0] || !got[1] {
			t.Errorf("política redirecionada não aplicada: /private=%v /public=%v", got[0], got[1])
		}
	case <-time.After(10 * time.Second):
		t.Fatal("verificação de robots não terminou: recursão no redirecionamento do robots.txt")
	}
	if robotsHits != 1 {
		t.Errorf("robots.txt deveria ser buscado uma vez (cache), foi %d", robotsHits)
	}
}
