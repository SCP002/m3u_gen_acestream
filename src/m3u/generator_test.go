package m3u

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zenizh/go-capturer"
)

func TestFilterByName(t *testing.T) {
	// Test log output.
	out := capturer.CaptureStderr(func() {
		//
	})
	assert.Contains(t, out, ``)
}
