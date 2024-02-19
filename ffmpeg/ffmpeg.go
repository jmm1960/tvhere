package ffmpeg

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"time"
)

func ProbeStreamInfo(link string) (steam ProbeStream, cost time.Duration, err error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), time.Second*20)
	defer cancelFunc()
	cmd := exec.CommandContext(timeout, "ffprobe", "-v", "quiet", "-print_format", "json", "-show_streams", "-i", link)
	now := time.Now()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return steam, cost, err
	}
	cost = time.Now().Sub(now)
	err = json.Unmarshal(output, &steam)
	if err != nil {
		return steam, cost, err
	}
	if len(steam.Streams) == 0 {
		return steam, cost, errors.New("invalid stream")
	}
	return steam, cost, nil
}

type ProbeStream struct {
	Streams []StreamItem `json:"streams,omitempty"`
}
type Codec string

const (
	CodecAudio      Codec = "audio"
	CodecVideo            = "video"
	CodecSubtitle         = "subtitle"
	CodecData             = "data"
	CodecAttachment       = "attachment"
)

// StreamItem 只是合并了 audio video data 三种 codec 类型，还有 subtitle, attachment
type StreamItem struct {
	Index              int    `json:"index"`
	CodecName          string `json:"codec_name"`
	CodecLongName      string `json:"codec_long_name"`
	Profile            string `json:"profile"`
	CodecType          Codec  `json:"codec_type"`
	CodecTagString     string `json:"codec_tag_string"`
	CodecTag           string `json:"codec_tag"`
	Width              int    `json:"width,omitempty"`
	Height             int    `json:"height,omitempty"`
	CodedWidth         int    `json:"coded_width,omitempty"`
	CodedHeight        int    `json:"coded_height,omitempty"`
	ClosedCaptions     int    `json:"closed_captions,omitempty"`
	FilmGrain          int    `json:"film_grain,omitempty"`
	HasBFrames         int    `json:"has_b_frames,omitempty"`
	SampleAspectRatio  string `json:"sample_aspect_ratio,omitempty"`
	DisplayAspectRatio string `json:"display_aspect_ratio,omitempty"`
	PixFmt             string `json:"pix_fmt,omitempty"`
	Level              int    `json:"level,omitempty"`
	ColorRange         string `json:"color_range,omitempty"`
	ColorSpace         string `json:"color_space,omitempty"`
	ColorTransfer      string `json:"color_transfer,omitempty"`
	ColorPrimaries     string `json:"color_primaries,omitempty"`
	ChromaLocation     string `json:"chroma_location,omitempty"`
	Refs               int    `json:"refs,omitempty"`
	IsAvc              string `json:"is_avc,omitempty"`
	NalLengthSize      string `json:"nal_length_size,omitempty"`
	RFrameRate         string `json:"r_frame_rate"`
	AvgFrameRate       string `json:"avg_frame_rate"`
	TimeBase           string `json:"time_base"`
	StartPts           int64  `json:"start_pts"`
	StartTime          string `json:"start_time"`
	BitsPerRawSample   string `json:"bits_per_raw_sample,omitempty"`
	ExtradataSize      int    `json:"extradata_size,omitempty"`
	Disposition        struct {
		Default         int `json:"default"`
		Dub             int `json:"dub"`
		Original        int `json:"original"`
		Comment         int `json:"comment"`
		Lyrics          int `json:"lyrics"`
		Karaoke         int `json:"karaoke"`
		Forced          int `json:"forced"`
		HearingImpaired int `json:"hearing_impaired"`
		VisualImpaired  int `json:"visual_impaired"`
		CleanEffects    int `json:"clean_effects"`
		AttachedPic     int `json:"attached_pic"`
		TimedThumbnails int `json:"timed_thumbnails"`
		Captions        int `json:"captions"`
		Descriptions    int `json:"descriptions"`
		Metadata        int `json:"metadata"`
		Dependent       int `json:"dependent"`
		StillImage      int `json:"still_image"`
	} `json:"disposition"`
	Tags struct {
		VariantBitrate string `json:"variant_bitrate"`
	} `json:"tags"`
	SampleFmt     string `json:"sample_fmt,omitempty"`
	SampleRate    string `json:"sample_rate,omitempty"`
	Channels      int    `json:"channels,omitempty"`
	ChannelLayout string `json:"channel_layout,omitempty"`
	BitsPerSample int    `json:"bits_per_sample,omitempty"`
}

// IsHDR https://video.stackexchange.com/a/28715
func (s StreamItem) IsHDR() bool {
	return s.CodecType == "video" && s.ColorSpace == "bt2020nc" && s.ColorTransfer == "smpte2084" && s.ColorPrimaries == "bt2020"
}
