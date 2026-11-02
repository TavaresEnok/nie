package benchmark

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed 20_perguntas_telecom.json
var embeddedBenchmarkData []byte

type Question struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Query            string   `json:"query"`
	Topic            string   `json:"topic"`
	Vendors          []string `json:"vendors"`
	OS               string   `json:"os"`
	Models           []string `json:"models"`
	ExpectedCommands []string `json:"expected_commands"`
	Pitfalls         string   `json:"pitfalls"`
	Seeds            []string `json:"seeds"`
}

type Dataset struct {
	Questions []Question
}

func LoadEmbedded() (*Dataset, error) {
	var list []Question
	if err := json.Unmarshal(embeddedBenchmarkData, &list); err != nil {
		return nil, fmt.Errorf("decodificar dataset embutido: %w", err)
	}
	return &Dataset{Questions: list}, nil
}

func (d *Dataset) EvaluatePassage(q Question, text string) float64 {
	if len(q.ExpectedCommands) == 0 {
		return 0
	}
	matched := 0
	lowerText := strings.ToLower(text)
	for _, cmd := range q.ExpectedCommands {
		if strings.Contains(lowerText, strings.ToLower(cmd)) {
			matched++
		}
	}
	return float64(matched) / float64(len(q.ExpectedCommands))
}
