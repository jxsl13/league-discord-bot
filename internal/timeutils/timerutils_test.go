package timeutils

import (
	"testing"
	"time"
)

func TestCeil(t *testing.T) {
	input1 := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	t1 := Ceil(input1, time.Minute)
	expected1 := time.Date(2023, 10, 1, 12, 1, 0, 0, time.UTC)
	if t1 != expected1 {
		t.Errorf("Ceil(%v) = %v; want %v", input1, t1, expected1)
	}

	input2 := time.Date(2023, 10, 1, 12, 0, 30, 0, time.UTC)
	t2 := Ceil(input2, time.Minute)
	expected2 := time.Date(2023, 10, 1, 12, 1, 0, 0, time.UTC)
	if t2 != expected2 {
		t.Errorf("Ceil(%v) = %v; want %v", input2, t2, expected2)
	}

}
