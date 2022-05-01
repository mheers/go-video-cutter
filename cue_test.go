package cutter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCueSheet(t *testing.T) {
	sheet, err := GetCueSheetFromFile("demofiles/concert.cue")
	require.NoError(t, err)
	require.NotNil(t, sheet)
}

func TestCutByCueSheet(t *testing.T) {
	sheet, err := GetCueSheetFromFile("demofiles/concert.cue")
	require.NoError(t, err)
	err = CutByCueSheet(sheet, "demofiles/concert.mp4", "demofiles/output/", "mp4")
	require.NoError(t, err)
}
