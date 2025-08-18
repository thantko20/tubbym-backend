package pubsub

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

type Pubsub interface {
	Subscribe(topic string) *Client
	Unsubscribe(topic string, client *Client)
	Publish(topic, message string)
}

type Broker struct {
	topics map[string]map[*Client]struct{}
}
