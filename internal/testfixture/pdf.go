// Package testfixture contains synthetic documents used only by integration tests.
package testfixture

import _ "embed"

//go:embed scanned.pdf
var ScannedPDF []byte
