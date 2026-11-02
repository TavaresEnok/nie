package extract

import (
	"context"
	"errors"
	"os/exec"
	"strings"
)

var ErrOCRUnavailable = errors.New("OCR não disponível no ambiente")

// OCRService provides optional external OCR via Tesseract or pdftotext CLI when available.
type OCRService struct {
	available bool
	cmdPath   string
}

func NewOCRService() *OCRService {
	path, err := exec.LookPath("tesseract")
	if err != nil {
		path, err = exec.LookPath("pdftotext")
	}
	return &OCRService{
		available: err == nil,
		cmdPath:   path,
	}
}

func (o *OCRService) IsAvailable() bool {
	return o.available
}

// ExtractTextWithOCR runs OCR on an image or scanned document when installed in the container/host.
func (o *OCRService) ExtractText(ctx context.Context, inputPath string) (string, error) {
	if !o.available {
		return "", ErrOCRUnavailable
	}
	cmd := exec.CommandContext(ctx, o.cmdPath, inputPath, "stdout")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
