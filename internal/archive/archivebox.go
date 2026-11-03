package archive

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"nie/internal/netguard"
	"nie/internal/urlutil"
)

// lookupIP resolve o host no momento da decisão. Testes substituem a função
// para simular uma resposta DNS que muda entre um endereço público e um privado.
var lookupIP = func(ctx context.Context, host string) ([]net.IP, error) {
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	out := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		out = append(out, addr.IP)
	}
	return out, nil
}

// VetPublicURL recusa esquema, porta, nome e qualquer endereço resolvido que
// não seja público. Um nome com um endereço privado entre os resultados é recusado.
func VetPublicURL(ctx context.Context, raw string) error {
	u, _, err := urlutil.ParseNormalized(raw)
	if err != nil {
		return err
	}
	if err := urlutil.ValidateCrawlURL(u); err != nil {
		return err
	}
	host := u.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if !netguard.IsPublicIP(ip) {
			return urlutil.ErrPrivateHost
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ips, err := lookupIP(ctx, host)
	if err != nil {
		return fmt.Errorf("resolução de %s: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("resolução de %s não retornou endereços", host)
	}
	for _, ip := range ips {
		if !netguard.IsPublicIP(ip) {
			return fmt.Errorf("endereço %s de %s bloqueado", ip, host)
		}
	}
	return nil
}

// Add pede ao ArchiveBox uma captura seletiva da URL. O catálogo do NIE não
// passa a ser o ArchiveBox. A saída do processo só existe quando há um proxy
// de egress e cada salto da URL é público.
func Add(ctx context.Context, dataDir, proxy, pageURL string) error {
	dataDir = strings.TrimSpace(dataDir)
	proxy = strings.TrimSpace(proxy)
	if dataDir == "" {
		return fmt.Errorf("ArchiveBox não configurado")
	}
	if proxy == "" {
		return fmt.Errorf("ArchiveBox exige NIE_ARCHIVEBOX_PROXY para isolar a saída")
	}
	if err := VetPublicURL(ctx, pageURL); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "archivebox", "add", "--update", pageURL)
	cmd.Dir = dataDir
	cmd.Env = append(os.Environ(),
		"http_proxy="+proxy,
		"https_proxy="+proxy,
		"HTTP_PROXY="+proxy,
		"HTTPS_PROXY="+proxy,
		"ALL_PROXY="+proxy,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("archivebox: %s", msg)
	}
	return nil
}

// Follow verifica o destino e cada redirecionamento antes de qualquer execução.
func Follow(ctx context.Context, pageURL string, client *http.Client) (string, error) {
	if err := VetPublicURL(ctx, pageURL); err != nil {
		return "", err
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	next := pageURL
	for hop := 0; hop < 5; hop++ {
		if err := VetPublicURL(ctx, next); err != nil {
			return "", err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, next, nil)
		if err != nil {
			return "", err
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		loc := resp.Header.Get("Location")
		resp.Body.Close()
		if resp.StatusCode < 300 || resp.StatusCode > 399 || loc == "" {
			return next, nil
		}
		ref, err := req.URL.Parse(loc)
		if err != nil {
			return "", err
		}
		next = ref.String()
	}
	return "", fmt.Errorf("redirecionamentos demais")
}
