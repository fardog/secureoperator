package secureoperator

import "testing"

func TestRandSeq(t *testing.T) {
	var s string

	s = randSeq(10)

	if len(s) != 10 {
		t.Errorf("expected string length 10, got %v", len(s))
	}

	s = randSeq(20)

	if len(s) != 20 {
		t.Errorf("expected strig length 20, got %v", len(s))
	}
}
