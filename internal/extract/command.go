package extract

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// RunCommand grava a entrada num arquivo temporário e executa um parser externo.
// A saída padrão vira o texto. O processo é cancelável e a saída tem teto.
func RunCommand(ctx context.Context, parser, bin string, extraArgs []string, input []byte, timeout time.Duration) (Result, error) {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		return Result{}, fmt.Errorf("comando de extração vazio")
	}
	if timeout <= 0 || timeout > 2*time.Minute {
		timeout = 45 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	file, err := os.CreateTemp("", "nie-extract-*")
	if err != nil {
		return Result{}, err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err := file.Write(input); err != nil {
		file.Close()
		return Result{}, err
	}
	if err := file.Close(); err != nil {
		return Result{}, err
	}
	args := append(append([]string{}, extraArgs...), name)
	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return Result{}, fmt.Errorf("%s: %s", parser, msg)
	}
	text := stdout.String()
	if len(text) > MaxTextChars {
		text = text[:MaxTextChars]
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return Result{}, fmt.Errorf("%s não produziu texto", parser)
	}
	res := Result{Text: text, MIME: "text/plain", Parser: parser, ParserVersion: bin}
	res.Passages = passagesFromText(text)
	return res, nil
}

func passagesFromText(text string) []Passage {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	return []Passage{{Ordinal: 0, Text: text, EndOffset: len(text)}}
}
