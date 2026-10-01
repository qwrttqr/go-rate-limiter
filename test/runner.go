package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type ClientStats struct {
	Id         int
	Allowed    int
	Disallowed int
	Responded  int
	Errors     int
	AllowedAt  []time.Time
	AllowedLat []time.Duration
	LimitedLat []time.Duration
}

func Run(config TestingConfig) {
	if err := waitForReady("http://localhost:8080", 10*time.Second); err != nil {
		fmt.Printf("server never became ready: %v\n", err)
		os.Exit(1)
	}

	var httpClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        config.ClientCount,
			MaxIdleConnsPerHost: config.ClientCount,
			MaxConnsPerHost:     config.ClientCount,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	clients := CreateClients(config.ClientCount, config.Interval)
	stats := make([]*ClientStats, 0, len(clients))
	for i := range clients {
		stats = append(stats, &ClientStats{Id: clients[i].Id})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Duration)*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < config.ClientCount; i++ {
		wg.Add(1)
		go ClientRunner(ctx, httpClient, stats[i], clients[i], &wg)
	}
	wg.Wait()

	Report(stats, time.Since(start))

	fmt.Println("flood test finished")
}

func ClientRunner(ctx context.Context, httpClient *http.Client, stats *ClientStats, client Client, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(time.Duration(client.Interval) * time.Millisecond)
	defer ticker.Stop()

	var blockedUntil time.Time
	var secs int

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			requestStartedAt := time.Now()
			req, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:8080/limit", nil)
			if err != nil {
				stats.Errors++
				fmt.Println("failed to create request:", err.Error())
				continue
			}

			req.Header.Set("X-Client-Key", strconv.Itoa(client.Id))

			post, err := httpClient.Do(req)
			responseReceivedAt := time.Now()

			lat := time.Since(requestStartedAt)

			if err != nil {
				if ctx.Err() != nil {
					return
				}
				stats.Errors++
				stats.Responded++

				fmt.Println(err.Error())
				continue
			}
			io.Copy(io.Discard, post.Body)

			post.Body.Close()

			startedInWindow := requestStartedAt.Before(blockedUntil)
			finishedInWindow := responseReceivedAt.Before(blockedUntil)

			if startedInWindow && finishedInWindow && post.StatusCode == http.StatusOK {
				fmt.Printf(
					"client=%d\nretryAfter=%d\ndeadline=%s\nremaining=%s\n",
					client.Id,
					secs,
					blockedUntil.Format(time.RFC3339Nano),
					blockedUntil.Sub(responseReceivedAt),
				)
			}

			switch post.StatusCode {
			case http.StatusOK:
				stats.Allowed++
				stats.AllowedLat = append(stats.AllowedLat, lat)
				stats.AllowedAt = append(stats.AllowedAt, requestStartedAt)
			case http.StatusTooManyRequests:
				stats.Disallowed++
				stats.LimitedLat = append(stats.LimitedLat, lat)
				if s, err := strconv.Atoi(post.Header.Get("Retry-After")); err == nil {
					secs = s
					blockedUntil = requestStartedAt.Truncate(time.Second).Add(time.Duration(secs) * time.Second)
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
