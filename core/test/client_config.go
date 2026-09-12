package test

import (
	"math/rand/v2"
)

type ClientConfig struct {
	ClientCount       int
	ClientRequestsMin int
	ClientRequestsMax int
	ClientCooldownMin int
	ClientCooldownMax int
}

type Client struct {
	ClientId           int
	ClientRequestsMake int
	ClientCooldownTime int
}

func CreateClients(
	ClientCount int,
	ClientRequestsMin int,
	ClientRequestsMax int,
	ClientCooldownMin int,
	ClientCooldownMax int,
) []Client {
	clients := make([]Client, 0, ClientCount)
	for i := range ClientCount {
		clientRequestMake := rand.IntN(ClientRequestsMax-ClientRequestsMin+1) + ClientRequestsMin
		clientCoolDown := rand.IntN(ClientCooldownMax-ClientCooldownMin+1) + ClientRequestsMin
		clients = append(clients, Client{ClientId: i, ClientRequestsMake: clientRequestMake, ClientCooldownTime: clientCoolDown})
	}
	return clients
}
