package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type BaseConfiguration struct {
	UsedAlgo     string       `yaml:"use_algo"`
	AlgoSettings AlgoSettings `yaml:"algo_settings"`
}

type StandaloneServiceConfiguration struct {
	BaseConfiguration `yaml:",inline"`
	Store             string   `yaml:"store"`
	Backends          Backends `yaml:"backends"`
}

type Backends struct {
	InMemory InMemoryBackendSettings `yaml:"in_memory"`
	Redis    RedisBackendSettings    `yaml:"redis"`
}

type InMemoryBackendSettings struct {
	DefaultTtl   int64  `yaml:"default_ttl"`
	EvictionTime int64  `yaml:"eviction_time"`
	ShardsCount  uint32 `yaml:"shards_count"`
}

type RedisBackendSettings struct {
	Addr       string `yaml:"addr"`
	Password   string `yaml:"password"`
	Db         int    `yaml:"db"`
	DefaultTtl int64  `yaml:"default_ttl"`
}
type AlgoSettings struct {
	Capacity    *float64 `yaml:"capacity"`
	Rate        *float64 `yaml:"rate"`
	WindowSize  *int64   `yaml:"window_size"`
	MaxRequests *int64   `yaml:"max_requests"`
}

func ReadConfig() (StandaloneServiceConfiguration, error) {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return StandaloneServiceConfiguration{}, fmt.Errorf("error reading configuration: %s", err.Error())
	}
	var config StandaloneServiceConfiguration
	if err := yaml.Unmarshal(data, &config); err != nil {
		return StandaloneServiceConfiguration{}, fmt.Errorf("parsing config yaml: %w", err)
	}
	applyEnvOverrides(&config)
	return config, nil
}

func applyEnvOverrides(config *StandaloneServiceConfiguration) {
	if v := os.Getenv("STORE_BACKEND"); v != "" {
		config.Store = v
	}
}
