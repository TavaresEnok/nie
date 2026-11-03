package library

import "testing"

func TestCompare(t *testing.T) {
	if Compare("", "abc") != ChangeFirst {
		t.Fatal("primeira captura")
	}
	if Compare("abc", "abc") != ChangeUnchanged {
		t.Fatal("mesmo hash")
	}
	if Compare("abc", "def") != ChangeUpdated {
		t.Fatal("hash novo")
	}
}
