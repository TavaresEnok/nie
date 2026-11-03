package store

import "testing"

func TestVectorLiteral(t *testing.T) {
	got := vectorLiteral([]float32{1, 0.5, -2})
	if got != "[1,0.5,-2]" {
		t.Fatal(got)
	}
}
