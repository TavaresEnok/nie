import ipaddress
import os
import re
import socket
from typing import List, Optional
from urllib.parse import urljoin, urlparse

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import requests
import scrapy
from scrapy.http import HtmlResponse
from scrapy.linkextractors import LinkExtractor
from bs4 import BeautifulSoup

app = FastAPI(title="NIE Scrapy Worker Bridge", version="2.1.0")

class CrawlRequest(BaseModel):
    task_id: str
    url: str
    campaign_id: str
    max_bytes: Optional[int] = 5242880

class CrawlResponse(BaseModel):
    task_id: str
    effective_url: str
    status_code: int
    content_type: str
    text: str
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

@app.get("/health")
def health():
    return {
        "status": "ok",
        "worker": "scrapy",
        "version": "2.1.0",
        "scrapy_version": scrapy.__version__,
        "capabilities": ["http", "https", "robots_aware", "ssrf_guard"]
    }

@app.post("/crawl", response_model=CrawlResponse)
def crawl(req: CrawlRequest):
    headers = {
        "User-Agent": "NIE-ScrapyWorker/2.1 (+https://github.com/TavaresEnok/nie)",
        "Accept": "text/html,application/xhtml+xml,text/plain;q=0.9,*/*;q=0.8",
    }

    session = requests.Session()
    curr_url = req.url
    resp = None

    for _ in range(5):
        validate_public_target(curr_url)
        try:
            resp = session.get(curr_url, headers=headers, timeout=20, allow_redirects=False, stream=True)
        except Exception as e:
            raise HTTPException(status_code=502, detail=f"Erro ao coletar URL via Scrapy worker: {e}")

        if resp.status_code in (301, 302, 303, 307, 308):
            loc = resp.headers.get("Location")
            if not loc:
                break
            curr_url = urljoin(curr_url, loc)
            continue
        break

    if resp is None:
        raise HTTPException(status_code=502, detail="Nenhuma resposta recebida do destino")

    content_type = resp.headers.get("Content-Type", "text/html")
    raw_bytes = resp.raw.read(req.max_bytes + 1)
    if len(raw_bytes) > req.max_bytes:
        raise HTTPException(status_code=413, detail="Tamanho de conteúdo excedeu max_bytes")

    # Utiliza Scrapy HtmlResponse e Scrapy LinkExtractor nativo
    scrapy_resp = HtmlResponse(url=curr_url, body=raw_bytes, encoding="utf-8")
    extractor = LinkExtractor(
        deny_extensions=['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'zip', 'tar', 'gz', 'exe', 'bin', 'mp4'],
        tags=('a', 'area'),
        attrs=('href',)
    )
    extracted_scrapy_links = extractor.extract_links(scrapy_resp)

    soup = BeautifulSoup(raw_bytes.decode("utf-8", errors="replace"), "html.parser")
    for s in soup(["script", "style", "noscript", "svg", "header", "footer"]):
        s.extract()
    text = soup.get_text(separator="\n", strip=True)

    links = []
    for l in extracted_scrapy_links:
        full_url = l.url.strip()
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
        content_type=content_type,
        text=text,
        extracted_links=deduped_links[:200],
    )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8085)
