package cutter

import (
	"fmt"
	"os"

	"github.com/vchimishuk/chub/cue"
)

func GetCueSheetFromFile(filePath string) (*cue.Sheet, error) {
	return cue.ParseFile(filePath)
}

func CutByCueSheet(sheet *cue.Sheet, inputFile, outputFolder, format string) error {
	// mkdir
	err := os.MkdirAll(outputFolder, os.ModePerm)
	if err != nil {
		return err
	}

	// cut
	for _, file := range sheet.Files {
		for t, track := range file.Tracks {
			for _, index := range track.Indexes {
				startSec := GetSecondsFromCueTrackIndex(index)
				endSec := int64(-1)
				// get start seconds from next track (if exists)
				if t+1 < len(file.Tracks) {
					endSec = GetSecondsFromCueTrackIndex(file.Tracks[t+1].Indexes[0])
				}
				err := Cut(inputFile, startSec, endSec, fmt.Sprintf("%s/%d_%s-%s.%s", outputFolder, track.Number, track.Performer, track.Title, format))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func GetSecondsFromCueTrackIndex(index *cue.Index) int64 {
	return int64(index.Time.Min*60) + int64(index.Time.Sec)
}
