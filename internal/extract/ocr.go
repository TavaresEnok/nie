package extract

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ErrOCRUnavailable = errors.New("OCR não disponível no ambiente")

// OCRService provides optional external OCR via Tesseract or pdftotext CLI when available.
type OCRService struct {
	available bool
	cmdPath   string
	tool      string
	pdfToPPM  string
	pdfInfo   string
	cfg       OCRConfig
}

type OCRConfig struct {
	MaxPages     int
	MaxDimension int
	Language     string
	Timeout      time.Duration
}

func NewOCRService() *OCRService {
	return NewOCRServiceWithConfig(OCRConfig{})
}

func NewOCRServiceWithConfig(cfg OCRConfig) *OCRService {
	if cfg.MaxPages <= 0 {
		cfg.MaxPages = 10
	}
	cfg.MaxPages = min(cfg.MaxPages, 50)
	if cfg.MaxDimension <= 0 {
		cfg.MaxDimension = 1800
	}
	cfg.MaxDimension = max(300, min(cfg.MaxDimension, 3000))
	if cfg.Language == "" {
		cfg.Language = "eng"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}
	tool := "tesseract"
	path, err := exec.LookPath(tool)
	if err != nil {
		tool = "pdftotext"
		path, err = exec.LookPath(tool)
	}
	pdfToPPM, _ := exec.LookPath("pdftoppm")
	pdfInfo, _ := exec.LookPath("pdfinfo")
	return &OCRService{
		available: err == nil,
		cmdPath:   path,
		tool:      tool,
		pdfToPPM:  pdfToPPM, pdfInfo: pdfInfo, cfg: cfg,
	}
}

func (o *OCRService) CanOCRPDF() bool {
	return o != nil && o.available && o.tool == "tesseract" && o.pdfToPPM != "" && o.pdfInfo != ""
}

func (o *OCRService) Settings() OCRConfig { return o.cfg }

type outputLimit struct {
	data bytes.Buffer
	max  int
}

func (b *outputLimit) Write(p []byte) (int, error) {
	if b.data.Len()+len(p) > b.max {
		return 0, errors.New("saída de extrator excedeu o limite")
	}
	return b.data.Write(p)
}

func runExtractor(ctx context.Context, path string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "OMP_THREAD_LIMIT=1")
	out := &outputLimit{max: MaxTextChars}
	cmd.Stdout = out
	// stderr is deliberately discarded; PDF metadata is untrusted and may
	// contain excessive diagnostics. Failures retain their exit status.
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return out.data.String(), nil
}

func (o *OCRService) Tool() string {
	if o == nil || !o.available {
		return ""
	}
	return o.tool
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
	return o != nil && o.available
}

// ExtractText runs OCR on an image or scanned document when installed in the container/host.
func (o *OCRService) ExtractText(ctx context.Context, inputPath string) (string, error) {
	if !o.IsAvailable() {
		return "", ErrOCRUnavailable
	}
	args := ocrArgs(o.tool, inputPath)
	if o.tool == "tesseract" {
		args = append(args, "-l", o.cfg.Language, "--psm", "6")
	}
	out, err := runExtractor(ctx, o.cmdPath, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ExtractBytes grava temporariamente os bytes recebidos, executa OCR/pdftotext e devolve um Result versionado.
func (o *OCRService) ExtractBytes(ctx context.Context, body []byte, mime string) (Result, error) {
	if !o.IsAvailable() {
		return Result{}, ErrOCRUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, o.cfg.Timeout)
	defer cancel()
	f, err := os.CreateTemp("", "nie-ocr-*")
	if err != nil {
		return Result{}, err
	}
	tmpPath := f.Name()
	defer os.Remove(tmpPath)
	if _, err := f.Write(body); err != nil {
		_ = f.Close()
		return Result{}, err
	}
	_ = f.Close()

	if strings.SplitN(mime, ";", 2)[0] == "application/pdf" && o.tool == "tesseract" {
		return o.extractPDF(ctx, tmpPath, body)
	}
	rawText, err := o.ExtractText(ctx, tmpPath)
	if err != nil {
		return Result{}, err
	}
	clean := NormalizeText(rawText)
	if clean == "" {
		return Result{}, errors.New("OCR não produziu texto extraível")
	}
	if len(clean) > MaxTextChars {
		clean = clean[:MaxTextChars]
	}
	if mime == "" {
		mime = "application/pdf"
	}
	res := Result{
		Title:         firstLine(clean),
		Text:          clean,
		MIME:          mime,
		Parser:        "ocr-" + o.tool,
		ParserVersion: o.version(ctx),
		Passages:      MakePassages(clean, nil, DefaultPassageChars),
		Links:         ExtractURLLinks(clean),
	}
	for _, l := range ExtractPDFAnnotationLinks(body) {
		res.Links = append(res.Links, l)
	}
	return res, nil
}

func (o *OCRService) version(ctx context.Context) string {
	version, err := runExtractor(ctx, o.cmdPath, "--version")
	if err != nil {
		return "unknown; adapter-2.0.0"
	}
	return firstLine(version) + "; adapter-2.0.0"
}

var pdfPageCount = regexp.MustCompile(`(?m)^Pages:\s+(\d+)\s*$`)

func (o *OCRService) extractPDF(ctx context.Context, input string, body []byte) (Result, error) {
	if !o.CanOCRPDF() {
		return Result{}, ErrOCRUnavailable
	}
	info, err := runExtractor(ctx, o.pdfInfo, input)
	if err != nil {
		return Result{}, err
	}
	match := pdfPageCount.FindStringSubmatch(info)
	if len(match) != 2 {
		return Result{}, errors.New("não foi possível determinar as páginas do PDF")
	}
	pages, err := strconv.Atoi(match[1])
	if err != nil || pages < 1 || pages > o.cfg.MaxPages {
		return Result{}, fmt.Errorf("PDF excede limite de OCR de %d páginas", o.cfg.MaxPages)
	}
	dir, err := os.MkdirTemp("", "nie-ocr-pages-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(dir)
	result := Result{MIME: "application/pdf", Parser: "tesseract-poppler", ParserVersion: o.version(ctx)}
	var all strings.Builder
	for page := 1; page <= pages; page++ {
		prefix := filepath.Join(dir, fmt.Sprintf("page-%d", page))
		if _, err := runExtractor(ctx, o.pdfToPPM, "-f", strconv.Itoa(page), "-l", strconv.Itoa(page), "-singlefile", "-scale-to", strconv.Itoa(o.cfg.MaxDimension), "-png", input, prefix); err != nil {
			return Result{}, err
		}
		image := prefix + ".png"
		text, err := o.ExtractText(ctx, image)
		_ = os.Remove(image)
		if err != nil {
			return Result{}, err
		}
		text = NormalizeText(text)
		if text == "" {
			continue
		}
		if all.Len()+len(text)+2 > MaxTextChars {
			return Result{}, errors.New("texto OCR excede limite")
		}
		base := all.Len()
		pn := page
		passages := MakePassages(text, &pn, DefaultPassageChars)
		for i := range passages {
			passages[i].Ordinal = len(result.Passages) + i
			passages[i].StartOffset += base
			passages[i].EndOffset += base
		}
		result.Passages = append(result.Passages, passages...)
		all.WriteString(text)
		all.WriteString("\n\n")
	}
	result.Text = strings.TrimSpace(all.String())
	if result.Text == "" {
		return Result{}, errors.New("OCR não produziu texto extraível")
	}
	result.Title = firstLine(result.Text)
	result.Links = append(ExtractURLLinks(result.Text), ExtractPDFAnnotationLinks(body)...)
	return result, nil
}
