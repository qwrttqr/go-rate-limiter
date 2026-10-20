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
	Store             string          `yaml:"store"`
	Backends          Backends        `yaml:"backends"`
	Metrics           MetricsSettings `yaml:"metrics"`
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

type MetricsSettings struct {
	UsePrometheus bool `yaml:"use_prometheus"`
	Port          int  `yaml:"port"`
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
	if err := config.Validate(); err != nil {
		return StandaloneServiceConfiguration{}, fmt.Errorf("invalid configuration: %w", err)
	}
	applyEnvOverrides(&config)
	return config, nil
}

func applyEnvOverrides(config *StandaloneServiceConfiguration) {
	if v := os.Getenv("STORE_BACKEND"); v != "" {
		config.Store = v
	}
}

func (c *StandaloneServiceConfiguration) Validate() error {
	if c.Store == "" {
		return fmt.Errorf("field 'store' is required")
	}
	if c.UsedAlgo == "" {
		return fmt.Errorf("field 'use_algo' is required")
	}

	switch c.Store {
	case "in_memory":
		if c.Backends.InMemory.ShardsCount == 0 {
			return fmt.Errorf("backends.in_memory.shards_count must be greater than 0")
		}
		if c.Backends.InMemory.DefaultTtl <= 0 {
			return fmt.Errorf("backends.in_memory.default_ttl must be positive")
		}
	case "redis":
		if c.Backends.Redis.Addr == "" {
			return fmt.Errorf("backends.redis.addr is required when store is 'redis'")
		}
	default:
		return fmt.Errorf("unknown store configuration type: %q", c.Store)
	}

	return nil
}
