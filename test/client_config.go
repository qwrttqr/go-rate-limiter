package main

import (
	"math/rand/v2"
)

type TestingConfig struct {
	ClientCount       int
	ClientRequestsMin int
	ClientRequestsMax int
	ClientCooldownMin int
	ClientCooldownMax int
	Iterations        int
}

type Client struct {
	Id                 int
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
		clients = append(clients, Client{Id: i, ClientRequestsMake: clientRequestMake, ClientCooldownTime: clientCoolDown})
	}
	return clients
}
