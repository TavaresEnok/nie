import ipaddress
import os
import re
import socket
from typing import List, Optional
from urllib.parse import urljoin, urlparse

try:
    from fastapi import FastAPI, HTTPException
except ImportError:
    from starlette.applications import Starlette
    from starlette.exceptions import HTTPException

    class FastAPI(Starlette):
        def __init__(self, *args, **kwargs):
            kwargs.pop("title", None)
            kwargs.pop("version", None)
            super().__init__(*args, **kwargs)

        def post(self, *args, **kwargs):
            def decorator(func):
                return func
            return decorator

        def get(self, *args, **kwargs):
            def decorator(func):
                return func
            return decorator

from pydantic import BaseModel
import requests
from bs4 import BeautifulSoup

app = FastAPI(title="NIE Browser / Crawl4AI Bridge", version="2.1.0")

class CrawlRequest(BaseModel):
    task_id: str
    url: str
    campaign_id: str
    mode: Optional[str] = "browser"

class CrawlResponse(BaseModel):
    task_id: str
    effective_url: str
    status_code: int
    content_type: str = "text/markdown"
    markdown: str
    raw_html: str
    extracted_links: List[str]
    browser_success: bool = False
    browser_error: str = ""
    fallback_reason: str = ""

BLOCKED_HOSTNAMES = {
    "localhost",
    "metadata.google.internal",
    "metadata.internal",
    "instance-data",
    "host.docker.internal",
}

def is_public_ip(ip_str: str) -> bool:
    try:
        ip = ipaddress.ip_address(ip_str)
        if not ip.is_global:
            return False
        if (
            ip.is_loopback
            or ip.is_private
            or ip.is_link_local
            or ip.is_multicast
            or ip.is_reserved
            or ip.is_unspecified
        ):
            return False
        # Link-local / Cloud metadata range 169.254.0.0/16
        if str(ip).startswith("169.254."):
            return False
        # Carrier-grade NAT 100.64.0.0/10
        if ip.version == 4 and ip in ipaddress.ip_network("100.64.0.0/10"):
            return False
        return True
    except ValueError:
        return False

def validate_public_target(url: str):
    parsed = urlparse(url)
    if parsed.scheme not in ("http", "https"):
        raise HTTPException(status_code=400, detail="Apenas protocolos HTTP e HTTPS são permitidos")
    host = parsed.hostname
    if not host:
        raise HTTPException(status_code=400, detail="Hostname inválido")

    lower_host = host.lower()
    if lower_host in BLOCKED_HOSTNAMES or lower_host.endswith(".local") or lower_host.endswith(".internal"):
        raise HTTPException(status_code=403, detail=f"Hostname {host} bloqueado por política de egress")

    try:
        # Tenta interpretar se já é um IP literal
        ip = ipaddress.ip_address(host)
        if not is_public_ip(str(ip)):
            raise HTTPException(status_code=403, detail=f"IP {host} bloqueado por política de egress e SSRF")
    except ValueError:
        # É um hostname de domínio, resolve DNS
        try:
            addr_info = socket.getaddrinfo(
                host,
                parsed.port or (443 if parsed.scheme == "https" else 80),
                proto=socket.IPPROTO_TCP
            )
            if not addr_info:
                raise HTTPException(status_code=400, detail="Não foi possível resolver o hostname")
            for _, _, _, _, sockaddr in addr_info:
                ip_str = sockaddr[0]
                if not is_public_ip(ip_str):
                    raise HTTPException(status_code=403, detail=f"Destino {ip_str} bloqueado por política de egress e SSRF")
        except socket.gaierror as e:
            raise HTTPException(status_code=400, detail=f"Erro de resolução DNS: {e}")

PLAYWRIGHT_AVAILABLE = False
try:
    from playwright.async_api import async_playwright
    PLAYWRIGHT_AVAILABLE = True
except ImportError:
    PLAYWRIGHT_AVAILABLE = False

@app.get("/health")
def health():
    return {
        "status": "ok",
        "worker": "crawl4ai_browser",
        "version": "2.1.0",
        "playwright_enabled": PLAYWRIGHT_AVAILABLE,
        "capabilities": ["browser", "markdown", "ssrf_guard", "subresource_firewall"]
    }

