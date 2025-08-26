package pubsub

import (
	"sync"
)

type SubscribeReq struct {
	Topic  string
	Client *Client
}

type UnsubscribeReq struct {
	Topic  string
	Client *Client
}

type Client struct {
	ch   chan string
	done chan struct{}
}

func NewClient() *Client {
	return &Client{
		ch:   make(chan string, 10), // buffered channel to prevent blocking
		done: make(chan struct{}),
	}
}

func (c *Client) Channel() <-chan string {
	return c.ch
}

func (c *Client) Done() <-chan struct{} {
	return c.done
}

func (c *Client) Close() {
	close(c.done)
}

type Pubsub interface {
	Subscribe(topic string) *Client
	Unsubscribe(topic string, client *Client)
	Publish(topic, message string)
	Close()
}

type Broker struct {
	topics map[string]map[*Client]struct{}
	mu     sync.RWMutex
}

func NewBroker() *Broker {
	return &Broker{
		topics: make(map[string]map[*Client]struct{}),
	}
}

func (b *Broker) Subscribe(topic string) *Client {
	b.mu.Lock()
	defer b.mu.Unlock()

	client := NewClient()

	if b.topics[topic] == nil {
		b.topics[topic] = make(map[*Client]struct{})
	}

	b.topics[topic][client] = struct{}{}
	return client
}

func (b *Broker) Unsubscribe(topic string, client *Client) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if clients, exists := b.topics[topic]; exists {
		delete(clients, client)
		if len(clients) == 0 {
			delete(b.topics, topic)
		}
	}

	client.Close()
}

func (b *Broker) Publish(topic, message string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if clients, exists := b.topics[topic]; exists {
		for client := range clients {
			select {
			case client.ch <- message:
			case <-client.done:
				// Client is closed, remove it
				go func(c *Client) {
					b.Unsubscribe(topic, c)
				}(client)
			default:
				// Channel is full, skip this client to prevent blocking
			}
		}
	}
}

func (b *Broker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for topic, clients := range b.topics {
		for client := range clients {
			client.Close()
		}
		delete(b.topics, topic)
	}
}
