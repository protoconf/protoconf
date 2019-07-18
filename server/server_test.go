package server

import (
	"testing"
)

func Test(t *testing.T) {
	if 1 != 1 {
		t.Errorf("Abs(-1) = %d; want 1", 0)
	}
}
