package config

import "regexp"

type ConfigSource struct {
	Name   string `yaml:"name"`
	Format string `yaml:"format"`
	URI    string `yaml:"uri"`
}

type ConfigChannel struct {
	Name    string         `yaml:"name"`
	Station string         `yaml:"station"`
	Regex   string         `yaml:"regex"`
	Regexp  *regexp.Regexp `yaml:"-"`
}

type ConfigGroup struct {
	Name     string   `yaml:"name"`
	Stations []string `yaml:"stations"`
	Channels []string `yaml:"channels"`
}

func (g ConfigGroup) AllChannels(channelsByStation map[string][]string) []string {
	rlt := make([]string, 0)
	for _, station := range g.Stations {
		rlt = append(rlt, channelsByStation[station]...)
	}
	rlt = append(rlt, g.Channels...)
	return rlt
}

type Config struct {
	Sources       []ConfigSource  `yaml:"sources"`
	Channels      []ConfigChannel `yaml:"channels"`
	Groups        []ConfigGroup   `yaml:"groups"`
	EPGs          []string        `yaml:"epgs"`
	ExportFormats []string        `yaml:"exports"`

	ChannelsByStation map[string][]string `yaml:"-"`
}

func (c *Config) Check() error {
	channelsByStation := make(map[string][]string, 0)
	for i, channel := range c.Channels {
		compile, err := regexp.Compile(channel.Regex)
		if err != nil {
			return err
		}
		c.Channels[i].Regexp = compile
		if _, exist := channelsByStation[channel.Station]; !exist {
			channelsByStation[channel.Station] = make([]string, 0)
		}
		channelsByStation[channel.Station] = append(channelsByStation[channel.Station], channel.Name)
	}
	c.ChannelsByStation = channelsByStation
	return nil
}
