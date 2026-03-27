package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Allowed []struct {
		Host  string   `yaml:"host"`
		Users []string `yaml:"users"`
	} `yaml:"allowed"`
}

func ReadConfig(path string) (*Config, error) {
	configData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	conf := &Config{}
	err = yaml.Unmarshal(configData, conf)
	return conf, err
}
