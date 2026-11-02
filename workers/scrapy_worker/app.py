import os
import re
from typing import List, Optional
from urllib.parse import urljoin, urlparse

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import requests
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

@app.get("/health")
def health():
    return {"status": "ok", "worker": "scrapy", "version": "2.1.0"}

@app.post("/crawl", response_model=CrawlResponse)
def crawl(req: CrawlRequest):
    headers = {
        "User-Agent": "NIE-ScrapyWorker/2.1 (+https://github.com/TavaresEnok/nie)",
        "Accept": "text/html,application/xhtml+xml,text/plain;q=0.9,*/*;q=0.8",
    }
    try:
        resp = requests.get(req.url, headers=headers, timeout=20, stream=True)
        resp.raise_for_status()
    except Exception as e:
        raise HTTPException(status_code=502, detail=f"Erro ao coletar URL via Scrapy worker: {e}")

    content_type = resp.headers.get("Content-Type", "text/html")
    raw_bytes = resp.raw.read(req.max_bytes + 1)
    if len(raw_bytes) > req.max_bytes:
        raise HTTPException(status_code=413, detail="Tamanho de conteúdo excedeu max_bytes")

    html_text = raw_bytes.decode("utf-8", errors="replace")
    soup = BeautifulSoup(html_text, "html.parser")

    # Remove tags que não são texto útil
    for s in soup(["script", "style", "noscript", "svg", "header", "footer"]):
        s.extract()

    text = soup.get_text(separator="\n", strip=True)

    links = []
    base_url = resp.url
    for a in soup.find_all("a", href=True):
        href = a["href"].strip()
        if href.startswith(("#", "javascript:", "mailto:", "tel:")):
            continue
        full_url = urljoin(base_url, href)
        parsed = urlparse(full_url)
        if parsed.scheme in ("http", "https"):
            links.append(full_url)

    # Deduplica links mantendo ordem
    seen = set()
    deduped_links = []
    for l in links:
        if l not in seen:
            seen.add(l)
            deduped_links.append(l)

    return CrawlResponse(
        task_id=req.task_id,
        effective_url=resp.url,
        status_code=resp.status_code,
        content_type=content_type,
        text=text,
        extracted_links=deduped_links[:200],
    )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8085)
