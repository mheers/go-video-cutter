package cutter

import (
	"fmt"
	"log"

	"github.com/3d0c/gmf"
	"github.com/sirupsen/logrus"
)

// Cut cuts a video file from start to end (in seconds). If end is -1, the end of the video is used.
func Cut(input string, start, end int64, output string) error {
	// input
	inputCtx, err := gmf.NewInputCtx(input)
	if err != nil {
		return fmt.Errorf("could not open input context: %s", err)
	}
	defer inputCtx.Free()
	inputCtx.Dump()

	videoInputStream, err := inputCtx.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if err != nil {
		return fmt.Errorf("failed to find video stream")
	}

	tbIn := videoInputStream.TimeBase()
	tbInAVR := tbIn.AVR()
	logrus.Print("Timebase: ", tbInAVR.Num, "/", tbInAVR.Den)

	// get the first frame of the video
	firstPacket, err := inputCtx.GetFirstPacketForStreamIndex(videoInputStream.Index())
	if err != nil {
		return fmt.Errorf("failed to get next packet: %s", err)
	}
	defer firstPacket.Free()
	firstPts := firstPacket.Pts()
	logrus.Print("first position: ", firstPts)

	// get the last frame of the video
	lastPacket, err := inputCtx.GetLastPacketForStreamIndex(videoInputStream.Index())
	if err != nil {
		return fmt.Errorf("failed to get next packet: %s", err)
	}
	defer lastPacket.Free()
	lastPts := lastPacket.Pts()
	logrus.Print("last position: ", lastPts)

	if end == -1 {
		end = int64(inputCtx.Duration())
	}

	// get the last frame we want to have of the video
	err = inputCtx.SeekFrameAt(end, videoInputStream.Index())
	if err != nil {
		return fmt.Errorf("failed to seek frame: %s", err)
	}
	endPacket, err := inputCtx.GetNextPacketForStreamIndex(videoInputStream.Index())
	if err != nil {
		return fmt.Errorf("failed to get next packet: %s", err)
	}
	defer endPacket.Free()
	endPts := endPacket.Pts()
	logrus.Print("End position: ", endPts)
	endSec := endPacket.Time(tbIn)
	logrus.Print("End sec: ", endSec)

	// go to the start where we want the video from
	err = inputCtx.SeekFrameAt(start, videoInputStream.Index())
	if err != nil {
		return fmt.Errorf("failed to seek frame: %s", err)
	}
	startPacket, err := inputCtx.GetNextPacketForStreamIndex(videoInputStream.Index())
	if err != nil {
		return fmt.Errorf("failed to get next packet: %s", err)
	}
	defer startPacket.Free()
	startPts := startPacket.Pts()
	logrus.Print("Start position: ", startPts)
	startSec := startPacket.Time(tbIn)
	logrus.Print("Start sec: ", startSec)

	// create new output
	outputCtx, err := gmf.NewOutputCtx(output)
	if err != nil {
		return fmt.Errorf("could not open output context: %s", err)
	}
	defer outputCtx.Free()

	videoInputStreamCodecCtx := videoInputStream.CodecCtx()

	videoOutputStreamCodec, err := gmf.FindEncoder(videoInputStreamCodecCtx.Codec().Id())
	if err != nil {
		return fmt.Errorf("could not find encoder: %s", err)
	}
	videoOutputStreamCodecCtx := gmf.NewCodecCtx(videoOutputStreamCodec)
	videoOutputStream, err := outputCtx.AddStreamWithCodeCtx(videoInputStreamCodecCtx)
	if err != nil {
		return fmt.Errorf("could not add video stream: %s", err)
	}
	defer videoOutputStream.Free()

	if outputCtx.IsGlobalHeader() {
		videoOutputStreamCodecCtx.SetFlag(gmf.CODEC_FLAG_GLOBAL_HEADER)
	}

	// write output header
	if err := outputCtx.WriteHeader(); err != nil {
		log.Fatal("new output fail: ", err.Error())
		return err
	}
	outputCtx.Dump()

	packetsFromKeyframeToStart, err := getPacketsFromKeyframeToStart(inputCtx, videoInputStream, start)
	if err != nil {
		return fmt.Errorf("failed to get packets from keyframe to start: %s", err)
	}

	// seek to the start
	err = inputCtx.SeekFrameAt(start, videoInputStream.Index())
	if err != nil {
		return fmt.Errorf("failed to seek frame: %s", err)
	}

	var (
		packets int64 = 0 - int64(len(packetsFromKeyframeToStart))
	)
	for {
		packet, err := inputCtx.GetNextPacketForStreamIndex(videoInputStream.Index())
		if err != nil {
			return fmt.Errorf("failed to get next packet: %s", err)
		}

		// stop if we reached the end
		if packet.Pts() == endPacket.Pts() {
			break
		}

		writePacket := packet.Clone()
		packet.Free()

		// TODO: this is a hack to get the correct timebase
		outputTBOrig := videoOutputStream.TimeBase()
		outputTB := &gmf.AVR{Num: outputTBOrig.AVR().Num, Den: outputTBOrig.AVR().Den * 2}
		newPts := gmf.RescaleQ(packets, videoInputStreamCodecCtx.TimeBase(), outputTB.AVRational())

		writePacket = writePacket.SetPts(newPts)
		writePacket = writePacket.SetDts(newPts)

		// TODO: maybe this is a better approach to get the correct timebase?
		// gmf.RescaleTs(writePacket, videoOutputStream.CodecCtx().TimeBase(), videoOutputStream.TimeBase())
		writePacket = writePacket.SetStreamIndex(0)

		err = outputCtx.WritePacket(writePacket)
		if err != nil {
			return fmt.Errorf("failed to write packet: %s", err)
		}
		writePacket.Free()

		packets++
	}

	outputCtx.WriteTrailer()

	return nil
}

