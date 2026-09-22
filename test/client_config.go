package main

type TestingConfig struct {
	ClientCount int
	Duration    int
	Interval    int
}

type Client struct {
	Id       int
	Interval int
}

func CreateClients(
	ClientCount int,
	Interval int,
) []Client {
	clients := make([]Client, 0, ClientCount)
	for i := range ClientCount {
		clients = append(clients, Client{Id: i, Interval: Interval})
	}
	return clients
}
