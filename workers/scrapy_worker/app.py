import base64
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
from bs4 import BeautifulSoup, UnicodeDammit

SCRAPY_AVAILABLE = False
try:
    import scrapy
    from scrapy.http import HtmlResponse
    from scrapy.linkextractors import LinkExtractor
    SCRAPY_AVAILABLE = True
except ImportError:
    SCRAPY_AVAILABLE = False

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
    raw_html: str
    raw_b64: Optional[str] = None
    extracted_links: List[str]

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
        if str(ip).startswith("169.254."):
            return False
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
        ip = ipaddress.ip_address(host)
        if not is_public_ip(str(ip)):
            raise HTTPException(status_code=403, detail=f"IP {host} bloqueado por política de egress e SSRF")
    except ValueError:
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

def read_limited_body(resp, max_bytes: int) -> bytes:
    """Lê o corpo da entidade já sem Content-Encoding (gzip/deflate), com teto de bytes.

    O teto vale para os bytes descomprimidos, o que também limita bombas de compressão.
    """
    buf = bytearray()
    for chunk in resp.iter_content(chunk_size=65536):
        buf.extend(chunk)
        if len(buf) > max_bytes:
            raise HTTPException(status_code=413, detail="Tamanho de conteúdo excedeu max_bytes")
    return bytes(buf)

_CHARSET_RE = re.compile(r"charset\s*=\s*[\"']?([\w.:-]+)", re.IGNORECASE)

def decode_html(body: bytes, content_type: str):
    """Decodifica o HTML respeitando charset do cabeçalho, BOM e <meta charset>.

    Retorna (texto, encoding). Os bytes originais não são alterados.
    """
    known = []
    m = _CHARSET_RE.search(content_type or "")
    if m:
        known.append(m.group(1))
    dammit = UnicodeDammit(body, known_definite_encodings=known, is_html=True)
    if dammit.unicode_markup is None:
        return body.decode("utf-8", errors="replace"), "utf-8"
    return dammit.unicode_markup, dammit.original_encoding or "utf-8"

@app.get("/health")
def health():
    return {
        "status": "ok",
        "worker": "scrapy",
        "version": "2.1.0",
        "scrapy_version": scrapy.__version__ if SCRAPY_AVAILABLE else "fallback_bs4",
        "capabilities": ["http", "https", "robots_aware", "ssrf_guard", "raw_content_preservation"]
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
            validate_public_target(curr_url)
            continue
        break

    if resp is None:
        raise HTTPException(status_code=502, detail="Nenhuma resposta recebida do destino")

    content_type = resp.headers.get("Content-Type", "text/html")
    # resp.raw.read() devolveria o corpo ainda comprimido (Content-Encoding);
    # o original preservado é a entidade descomprimida, no charset de origem.
    raw_bytes = read_limited_body(resp, req.max_bytes)

    raw_b64 = base64.b64encode(raw_bytes).decode("ascii")
    raw_html_str, encoding = decode_html(raw_bytes, content_type)

    links = []
    if SCRAPY_AVAILABLE:
        # Utiliza Scrapy HtmlResponse e Scrapy LinkExtractor nativo
        scrapy_resp = HtmlResponse(url=curr_url, body=raw_bytes, encoding=encoding)
        extractor = LinkExtractor(
            deny_extensions=['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'zip', 'tar', 'gz', 'exe', 'bin', 'mp4'],
            tags=('a', 'area'),
            attrs=('href',)
        )
        for l in extractor.extract_links(scrapy_resp):
            full_url = l.url.strip()
            parsed = urlparse(full_url)
            if parsed.scheme in ("http", "https"):
                links.append(full_url)

    soup = BeautifulSoup(raw_html_str, "html.parser")
    for s in soup(["script", "style", "noscript", "svg", "header", "footer"]):
        s.extract()
    text = soup.get_text(separator="\n", strip=True)

    if not SCRAPY_AVAILABLE:
        for a in soup.find_all("a", href=True):
            full_url = urljoin(curr_url, a["href"].strip())
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
        raw_html=raw_html_str,
        raw_b64=raw_b64,
        extracted_links=deduped_links[:200],
    )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8085)
