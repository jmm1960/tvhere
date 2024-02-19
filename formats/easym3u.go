package formats

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

var defaultEM3U = &EasyM3U{}

func init() {
	Register(defaultEM3U)
}

type EasyM3U struct {
}

func (s EasyM3U) Name() string {
	return "em3u"
}

func (EasyM3U) Ext() string {
	return ".m3u"
}

func (s EasyM3U) Decode(buf *bytes.Buffer) (pi PlaylistImport, err error) {
	var eof bool
	var line string
	var pendingChannelName string
	for !eof {
		if line, err = buf.ReadString('\n'); err == io.EOF {
			eof = true
		} else if err != nil {
			break
		}
		line = strings.TrimSuffix(strings.TrimSpace(line), "\n")
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#EXTINF:") {
			index := strings.LastIndex(line, ",")
			pendingChannelName = strings.TrimSpace(line[index+1:])
		} else if strings.HasPrefix(line, "#") {
			continue
		} else {
			if pendingChannelName != "" && strings.Contains(line, "://") {
				pi.Channels = append(pi.Channels, ChannelImport{
					Name: pendingChannelName,
					URL:  line,
				})
				pendingChannelName = ""
			}
		}
	}
	return pi, nil
}

func (s EasyM3U) Encode(pe PlaylistExport) (buf *bytes.Buffer, err error) {
	buf = new(bytes.Buffer)
	buf.WriteString("#EXTM3U")
	if len(pe.EPGs) > 0 { // url-tvg= x-tvg-url=
		buf.WriteString(fmt.Sprintf(` url-tvg="%s"`, strings.Join(pe.EPGs, ",")))
		buf.WriteString(fmt.Sprintf(` x-tvg-url="%s"`, strings.Join(pe.EPGs, ",")))
	}
	buf.WriteByte('\n')
	for _, cg := range pe.ChannelGroups {
		for _, channel := range cg.Channels {
			buf.WriteString("#EXTINF:-1")
			if channel.EPGName != "" {
				buf.WriteString(fmt.Sprintf(` tvg-id="%s"`, channel.EPGName))
				buf.WriteString(fmt.Sprintf(` tvg-name="%s"`, channel.EPGName))
				buf.WriteString(fmt.Sprintf(` group-title="%s"`, cg.GroupName))
			}
			if channel.Logo != "" {
				buf.WriteString(fmt.Sprintf(` tvg-logo="%s"`, channel.Logo))
			}

			buf.WriteString(fmt.Sprintf(",%s\n", channel.ShowName))
			buf.WriteString(channel.SortedSources[0].URL)
			buf.WriteByte('\n')
		}
		buf.WriteByte('\n')
	}
	return
}
