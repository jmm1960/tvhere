package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/go-creed/sat"
	"gopkg.in/yaml.v3"
	"heretv/config"
	"heretv/ffmpeg"
	"heretv/formats"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	flagFile = flag.String("f", "live.yml", "specify the live config file. default: live.yml")
	httpPort = flag.Int("p", 8090, "the port http server serve at")
)

func serveFolder(folder string, port int) {
	mutex := http.NewServeMux()
	mutex.Handle("/", http.FileServer(http.Dir(folder)))
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), mutex)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("serve %s at port %d\n", folder, port)
}

func main() {
	flag.Parse()
	fileOrFolderName := "live.yml"
	if *flagFile != "" {
		fileOrFolderName = *flagFile
	}
	stat, err := os.Stat(fileOrFolderName)
	if err != nil {
		panic(err)
	}
	if !stat.IsDir() {
		processConfigFile(fileOrFolderName)
	} else {
		go func() {
			for {
				err := filepath.WalkDir(fileOrFolderName, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if d.IsDir() || filepath.Ext(d.Name()) != ".yml" {
						return nil
					}
					processConfigFile(path)
					return nil
				})
				if err != nil {
					panic(err)
				}
				time.Sleep(time.Hour * 6)
			}
		}()
		serveFolder(fileOrFolderName, *httpPort)
	}
}

func processConfigFile(file string) {
	log.Printf("processing config %s\n", file)
	cfgFile, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	var liveCfg config.Config
	err = yaml.Unmarshal(cfgFile, &liveCfg)
	if err != nil {
		panic(err)
	}
	if err := liveCfg.Check(); err != nil {
		panic(err)
	}
	dir := filepath.Dir(file)
	// 1. 根据频道列表正则，筛选出感兴趣的源
	interestedSourceChannels := filterInterestedSourceChannels(dir, liveCfg)
	for channelName, channelLinks := range interestedSourceChannels {
		for link, source := range channelLinks {
			fmt.Println(channelName, source, link)
		}
	}
	// 2. 源检测，剔除无法使用的，剩余按照质量排序
	channelValidSourcesByQuality := detectAndRankingSourceChannels(interestedSourceChannels)
	for _, group := range liveCfg.Groups {
		channels := group.AllChannels(liveCfg.ChannelsByStation)
		for _, channel := range channels {
			validSources := channelValidSourcesByQuality[channel]
			if len(validSources) == 0 {
				log.Printf("CHSRC: %s Not Found\n", channel)
				continue
			}
			for idx, source := range validSources {
				if idx == 0 {
					log.Printf("CHSRC[✓]: %s", source.Print())
					continue
				}
				log.Printf("CHSRC: %s", source.Print())
			}
		}
	}
	// 3. 导出源
	exportPlaylist(liveCfg, channelValidSourcesByQuality, file[:len(file)-len(filepath.Ext(file))])
}

func exportPlaylist(liveCfg config.Config, channelValidSourcesByQuality map[string]SourceChannelQualityCompare, name string) {
	for _, f := range liveCfg.ExportFormats {
		format := formats.Obtain(f)
		if format == nil {
			log.Printf("format %s not exist skip export", f)
			continue
		}
		groups := make([]formats.ChannelGroup, 0)
		for _, group := range liveCfg.Groups {
			allChannels := make([]string, 0)
			for _, station := range group.Stations {
				allChannels = append(allChannels, liveCfg.ChannelsByStation[station]...)
			}
			allChannels = append(allChannels, group.Channels...)
			channelsExport := make([]formats.ChannelExport, 0)
			for _, channelName := range allChannels {
				sources := channelValidSourcesByQuality[channelName]
				if len(sources) == 0 {
					continue
				}
				sortedSources := make([]formats.Source, len(sources))
				for sIdx, source := range sources {
					sortedSources[sIdx] = formats.Source{
						URL:             source.Url,
						AudioCodecName:  source.AudioCodecName,
						AudioChannels:   source.AudioChannels,
						AudioSampleRate: source.AudioSampleRate,
						VideoCodecName:  source.VideoCodecName,
						VideoWidth:      source.VideoWidth,
						VideoHeight:     source.VideoHeight,
						VideoFrameRate:  source.VideoFrameRate,
					}
				}
				channelsExport = append(channelsExport, formats.ChannelExport{
					ShowName:      channelName,
					EPGName:       channelName, // TODO
					Logo:          "",          // TODO
					SortedSources: sortedSources,
				})
			}
			if len(channelsExport) > 0 {
				groups = append(groups, formats.ChannelGroup{
					GroupName: group.Name,
					Channels:  channelsExport,
				})
			}
		}
		playlistExport := formats.PlaylistExport{
			EPGs:          liveCfg.EPGs,
			ChannelGroups: groups,
		}
		buf, err := format.Encode(playlistExport)
		if err != nil {
			log.Printf("fail to export format %s: %v\n", format.Name(), err)
			continue
		}
		if err = os.WriteFile(name+format.Ext(), buf.Bytes(), os.ModePerm); err != nil {
			log.Printf("fail to write file %s: %v\n", format.Name(), err)
			continue
		}
	}
}

