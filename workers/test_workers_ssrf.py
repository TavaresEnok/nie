import os
import sys
import unittest

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

try:
    from fastapi import HTTPException
except ImportError:
    from starlette.exceptions import HTTPException

from workers.crawl4ai_worker.app import (
    app as crawl_app,
    is_public_ip as crawl_is_public,
    validate_public_target as crawl_validate,
    html_to_markdown,
)
from workers.scrapy_worker.app import (
    app as scrapy_app,
    is_public_ip as scrapy_is_public,
    validate_public_target as scrapy_validate,
)
import workers.scrapy_worker.app as scrapy_module
from bs4 import BeautifulSoup
import base64
import gzip
import http.server
import threading

LATIN1_HTML = '<html><body><p>Café original</p><a href="/docs/next">Próximo</a></body></html>'.encode("iso-8859-1")
META_HTML = '<html><head><meta charset="iso-8859-1"></head><body><p>Configuração</p></body></html>'.encode("iso-8859-1")

class _FixtureHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/meta":
            body, ctype = META_HTML, "text/html"
        elif self.path == "/big":
            body, ctype = b"<html>" + b"a" * 4096 + b"</html>", "text/html"
        else:
            body, ctype = LATIN1_HTML, "text/html; charset=iso-8859-1"
        gz = "gzip" in self.headers.get("Accept-Encoding", "")
        data = gzip.compress(body) if gz else body
        self.send_response(200)
        self.send_header("Content-Type", ctype)
        if gz:
            self.send_header("Content-Encoding", "gzip")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, *args):
        pass

class TestScrapyWorkerRawContentIntegrity(unittest.TestCase):
    """O original preservado deve ser a entidade descomprimida, no charset de origem."""

    @classmethod
    def setUpClass(cls):
        cls.server = http.server.HTTPServer(("127.0.0.1", 0), _FixtureHandler)
        threading.Thread(target=cls.server.serve_forever, daemon=True).start()
        cls.base = f"http://127.0.0.1:{cls.server.server_port}"
        # O servidor de fixture é local; a política de egress é coberta pelos testes acima.
        cls._orig_validate = scrapy_module.validate_public_target
        scrapy_module.validate_public_target = lambda url: None

    @classmethod
    def tearDownClass(cls):
        scrapy_module.validate_public_target = cls._orig_validate
        cls.server.shutdown()
        cls.server.server_close()

    def crawl(self, path, max_bytes=5242880):
        return scrapy_module.crawl(scrapy_module.CrawlRequest(
            task_id="t1", url=self.base + path, campaign_id="c1", max_bytes=max_bytes))

    def test_gzip_body_is_decompressed_and_bytes_preserved(self):
        res = self.crawl("/")
        self.assertEqual(base64.b64decode(res.raw_b64), LATIN1_HTML)
        self.assertIn("Café original", res.text)
        self.assertIn(self.base + "/docs/next", res.extracted_links)

    def test_charset_from_meta_tag(self):
        res = self.crawl("/meta")
        self.assertEqual(base64.b64decode(res.raw_b64), META_HTML)
        self.assertIn("Configuração", res.text)

    def test_max_bytes_applies_to_decompressed_body(self):
        with self.assertRaises(HTTPException):
            self.crawl("/big", max_bytes=1024)

class TestCrawl4AIWorkerHTTPFallbackIntegrity(TestScrapyWorkerRawContentIntegrity):
    """O fallback HTTP do worker de navegador segue as mesmas regras de preservação."""

    @classmethod
    def setUpClass(cls):
        import workers.crawl4ai_worker.app as crawl_module
        cls.crawl_module = crawl_module
        super().setUpClass()
        cls._orig_crawl_validate = crawl_module.validate_public_target
        crawl_module.validate_public_target = lambda url: None

    @classmethod
    def tearDownClass(cls):
        cls.crawl_module.validate_public_target = cls._orig_crawl_validate
        super().tearDownClass()

    def crawl(self, path, max_bytes=5242880):
        import asyncio
        m = self.crawl_module
        return asyncio.run(m.crawl(m.CrawlRequest(
            task_id="t1", url=self.base + path, campaign_id="c1", mode="http", max_bytes=max_bytes)))

    def test_gzip_body_is_decompressed_and_bytes_preserved(self):
        res = self.crawl("/")
        self.assertEqual(base64.b64decode(res.raw_b64), LATIN1_HTML)
        self.assertIn("Café original", res.markdown)
        self.assertIn(self.base + "/docs/next", res.extracted_links)
        self.assertEqual(res.content_type, "text/html; charset=iso-8859-1")
        self.assertEqual(res.rendered_html, "")

    def test_charset_from_meta_tag(self):
        res = self.crawl("/meta")
        self.assertEqual(base64.b64decode(res.raw_b64), META_HTML)
        self.assertIn("Configuração", res.markdown)

