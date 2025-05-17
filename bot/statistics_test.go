package bot

import (
	"testing"
	"time"
)

func TestUntilMidnight(t *testing.T) {
	now := time.Date(2025, 5, 17, 15, 30, 0, 0, time.UTC)
	expected := 8*time.Hour + 30*time.Minute

	result := untilMidnight(now)

	if result != expected {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}
