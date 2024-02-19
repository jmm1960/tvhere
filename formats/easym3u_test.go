package formats

import (
	"bytes"
	"testing"
)

func TestSimpleM3U_Decode(t *testing.T) {
	u := EasyM3U{}

	u.Decode(bytes.NewBufferString(`#EXTM3U
#EXTINF:-1 tvg-id="CCTV1" tvg-name="CCTV1" tvg-logo="https://epg.112114.xyz/logo/CCTV1.png" group-title="央视",CCTV-1 综合
http://192.168.31.4:5678/sxg.php?id=CCTV-1H265_4000
#EXTINF:-1 tvg-id="CCTV2" tvg-name="CCTV2" tvg-logo="https://epg.112114.xyz/logo/CCTV2.png" group-title="央视",CCTV-2 财经
http://192.168.31.4:5678/sxg.php?id=CCTV-2H265_4000`))
}
