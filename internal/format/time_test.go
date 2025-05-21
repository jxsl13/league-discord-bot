package format

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDuration(t *testing.T) {
	expected := "15 minutes"
	got := Duration(15 * time.Minute)
	assert.Equal(t, expected, got)
	expected = "1 hour"
	got = Duration(1 * time.Hour)
	assert.Equal(t, expected, got)

	expected = "15 minutes"
	got = Duration(14*time.Minute + 30*time.Second)
	assert.Equal(t, expected, got)
}