func getPacketsFromKeyframeToStart(inputCtx *gmf.FmtCtx, videoInputStream *gmf.Stream, startSec int64) ([]*gmf.Packet, error) {
	// seek to the start
	err := inputCtx.SeekFrameAt(startSec, videoInputStream.Index())
	if err != nil {
		return nil, fmt.Errorf("failed to seek frame: %s", err)
	}

	var packets []*gmf.Packet
	for {
		packet, err := inputCtx.GetNextPacketForStreamIndex(videoInputStream.Index())
		if err != nil {
			return nil, fmt.Errorf("failed to get next packet: %s", err)
		}

		currentSec := packet.Time(videoInputStream.TimeBase())
		if int64(currentSec) >= startSec {
			break
		}
		packets = append(packets, packet)

	}

	return packets, nil
}

// CutByTimeCode cuts a video file from start to end as timecode (e.g. "20:30:00:00")
func CutByTimeCode(input string, start, end string, output string) error {
	inputCtx, err := gmf.NewInputCtx(input)
	if err != nil {
		return fmt.Errorf("could not open input context: %s", err)
	}
	defer inputCtx.Free()

	videoInputStream, err := inputCtx.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if err != nil {
		return fmt.Errorf("failed to find video stream")
	}
	timebase := videoInputStream.TimeBase()

	err = inputCtx.SeekFrameAtTimeCode(start, videoInputStream.Index())
	if err != nil {
		return fmt.Errorf("failed to seek frame: %s", err)
	}

	startPacket, err := inputCtx.GetNextPacketForStreamIndex(videoInputStream.Index())
	if err != nil {
		return fmt.Errorf("failed to get next packet: %s", err)
	}
	defer startPacket.Free()
	startSec := startPacket.Time(timebase)

	err = inputCtx.SeekFrameAtTimeCode(end, videoInputStream.Index())
	if err != nil {
		return fmt.Errorf("failed to seek frame: %s", err)
	}

	endPacket, err := inputCtx.GetNextPacketForStreamIndex(videoInputStream.Index())
	if err != nil {
		return fmt.Errorf("failed to get next packet: %s", err)
	}
	defer endPacket.Free()
	endSec := endPacket.Time(timebase)

	return Cut(input, int64(startSec), int64(endSec), output)
}
