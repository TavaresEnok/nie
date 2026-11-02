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
from bs4 import BeautifulSoup

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