class TestWorkerSecurityAndExtraction(unittest.TestCase):
    def test_ssrf_ip_ranges(self):
        blocked_ips = [
            "127.0.0.1",
            "127.0.1.1",
            "10.0.0.1",
            "10.255.255.255",
            "172.16.0.1",
            "172.31.255.255",
            "192.168.1.1",
            "169.254.169.254",
            "169.254.1.1",
            "100.64.0.1",
            "0.0.0.0",
            "::1",
            "fe80::1",
            "fc00::1",
        ]
        for ip in blocked_ips:
            self.assertFalse(crawl_is_public(ip), f"Crawl4AI worker permitted private IP: {ip}")
            self.assertFalse(scrapy_is_public(ip), f"Scrapy worker permitted private IP: {ip}")

        public_ips = [
            "8.8.8.8",
            "1.1.1.1",
            "93.184.216.34",
        ]
        for ip in public_ips:
            self.assertTrue(crawl_is_public(ip), f"Crawl4AI blocked valid public IP: {ip}")
            self.assertTrue(scrapy_is_public(ip), f"Scrapy blocked valid public IP: {ip}")

    def test_validate_public_target_ssrf_blocking(self):
        blocked_targets = [
            "http://127.0.0.1:8080/admin",
            "http://169.254.169.254/latest/meta-data/",
            "http://localhost:5432",
            "http://metadata.google.internal/computeMetadata/v1/",
            "http://192.168.1.1/router",
            "http://10.0.0.5/",
        ]
        for url in blocked_targets:
            with self.assertRaises(HTTPException):
                crawl_validate(url)

            with self.assertRaises(HTTPException):
                scrapy_validate(url)

    def test_redirect_to_private_host_is_rejected(self):
        from workers.crawl4ai_worker.app import assert_public_redirect

        with self.assertRaises(HTTPException):
            assert_public_redirect("https://docs.example/manual", "http://127.0.0.1/secret")
        with self.assertRaises(HTTPException):
            assert_public_redirect("https://docs.example/manual", "http://169.254.169.254/latest/meta-data/")

    def test_dns_answer_with_private_address_is_rejected(self):
        import socket
        from unittest.mock import patch

        def fake_getaddrinfo(host, *args, **kwargs):
            return [(socket.AF_INET, socket.SOCK_STREAM, 6, "", ("10.1.1.1", 443))]

        with patch("workers.crawl4ai_worker.app.socket.getaddrinfo", fake_getaddrinfo):
            with self.assertRaises(HTTPException):
                crawl_validate("https://rebind.example/a")

    def test_markdown_can_use_crawl4ai_on_already_fetched_html(self):
        from workers.crawl4ai_worker.app import render_markdown

        html = "<html><body><p>Atomic fact</p></body></html>"
        soup = BeautifulSoup(html, "html.parser")
        text = render_markdown(html, soup, converter=lambda raw: "FROM-CRAWL4AI " + raw)
        self.assertTrue(text.startswith("FROM-CRAWL4AI"))
        self.assertIn("Atomic fact", text)

    def test_html_content_markdown_extraction_success(self):
        # Simula HTML rico que teria sido renderizado por JavaScript
        js_rendered_html = """
        <html>
            <head><title>Dynamic JS Page</title></head>
            <body>
                <h1>Telecom Dynamic Manual</h1>
                <p>Configuração gerada via client-side script:</p>
                <pre>bgp 65000\n peer 10.1.1.1 as-number 65001</pre>
                <a href="/docs/bgp-advanced">Advanced Peering</a>
                <a href="javascript:void(0)">Ignored</a>
            </body>
        </html>
        """
        soup = BeautifulSoup(js_rendered_html, "html.parser")
        md = html_to_markdown(soup)
        self.assertIn("# Telecom Dynamic Manual", md)
        self.assertIn("```\nbgp 65000", md)

if __name__ == "__main__":
    unittest.main()
