package formats

import "bytes"

var formats = make(map[string]IFormat)

func Obtain(format string) IFormat {
	if format == "" {
		return defaultEM3U
	}
	return formats[format]
}

func Register(format IFormat) {
	formats[format.Name()] = format
}

type PlaylistImport struct {
	Channels []ChannelImport
}

type ChannelImport struct {
	Name string
	URL  string
}

type PlaylistExport struct {
	EPGs          []string
	ChannelGroups []ChannelGroup
}

type ChannelGroup struct {
	GroupName string
	Channels  []ChannelExport
}

type ChannelExport struct {
	ShowName      string
	EPGName       string
	Logo          string
	SortedSources []Source
}

type Source struct {
	URL string

	AudioCodecName  string
	AudioChannels   int
	AudioSampleRate string

	VideoCodecName string
	VideoWidth     int
	VideoHeight    int
	VideoFrameRate float64
}

type IFormat interface {
	Name() string
	Ext() string
	Decode(b *bytes.Buffer) (pi PlaylistImport, err error)
	Encode(pe PlaylistExport) (b *bytes.Buffer, err error)
}
