package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

func Run(config TestingConfig) {
	var wg sync.WaitGroup
	clients := CreateClients(config.ClientCount, config.ClientRequestsMin, config.ClientCooldownMax, config.ClientCooldownMin, config.ClientRequestsMax)
	for i := range config.Iterations {
		wg.Add(1)
		go ClientRunner(clients[i], &wg)
	}
	wg.Wait()
	fmt.Println("All client runners have finished")
}

func ClientRunner(client Client, wg *sync.WaitGroup) {
	defer wg.Done()
	post, err := http.Post("http://localhost:8080", "application/json", strings.NewReader(fmt.Sprintf(`{"client_key": "%s"}`, client.Id)))
	if err != nil {
		return
	}
	defer post.Body.Close()
	retryAfter := post.Header.Get("Retry-After")
	if retryAfter != "" {
		seconds, err := strconv.Atoi(retryAfter)
		if err != nil {
			fmt.Printf("Invalid retry after value %s", retryAfter)
		}
		fmt.Printf("Client %s waiting for %s seconds", client.Id, seconds)
		time.Sleep(time.Duration(seconds) * time.Second)
	}
}
