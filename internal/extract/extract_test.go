package extract

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"testing"
)

func TestHTML(t *testing.T) {
	body := []byte(`<!doctype html>
<html><head><title>BGP Guide</title><style>body{color:red}</style></head>
<body>
  <h1>Configuração</h1>
  <p>Use <code>display bgp peer</code> para verificar as sessões.</p>
  <script>alert('não deve aparecer')</script>
  <a href="/next">Próxima</a>
  <a href="mailto:ops@example.org">Contato</a>
</body></html>`)
	base, _ := url.Parse("https://example.org/docs/")
	res, err := HTML(base, "text/html; charset=utf-8", body)
	if err != nil {
		t.Fatal(err)
	}
	if res.Title != "BGP Guide" {
		t.Errorf("título = %q", res.Title)
	}
	if !strings.Contains(res.Text, "display bgp peer") {
		t.Errorf("texto não contém o comando: %q", res.Text)
	}
	if strings.Contains(res.Text, "alerta") || strings.Contains(res.Text, "não deve aparecer") {
		t.Errorf("script não deveria ser extraído: %q", res.Text)
	}
	if len(res.Links) != 1 || res.Links[0] != "https://example.org/next" {
		t.Errorf("links = %v", res.Links)
	}
	if len(res.Passages) == 0 {
		t.Fatal("deveria produzir trechos")
	}
	if res.Passages[0].Ordinal != 0 || res.Passages[0].Page != nil {
		t.Errorf("metadados do trecho inesperados: %+v", res.Passages[0])
	}
}

func TestText(t *testing.T) {
	res, err := Text([]byte("Primeira linha\r\n\r\n\r\nSegunda linha\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Title != "Primeira linha" {
		t.Errorf("título = %q", res.Title)
	}
	if res.Text != "Primeira linha\n\nSegunda linha" {
		t.Errorf("texto = %q", res.Text)
	}
	if res.MIME != "text/plain" {
		t.Errorf("mime = %q", res.MIME)
	}
}

func TestMakePassagesSplitsLongText(t *testing.T) {
	text := strings.Repeat("palavra ", 100)
	got := MakePassages(text, nil, 200)
	if len(got) < 2 {
		t.Fatalf("esperava múltiplos trechos, obteve %d", len(got))
	}
	for i, p := range got {
		if p.Ordinal != i {
			t.Errorf("ordinal = %d, quer %d", p.Ordinal, i)
		}
		if len(p.Text) > 220 {
			t.Errorf("trecho %d tem %d caracteres", i, len(p.Text))
		}
	}
}

func TestPDF(t *testing.T) {
	data := minimalPDF("Hello NIE PDF")
	res, err := PDF(data)
	if err != nil {
		t.Fatalf("PDF: %v", err)
	}
	if !strings.Contains(res.Text, "Hello NIE PDF") {
		t.Fatalf("texto extraído = %q", res.Text)
	}
	if len(res.Passages) != 1 || res.Passages[0].Page == nil || *res.Passages[0].Page != 1 {
		t.Fatalf("trechos/página inesperados: %+v", res.Passages)
	}
}

func TestPDFInvalid(t *testing.T) {
	if _, err := PDF([]byte("isto não é um PDF")); err == nil {
		t.Fatal("PDF inválido deveria retornar erro")
	}
}

// minimalPDF gera um PDF 1.4 de uma página com texto simples e xref válido.
func minimalPDF(text string) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, 6)
	content := fmt.Sprintf("BT /F1 24 Tf 72 720 Td (%s) Tj ET", text)

	offsets[1] = buf.Len()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	offsets[2] = buf.Len()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	offsets[3] = buf.Len()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")
	offsets[4] = buf.Len()
	fmt.Fprintf(&buf, "4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(content), content)
	offsets[5] = buf.Len()
	buf.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>\nendobj\n")

	xref := buf.Len()
	buf.WriteString("xref\n0 6\n")
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xref)
	return buf.Bytes()
}

func TestExtractURLLinks(t *testing.T) {
	text := "Consulte o manual oficial em https://support.huawei.com/enterprise/doc e https://wiki.mikrotik.com/wiki/BGP."
	links := ExtractURLLinks(text)
	if len(links) != 2 {
		t.Fatalf("Esperava 2 links extraídos, obteve %d: %v", len(links), links)
	}
	if links[0] != "https://support.huawei.com/enterprise/doc" {
		t.Errorf("Link 1 incorreto: %s", links[0])
	}
	if links[1] != "https://wiki.mikrotik.com/wiki/BGP" {
		t.Errorf("Link 2 incorreto: %s", links[1])
	}
}

func TestOCRArgsWriteToStdout(t *testing.T) {
	if got := strings.Join(ocrArgs("pdftotext", "in.pdf"), " "); got != "-layout in.pdf -" {
		t.Errorf("pdftotext deve escrever em stdout com '-', obteve %q", got)
	}
	if got := strings.Join(ocrArgs("tesseract", "scan.png"), " "); got != "scan.png stdout" {
		t.Errorf("tesseract deve usar 'stdout', obteve %q", got)
	}
}

func TestExtractPDFAnnotationLinksAndOffsets(t *testing.T) {
	rawPDF := append(minimalPDF("Huawei NE8000 VRP v8 Manual"),
		[]byte("\n<< /Type /Annot /Subtype /Link /A << /S /URI /URI (https://support.huawei.com/hedex/ne8000-vrpv8.pdf) >> >>\n")...)
	links := ExtractPDFAnnotationLinks(rawPDF)
	if len(links) != 1 || links[0] != "https://support.huawei.com/hedex/ne8000-vrpv8.pdf" {
		t.Fatalf("esperava extrair link de anotação /URI do PDF, obteve: %v", links)
	}
	passages := MakePassages("Linha 1 de configuração\nLinha 2 de verificação", nil, 200)
	if len(passages) != 1 || passages[0].EndOffset <= passages[0].StartOffset {
		t.Fatalf("esperava offsets válidos no trecho, obteve: %+v", passages)
	}
}
