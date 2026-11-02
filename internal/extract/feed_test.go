package extract

import (
	"net/url"
	"testing"
)

func TestFeedSitemapIndexAndAtom(t *testing.T) {
	base, _ := url.Parse("https://docs.example.org/sitemap_index.xml")
	idx, err := Feed(base, []byte(`<sitemapindex><sitemap><loc>https://docs.example.org/sitemap-1.xml</loc></sitemap></sitemapindex>`))
	if err != nil {
		t.Fatal(err)
	}
	if idx.LinkKind != LinkKindSitemap || len(idx.Links) != 1 || idx.Links[0] != "https://docs.example.org/sitemap-1.xml" {
		t.Errorf("sitemapindex: kind=%q links=%v", idx.LinkKind, idx.Links)
	}

	atom, err := Feed(base, []byte(`<feed xmlns="http://www.w3.org/2005/Atom">
  <link rel="self" href="https://docs.example.org/atom.xml"/>
  <entry><title>Release VRP V800R023</title><link href="/releases/v800r023"/></entry>
  <entry><title>Nota</title><link rel="enclosure" href="/file.bin"/></entry>
</feed>`))
	if err != nil {
		t.Fatal(err)
	}
	if atom.LinkKind != LinkKindFeed || len(atom.Links) != 1 || atom.Links[0] != "https://docs.example.org/releases/v800r023" {
		t.Errorf("atom deve seguir só o link alternate da entrada: kind=%q links=%v", atom.LinkKind, atom.Links)
	}
}

func TestFeedRejectsOtherXMLAndUnsafeLinks(t *testing.T) {
	base, _ := url.Parse("https://docs.example.org/config.xml")
	if _, err := Feed(base, []byte(`<config><loc>https://x.example/</loc></config>`)); err == nil {
		t.Error("XML que não é sitemap/feed deve retornar erro para cair na extração de texto")
	}
	sm, err := Feed(base, []byte(`<urlset><url><loc>javascript:alert(1)</loc></url><url><loc>file:///etc/passwd</loc></url></urlset>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(sm.Links) != 0 {
		t.Errorf("esquemas não HTTP devem ser descartados, obteve %v", sm.Links)
	}
	if IsFeedCandidate("text/html; charset=utf-8", "/sitemap.xml") {
		t.Error("Content-Type HTML tem precedência sobre a extensão .xml")
	}
}
