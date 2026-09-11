package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Configuration struct {
	UsedAlgo     string       `yaml:"use_algo"`
	Store        string       `yaml:"store"`
	AlgoSettings AlgoSettings `yaml:"algo_settings"`
	Backends     Backends     `yaml:"backends"`
}

type Backends struct {
	InMemory InMemoryBackendSettings `yaml:"in_memory"`
	Redis    RedisBackendSettings    `yaml:"redis"`
}

type InMemoryBackendSettings struct {
	DefaultTtl   int64 `yaml:"default_ttl"`
	EvictionTime int64 `yaml:"eviction_time"`
}

type RedisBackendSettings struct {
	Addr       string `yaml:"addr"`
	Password   string `yaml:"password"`
	Db         int    `yaml:"db"`
	DefaultTtl int64  `yaml:"default_ttl"`
}
type AlgoSettings struct {
	Capacity    *int64   `yaml:"capacity"`
	Rate        *float64 `yaml:"rate"`
	WindowSize  *int64   `yaml:"window_size"`
	MaxRequests *int64   `yaml:"max_requests"`
	MinInterval *string  `yaml:"min_interval"`
}

func ReadConfig() (Configuration, error) {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return Configuration{}, fmt.Errorf("error reading configuration: %s", err.Error())
	}
	var config Configuration
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Configuration{}, fmt.Errorf("parsing config yaml: %w", err)
	}
	return config, nil
}
