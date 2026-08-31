package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Root struct {
	Configuration Configuration `yaml:"configuration"`
}
type Configuration struct {
	UsedAlgo         string                   `yaml:"use_algo"`
	IdentifyClientBy []IdentificationStrategy `yaml:"identify_client_by"`
	AlgoSettings     AlgoSettings             `yaml:"algo_settings"`
}

type IdentificationStrategy struct {
	Type       string `yaml:"type,omitempty"`
	Format     string `yaml:"format,omitempty"`
	CookieName string `yaml:"cookie_name,omitempty"`
	Version    string `yaml:"version,omitempty"`
}
type AlgoSettings struct {
	Capacity    *int64   `yaml:"capacity"`
	Rate        *float64 `yaml:"rate"`
	WindowSize  *string  `yaml:"window_size"`
	MaxRequests *int64   `yaml:"max_requests"`
	TTL         *string  `yaml:"ttl"`
	MinInterval *string  `yaml:"min_interval"`
}

func ReadConfig() (Configuration, error) {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return Configuration{}, fmt.Errorf("error reading configuration: %s", err.Error())
	}
	var root Root
	if err := yaml.Unmarshal(data, &root); err != nil {
		return Configuration{}, fmt.Errorf("parsing config yaml: %w", err)
	}
	return root.Configuration, nil
}
