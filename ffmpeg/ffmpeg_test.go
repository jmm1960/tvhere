package ffmpeg

import (
	"fmt"
	"testing"
)

func TestParseStreamInfo(t *testing.T) {
	testsLiveUrls := []string{
		"http://ebsonairios.ebs.co.kr/plus3tablet500k/tablet500k/plus3tablet500k.index.m3u8?zshijd",
		"http://[2409:8087:3869:8021:1001::e5]:6610/PLTV/88888910/224/3221225642/index.m3u8",
	}
	for i, url := range testsLiveUrls {
		steam, _, err := ProbeStreamInfo(url)
		if err != nil {
			t.Fatal(err)
		}
		for _, stream := range steam.Streams {
			switch stream.CodecType {
			case CodecAudio:
				t.Log(i, stream.Index, stream.CodecType, stream.CodecName, stream.SampleRate, stream.Channels)
			case CodecVideo:
				t.Log(i, stream.Index, stream.CodecType, stream.CodecName, fmt.Sprintf("%dx%d", stream.Width, stream.Height), stream.DisplayAspectRatio, stream.AvgFrameRate)
			default:
				t.Log(i, stream.Index, stream.CodecType, stream.CodecName)
			}
		}
	}

}
