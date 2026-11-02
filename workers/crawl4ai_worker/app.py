import ipaddress
import os
import re
import socket
from typing import List, Optional
from urllib.parse import urljoin, urlparse

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import requests
from bs4 import BeautifulSoup

app = FastAPI(title="NIE Crawl4AI Deep/JS Bridge", version="2.1.0")

class CrawlRequest(BaseModel):
    task_id: str
    url: str
    campaign_id: str
    mode: Optional[str] = "browser"

class CrawlResponse(BaseModel):
    task_id: str
    effective_url: str
    status_code: int
    content_type: str
    markdown: str
    extracted_links: List[str]

def is_public_ip(ip_str: str) -> bool:
    try:
        ip = ipaddress.ip_address(ip_str)
        if not ip.is_global:
            return False
        if ip.is_loopback or ip.is_private or ip.is_link_local or ip.is_multicast or ip.is_reserved or ip.is_unspecified:
            return False
        if str(ip).startswith("169.254."):
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

    try:
        addr_info = socket.getaddrinfo(host, parsed.port or (443 if parsed.scheme == "https" else 80), proto=socket.IPPROTO_TCP)
        if not addr_info:
            raise HTTPException(status_code=400, detail="Não foi possível resolver o hostname")
        for family, socktype, proto, canonname, sockaddr in addr_info:
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
        "worker": "crawl4ai",
        "version": "2.1.0",
        "playwright_enabled": PLAYWRIGHT_AVAILABLE,
        "capabilities": ["browser", "markdown", "ssrf_guard"]
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

    if req.mode == "browser" and PLAYWRIGHT_AVAILABLE:
        try:
            async with async_playwright() as p:
                browser = await p.chromium.launch(headless=True, args=["--no-sandbox", "--disable-dev-shm-usage"])
                context = await browser.new_context(user_agent="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 NIE-Crawl4AI/2.1")
                page = await context.new_page()
                resp = await page.goto(req.url, timeout=30000, wait_until="domcontentloaded")
                effective_url = page.url
                validate_public_target(effective_url)
                status_code = resp.status if resp else 200
                html_content = await page.content()
                await browser.close()
        except Exception:
            # Fallback transparente para coletor HTTP caso o browser encontre restrições
            html_content = ""

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
                raise HTTPException(status_code=502, detail=f"Erro ao renderizar via Crawl4AI bridge: {e}")

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

    soup = BeautifulSoup(resp.text, "html.parser")
    markdown = html_to_markdown(soup)

    links = []
    base_url = curr_url
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
        effective_url=curr_url,
        status_code=resp.status_code,
        content_type="text/markdown",
        markdown=markdown,
        extracted_links=deduped_links[:200],
    )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8086)