def html_to_markdown(soup: BeautifulSoup) -> str:
    for tag in soup(["script", "style", "noscript", "svg", "iframe"]):
        tag.decompose()

    lines = []
    for el in soup.find_all(["h1", "h2", "h3", "h4", "p", "pre", "code", "li"]):
        text = el.get_text(strip=True)
        if not text:
            continue
        if el.name == "h1":
            lines.append(f"# {text}\n")
        elif el.name == "h2":
            lines.append(f"## {text}\n")
        elif el.name == "h3":
            lines.append(f"### {text}\n")
        elif el.name in ("pre", "code"):
            lines.append(f"```\n{text}\n```\n")
        elif el.name == "li":
            lines.append(f"- {text}")
        else:
            lines.append(f"{text}\n")
    return "\n".join(lines)

@app.post("/crawl", response_model=CrawlResponse)
async def crawl(req: CrawlRequest):
    validate_public_target(req.url)

    html_content = ""
    effective_url = req.url
    status_code = 200
    browser_success = False
    browser_error = ""
    fallback_reason = ""

    # Caminho 1: Playwright Browser Worker (para páginas dinâmicas e SPAs)
    if req.mode == "browser" and PLAYWRIGHT_AVAILABLE:
        try:
            async with async_playwright() as p:
                browser = await p.chromium.launch(
                    headless=True,
                    args=["--no-sandbox", "--disable-dev-shm-usage"]
                )
                context = await browser.new_context(
                    user_agent="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 NIE-Browser/2.1"
                )
                page = await context.new_page()

                # P0-03: Intercepta e bloqueia cada requisição individual (subrecursos, XHR, redirects) contra SSRF
                async def intercept_route(route):
                    req_target = route.request.url
                    try:
                        validate_public_target(req_target)
                        await route.continue_()
                    except Exception:
                        await route.abort("blockedbyclient")

                await page.route("**/*", intercept_route)

                resp = await page.goto(req.url, timeout=30000, wait_until="domcontentloaded")
                effective_url = page.url
                validate_public_target(effective_url)
                status_code = resp.status if resp else 200
                html_content = await page.content()
                browser_success = True
                await browser.close()
        except Exception as e:
            browser_error = str(e)
            fallback_reason = f"Playwright indisponível ou falhou ({e}); acionando fallback HTTP seguro"
            html_content = ""

    # Caminho 2: Fallback transparente para coletor HTTP com proteção de redirecionamentos
    if not html_content:
        headers = {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 NIE-Crawl4AI/2.1",
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
        }

        session = requests.Session()
        curr_url = req.url
        resp = None

        for _ in range(5):
            validate_public_target(curr_url)
            try:
                resp = session.get(curr_url, headers=headers, timeout=30, allow_redirects=False)
            except Exception as e:
                raise HTTPException(status_code=502, detail=f"Erro ao renderizar via coletor HTTP: {e}")

            if resp.status_code in (301, 302, 303, 307, 308):
                loc = resp.headers.get("Location")
                if not loc:
                    break
                curr_url = urljoin(curr_url, loc)
                continue
            break

        if resp is None:
            raise HTTPException(status_code=502, detail="Nenhuma resposta recebida do destino")

        status_code = resp.status_code
        effective_url = curr_url
        html_content = resp.text

    # P0-02: Processa html_content (tanto do browser quanto do fallback HTTP)
    soup = BeautifulSoup(html_content, "html.parser")
    markdown = html_to_markdown(soup)

    links = []
    base_url = effective_url
    for a in soup.find_all("a", href=True):
        href = a["href"].strip()
        if href.startswith(("#", "javascript:", "mailto:", "tel:")):
            continue
        full_url = urljoin(base_url, href)
        parsed = urlparse(full_url)
        if parsed.scheme in ("http", "https"):
            links.append(full_url)

    seen = set()
    deduped_links = []
    for l in links:
        if l not in seen:
            seen.add(l)
            deduped_links.append(l)

    return CrawlResponse(
        task_id=req.task_id,
        effective_url=effective_url,
        status_code=status_code,
        content_type="text/markdown",
        markdown=markdown,
        raw_html=html_content,
        extracted_links=deduped_links[:200],
        browser_success=browser_success,
        browser_error=browser_error,
        fallback_reason=fallback_reason,
    )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8086)
