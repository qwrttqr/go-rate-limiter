package main

import (
	"context"
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

	clients := CreateClients(config.ClientCount, config.Interval)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Duration)*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < config.ClientCount; i++ {
		wg.Add(1)
		go ClientRunner(ctx, clients[i], &wg)
	}
	wg.Wait()
	fmt.Println("flood test finished")
}

func ClientRunner(ctx context.Context, client Client, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(time.Duration(client.Interval) * time.Millisecond)
	defer ticker.Stop()
	var retryAfterSeconds int
	var lastRequestTime time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			post, err := http.Post("http://localhost:8080/limit", "application/json", strings.NewReader(fmt.Sprintf(`{"client_key": "%d"}`, client.Id)))
			if err != nil {
				fmt.Printf(err.Error())
			}
			post.Body.Close()
			if retryAfterSeconds > 0 {
				retryUntil := lastRequestTime.Add(
					time.Duration(retryAfterSeconds) * time.Second,
				)

				if now.Before(retryUntil) {
					if post.StatusCode != 429 {
						fmt.Printf("!!!client %d was not fallback!!!\n", client.Id)
						fmt.Printf("retry after was %d", retryAfterSeconds)
					}
					fmt.Printf(
						"client %d: request is inside Retry-After period\n",
						client.Id,
					)
				}
			}

			retryAfter := post.Header.Get("Retry-After")

			if retryAfter != "" {
				var err error
				retryAfterSeconds, err = strconv.Atoi(retryAfter)
				if err != nil {
					fmt.Printf(err.Error())
				}
			}
		}
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