func detectAndRankingSourceChannels(interestedSourceChannels map[string]map[string]SourceChannelBasic) map[string]SourceChannelQualityCompare {
	basicCh := make(chan SourceChannelBasic)
	go func() {
		for channelName, channelLinks := range interestedSourceChannels {
			for link := range channelLinks {
				basicCh <- interestedSourceChannels[channelName][link]
			}
		}
		close(basicCh)
	}()
	scCh := make(chan SourceChannel)
	const routineCount = 10
	wg := sync.WaitGroup{}
	wg.Add(routineCount)
	for i := 0; i < routineCount; i++ {
		go func(idx int) {
			bk := false
			for {
				select {
				case basic, ok := <-basicCh:
					if !ok {
						bk = true
						break
					}
					sourceChannel := ffmpegCheck(basic)
					if sourceChannel != nil {
						scCh <- *sourceChannel
					}
				}
				if bk {
					break
				}
			}
			wg.Done()
			fmt.Println("routine", idx, "done")
		}(i)
	}
	go func() {
		wg.Wait()
		close(scCh)
	}()

	channelValidSourcesByQuality := make(map[string]SourceChannelQualityCompare)
	for sc := range scCh {
		if _, exist := channelValidSourcesByQuality[sc.Name]; !exist {
			channelValidSourcesByQuality[sc.Name] = make(SourceChannelQualityCompare, 0)
		}
		channelValidSourcesByQuality[sc.Name] = append(channelValidSourcesByQuality[sc.Name], sc)
	}
	fmt.Println("allroutine done")

	for s := range channelValidSourcesByQuality {
		sort.Sort(sort.Reverse(channelValidSourcesByQuality[s]))
	}
	return channelValidSourcesByQuality
}

func filterInterestedSourceChannels(dir string, cfg config.Config) map[string]map[string]SourceChannelBasic {
	s := sync.Map{}
	wg := sync.WaitGroup{}
	for i := range cfg.Sources {
		wg.Add(1)
		go func(sourceCfg config.ConfigSource) {
			defer wg.Done()
			log.Printf("parsing source %s", sourceCfg.Name)
			var bys []byte
			var err error
			if strings.HasPrefix(sourceCfg.URI, "http://") || strings.HasPrefix(sourceCfg.URI, "https://") {
				bys, err = RequestLink(sourceCfg.URI)
				if err != nil {
					log.Printf("fail to request source %s: %v\n", sourceCfg.Name, err)
					return
				}
			} else {
				filePath := sourceCfg.URI
				if !filepath.IsAbs(filePath) {
					filePath = filepath.Join(dir, sourceCfg.URI)
				}
				bys, err = os.ReadFile(filePath)
				if err != nil {
					log.Printf("fail to read local file %s: %v\n", sourceCfg.Name, err)
					return
				}
			}
			simple := t2s(string(bys))
			bys = []byte(simple)
			format := formats.Obtain(sourceCfg.Format)
			if format == nil {
				log.Printf("format %s not exist skip source %s\n", sourceCfg.Format, sourceCfg.Name)
				return
			}
			playlistImport, err := format.Decode(bytes.NewBuffer(bys))
			if err != nil {
				log.Printf("fail to decode %s using format %s: %v\n", sourceCfg.URI, format.Name(), err)
				return
			}
			basics := make([]SourceChannelBasic, len(playlistImport.Channels))
			for j, c := range playlistImport.Channels {
				basics[j] = SourceChannelBasic{
					NameInSource: c.Name,
					Url:          c.URL,
					Source:       sourceCfg.Name,
				}
			}
			var newBasics []SourceChannelBasic
			for i2, basic := range basics {
				matchedChannelIdx, isInterested := isChannelNameInterested(cfg.Channels, basic.NameInSource)
				if !isInterested {
					log.Printf("not interested in channel %s %s\n", sourceCfg.Name, basic.NameInSource)
					continue
				}
				matchName := cfg.Channels[matchedChannelIdx].Name
				log.Printf("channel match %s %s->%s\n", sourceCfg.Name, basic.NameInSource, matchName)
				basics[i2].Name = matchName
				newBasics = append(newBasics, basics[i2])
			}
			s.Store(sourceCfg.Name, newBasics)
		}(cfg.Sources[i])
	}
	wg.Wait()

	// chname->link->source
	chSrcs := make(map[string]map[string]SourceChannelBasic)
	s.Range(func(key, value any) bool {
		basics := value.([]SourceChannelBasic)
		for i, csb := range basics {
			if _, exist := chSrcs[csb.Name]; !exist {
				chSrcs[csb.Name] = make(map[string]SourceChannelBasic)
			}
			chSrcs[csb.Name][csb.Url] = basics[i]
		}
		return true
	})
	return chSrcs
}

