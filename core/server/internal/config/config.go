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
	UsedAlgo                                  string                          `yaml:"use_algo"`
	TokenBucketConfiguration                  TokenBucketConfiguration        `yaml:"token_bucket"`
	FixedWindowConfiguration                  FixedWindowConfiguration        `yaml:"fixed_window"`
	RollingWindowConfiguration                RollingWindowConfiguration      `yaml:"rolling_window"`
	RollingWindowRedisConfiguration           RollingWindowRedisConfiguration `yaml:"rolling_window_redis"`
	RollingWindowConcurrentRedisConfiguration RollingWindowConcurrentRedis    `yaml:"rolling_window_concurrent_redis"`
}
type TokenBucketConfiguration struct {
	Capacity int `yaml:"capacity"`
	Rate     int `yaml:"rate"`
}

type FixedWindowConfiguration struct {
	WindowSize  int `yaml:"window_size"`
	MaxRequests int `yaml:"max_requests"`
}

type RollingWindowConfiguration struct {
	WindowSize  int `yaml:"window_size"`
	MaxRequests int `yaml:"max_requests"`
}

type RollingWindowRedisConfiguration struct {
	WindowSize  int     `yaml:"window_size"`
	TTL         int     `yaml:"TTL"`
	MaxRequests int     `yaml:"max_requests"`
	MinInterval float64 `yaml:"min_interval"`
}

type RollingWindowConcurrentRedis struct {
	Capacity    int `yaml:"window_size"`
	TTL         int `yaml:"TTL"`
	MaxRequests int `yaml:"max_requests"`
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
