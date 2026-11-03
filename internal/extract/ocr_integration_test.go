package extract

import (
	"context"
	"nie/internal/testfixture"
	"os"
	"strings"
	"testing"
	"time"
)

func TestScannedPDFRealOCRIntegration(t *testing.T) {
	if os.Getenv("NIE_OCR_INTEGRATION") != "true" {
		t.Skip("defina NIE_OCR_INTEGRATION=true com Poppler e Tesseract instalados")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	service := NewOCRServiceWithConfig(OCRConfig{MaxPages: 2, Language: "eng", Timeout: 20 * time.Second})
	if !service.CanOCRPDF() {
		t.Fatal("OCR PDF indisponível no ambiente de validação")
	}
	pdf := testfixture.ScannedPDF
	if native, err := PDF(pdf); err == nil && strings.TrimSpace(native.Text) != "" {
		t.Fatal("fixture deveria ser um PDF sem camada textual")
	}
	result, err := service.ExtractBytes(ctx, pdf, "application/pdf")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("texto reconhecido: %q", result.Text)
	if !strings.Contains(strings.ToUpper(result.Text), "BGP NETWORK ENGINE") {
		t.Fatalf("OCR não recuperou texto da imagem: %q", result.Text)
	}
	if len(result.Passages) == 0 || result.Passages[0].Page == nil || *result.Passages[0].Page != 1 || result.ParserVersion == "" {
		t.Fatalf("proveniência OCR incompleta: %+v", result)
	}
	cancelled, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	if _, err := service.ExtractBytes(cancelled, pdf, "application/pdf"); err == nil {
		t.Fatal("OCR ignorou cancelamento")
	}
}
