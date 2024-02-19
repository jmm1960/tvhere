package formats

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

func init() {
	Register(&TXT{})
}

type TXT struct {
}

func (TXT) Name() string {
	return "txt"
}

func (TXT) Ext() string {
	return ".txt"
}

func (TXT) Decode(buf *bytes.Buffer) (pi PlaylistImport, err error) {
	var eof bool
	var line string
	for !eof {
		if line, err = buf.ReadString('\n'); err == io.EOF {
			eof = true
		} else if err != nil {
			break
		}
		line = strings.TrimSuffix(strings.TrimSpace(line), "\n")
		chname, link, found := strings.Cut(line, ",")
		if !found || len(chname) == 0 || len(link) == 0 || !strings.Contains(link, "://") {
			continue
		}
		pi.Channels = append(pi.Channels, ChannelImport{
			Name: chname,
			URL:  link,
		})
	}
	return pi, nil
}

func (TXT) Encode(pe PlaylistExport) (buf *bytes.Buffer, err error) {
	buf = new(bytes.Buffer)
	for _, cg := range pe.ChannelGroups {
		buf.WriteString(cg.GroupName + ",#genre#")
		buf.WriteByte('\n')
		for _, c := range cg.Channels {
			for _, s := range c.SortedSources {
				buf.WriteString(fmt.Sprintf("%s,%s\n", c.ShowName, s.URL))
			}
		}
		buf.WriteByte('\n')
	}
	return
}
