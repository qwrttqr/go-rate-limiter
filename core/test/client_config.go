package test

import (
	"fmt"
	"math/rand/v2"
)


type ClientConfig struct {
	ClientCount       int
	ClientMinRequests int
	ClientMaxRequests int
	ClientCooldownMin int
	ClientCooldownMax int
}

type Client struct {
	ClientId           string
	ClientRequestsMake int
	ClientCoolDownTime int
}

func CreateClients(
	ClientCount int,
	ClientMinRequests int,
	ClientMaxRequests int,
	ClientCooldownMin int,
	ClientCooldownMax int,
) []Client {
	clients := make([]Client, 0, ClientCount)
	for i := 0; i < ClientCount; i++ {

		clients = append(clients, &Client{ClientId: i, ClientRequestsMake: })
	}
}