func t2s(str string) string {
	return sat.DefaultDict().Read(str)
}

type SourceChannelBasic struct {
	Name         string
	NameInSource string
	Url          string
	Source       string
}

func ffmpegCheck(basic SourceChannelBasic) *SourceChannel {
	url := basic.Url
	steamInfo, cost, err := ffmpeg.ProbeStreamInfo(url)
	if err != nil {
		log.Printf("fail to parse stream info %s(%s): %v\n", basic.Name, url, err)
		return nil
	}
	var vsi, asi *ffmpeg.StreamItem
	for si, stream := range steamInfo.Streams {
		if stream.CodecType == ffmpeg.CodecVideo && vsi == nil {
			vsi = &steamInfo.Streams[si]
			continue
		}
		if stream.CodecType == ffmpeg.CodecAudio && asi == nil {
			asi = &steamInfo.Streams[si]
		}
	}
	if len(steamInfo.Streams) == 0 {
		err = errors.New("zero stream")
	} else if vsi == nil && asi == nil {
		err = errors.New("no video and audio stream")
	} else if vsi == nil {
		err = errors.New("no video but audio stream")
	} else if asi == nil {
		err = errors.New("no audio but video stream")
	}
	if err != nil {
		log.Printf("invalid stream %s(%s): %v\n", basic.Name, url, err)
		return nil
	}
	avgFrameRate := 0.0
	split := strings.Split(vsi.AvgFrameRate, "/")
	if len(split) == 2 {
		f0, _ := strconv.ParseInt(split[0], 10, 64)
		f1, _ := strconv.ParseInt(split[1], 10, 64)
		avgFrameRate = float64(f0) / float64(f1)
	}
	sc := SourceChannel{
		SourceChannelBasic: basic,
		VideoCodecName:     vsi.CodecName,
		VideoWidth:         vsi.Width,
		VideoHeight:        vsi.Height,
		VideoFrameRate:     avgFrameRate,
		AudioCodecName:     asi.CodecName,
		AudioChannels:      asi.Channels,
		AudioSampleRate:    asi.SampleRate,
		HDR:                vsi.IsHDR(),
		DetectCost:         cost,
	}
	log.Printf("valid stream cost[%s]: %s\n", cost.String(), sc.Print())
	return &sc
}

func isChannelNameInterested(channelsCfg []config.ConfigChannel, name string) (int, bool) {
	for idx, channel := range channelsCfg {
		if channel.Regexp.MatchString(name) {
			return idx, true
		}
	}
	return -1, false
}

type SourceChannelQualityCompare []SourceChannel

func (s SourceChannelQualityCompare) Len() int {
	return len(s)
}

func (s SourceChannelQualityCompare) Less(i, j int) bool {
	// VideoWidth-->HDR-->VideoFrameRate-->AudioChannels
	if s[i].VideoWidth == s[j].VideoWidth {
		if s[i].HDR == s[i].HDR {
			if s[i].VideoFrameRate == s[j].VideoFrameRate {
				return s[i].DetectCost > s[j].DetectCost
			}
			return s[i].VideoFrameRate < s[j].VideoFrameRate
		}
		return s[i].HDR
	}
	return s[i].VideoWidth < s[j].VideoWidth
}

func (s SourceChannelQualityCompare) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type SourceChannel struct {
	SourceChannelBasic

	AudioCodecName  string
	AudioChannels   int
	AudioSampleRate string

	VideoCodecName string
	VideoWidth     int
	VideoHeight    int
	VideoFrameRate float64

	HDR          bool
	DetectCost   time.Duration
	regexpedFlag bool
}

func (c SourceChannel) Resolution() string {
	return fmt.Sprintf("%dx%d", c.VideoWidth, c.VideoHeight)
}

func (c SourceChannel) StreamBasic() string {
	return fmt.Sprintf("video:%s/%s/%.3f audio:%s/%dch/%sHz", c.VideoCodecName, c.Resolution(),
		c.VideoFrameRate, c.AudioCodecName, c.AudioChannels, c.AudioSampleRate)
}

func (c SourceChannel) Print() string {
	return fmt.Sprintf("source[%s]%s->%s, %s, HDR[%v], delay[%s], url[%s]", c.Source, c.NameInSource, c.Name, c.StreamBasic(), c.HDR, c.DetectCost.String(), c.Url)
}

func RequestLink(link string) ([]byte, error) {
	resp, err := http.Get(link)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("status error: %d %s", resp.StatusCode, string(body))
	}
	return body, nil
}
