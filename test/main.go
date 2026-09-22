package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	var cfg TestingConfig

	args := os.Args[1:]

	for i := 0; i < len(args); i += 2 {
		flag := args[i]

		if i+1 >= len(args) {
			fmt.Printf("Error flag %s missing value\n", flag)
		}
		valueStr := args[i+1]
		valueInt, err := strconv.Atoi(valueStr)
		if err != nil {
			fmt.Printf("Error: value for flag %s must be parsable as integer", flag)
		}
		// example format -clients 100 -clients_min_reqs 10 -clients_max_reqs 100  -clients_min_cooldown 10 -clients_max_cooldown 100
		switch flag {
		case "-clients":
			cfg.ClientCount = valueInt
		case "-duration":
			cfg.Duration = valueInt
		case "-interval":
			cfg.Interval = valueInt
		}
	}
	fmt.Printf("Parsed Configuration: %+v\n", cfg)
	fmt.Println("Proceeding tests....")
	Run(cfg)
}
