package cutter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCut(t *testing.T) {
	err := Cut("demofiles/003CV7.mp4", 55*60, (55*60)+(2*60*60), "cutted.mp4")
	assert.NoError(t, err)
}

func TestCutByTimeCode(t *testing.T) {
	err := CutByTimeCode("demofiles/demo.mp4", "22:44:00:00", "22:44:50:00", "cutted-by-timecode.mp4")
	assert.NoError(t, err)
}
