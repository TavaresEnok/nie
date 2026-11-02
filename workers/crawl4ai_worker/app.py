import os
import re
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

@app.get("/health")
def health():
    return {"status": "ok", "worker": "crawl4ai", "version": "2.1.0"}

def html_to_markdown(soup: BeautifulSoup) -> str:
    # Remove scripts, estilos e cabeçalhos desnecessários
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
def crawl(req: CrawlRequest):
    headers = {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 NIE-Crawl4AI/2.1",
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
    }
    try:
        resp = requests.get(req.url, headers=headers, timeout=30)
        resp.raise_for_status()
    except Exception as e:
        raise HTTPException(status_code=502, detail=f"Erro ao renderizar via Crawl4AI bridge: {e}")

    soup = BeautifulSoup(resp.text, "html.parser")
    markdown = html_to_markdown(soup)

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
        content_type="text/markdown",
        markdown=markdown,
        extracted_links=deduped_links[:200],
    )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8086)
