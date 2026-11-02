package extract

import (
	"context"
	"errors"
	"os/exec"
	"strings"
)

var ErrOCRUnavailable = errors.New("OCR não disponível no ambiente")

// OCRService provides optional external OCR via Tesseract or pdftotext CLI when available.
// Ainda não é chamado pelo pipeline de coleta (ver README: OCR planejado).
type OCRService struct {
	available bool
	cmdPath   string
	tool      string
}

func NewOCRService() *OCRService {
	tool := "tesseract"
	path, err := exec.LookPath(tool)
	if err != nil {
		tool = "pdftotext"
		path, err = exec.LookPath(tool)
	}
	return &OCRService{
		available: err == nil,
		cmdPath:   path,
		tool:      tool,
	}
}

// ocrArgs devolve os argumentos para escrever o texto em stdout: o Tesseract
// usa a palavra "stdout"; o pdftotext usa "-" (com "stdout" criaria um arquivo).
func ocrArgs(tool, inputPath string) []string {
	if tool == "pdftotext" {
		return []string{"-layout", inputPath, "-"}
	}
	return []string{inputPath, "stdout"}
}

func (o *OCRService) IsAvailable() bool {
	return o.available
}

// ExtractTextWithOCR runs OCR on an image or scanned document when installed in the container/host.
func (o *OCRService) ExtractText(ctx context.Context, inputPath string) (string, error) {
	if !o.available {
		return "", ErrOCRUnavailable
	}
	cmd := exec.CommandContext(ctx, o.cmdPath, ocrArgs(o.tool, inputPath)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
