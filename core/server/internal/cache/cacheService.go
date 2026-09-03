package cache

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

type cacheEntry struct {
	entry any
	time  int64
}

type Cache struct {
	entries        map[string]*cacheEntry
	mutex          sync.Mutex
	commandChannel chan string
	expirationTime int64
	ticker         *time.Ticker
}

func (cache *Cache) Store(key string, value any) {
	cache.mutex.Lock()
	cache.entries[key] = &cacheEntry{entry: value, time: time.Now().Unix()}
	cache.mutex.Unlock()
}

func (cache *Cache) Get(key string) (any, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	result, present := cache.entries[key]
	if !present {
		return nil, errors.New("cache entry is not present")
	}
	return result.entry, nil
}

func (cache *Cache) LoadOrStore(key string, value any) any {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	result, present := cache.entries[key]
	if !present {
		cache.entries[key] = &cacheEntry{entry: value, time: time.Now().Unix()}
		return value
	}
	return result.entry
}

func (cache *Cache) Delete(key string) {
	cache.mutex.Lock()
	delete(cache.entries, key)
	cache.mutex.Unlock()
}

func (cache *Cache) handleEviction() {
	currentTime := time.Now().Unix()
	log.Println("doing cache eviction")
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	for key, value := range cache.entries {
		if currentTime-cache.expirationTime > value.time {
			log.Printf("deleting key %s\n", key)
			delete(cache.entries, key)
		}
	}
}

func (cache *Cache) sendCommand(cmd string) {
	cache.commandChannel <- cmd
}

func NewCache(expirationTime int64, evictionInterval time.Duration) *Cache {
	ticker := time.NewTicker(evictionInterval)
	cache := &Cache{
		entries:        make(map[string]*cacheEntry),
		expirationTime: expirationTime,
		commandChannel: make(chan string),
		ticker:         ticker}
	go func() {
		for {
			select {
			case cmd := <-cache.commandChannel:
				if cmd == "stop" {
					fmt.Println("cache ticker stopped, !!!eviction stopped!!!")
					cache.ticker.Stop()
					return
				}
			case <-cache.ticker.C:
				cache.handleEviction()
			}
		}
	}()

	return cache
}
