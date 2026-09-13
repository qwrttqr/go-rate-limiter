package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func Run(config TestingConfig) {
	if err := waitForReady("http://localhost:8080", 10*time.Second); err != nil {
		fmt.Printf("Server never became ready: %v\n", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	clients := CreateClients(config.ClientCount, config.ClientRequestsMin, config.ClientRequestsMax, config.ClientCooldownMin, config.ClientCooldownMax)
	for i := range config.Iterations {
		wg.Add(1)
		go ClientRunner(clients[i], &wg)
	}
	wg.Wait()
	fmt.Println("All client runners have finished")
}

func ClientRunner(client Client, wg *sync.WaitGroup) {
	defer wg.Done()
	post, err := http.Post("http://localhost:8080", "application/json", strings.NewReader(fmt.Sprintf(`{"client_key": "%d"}`, client.Id)))
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
		fmt.Printf("Client %d waiting for %d seconds", client.Id, seconds)
		time.Sleep(time.Duration(seconds) * time.Second)
	}
}

func waitForReady(target string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(target)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s", target)
}
