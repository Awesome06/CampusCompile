package redis

import (
	"context"
	"log"
	"sync"

	redisClient "github.com/redis/go-redis/v9"
)

// Hub multiplexes Redis Pub/Sub connections to multiple local Go channels.
type Hub struct {
	sync.RWMutex
	subscribers map[string]map[chan string]struct{}
	redisSubs   map[string]*redisClient.PubSub
	redis       *redisClient.Client
}

var GlobalHub *Hub

func InitHub(r *redisClient.Client) {
	GlobalHub = &Hub{
		subscribers: make(map[string]map[chan string]struct{}),
		redisSubs:   make(map[string]*redisClient.PubSub),
		redis:       r,
	}
}

// Subscribe returns a Go channel that will receive messages for the given topic.
func (h *Hub) Subscribe(topic string) chan string {
	h.Lock()
	defer h.Unlock()

	// Buffer of 10 to prevent blocking the broadcaster if a client has brief network lag
	ch := make(chan string, 10)

	if h.subscribers[topic] == nil {
		h.subscribers[topic] = make(map[chan string]struct{})
	}
	h.subscribers[topic][ch] = struct{}{}

	// If this is the VERY FIRST subscriber, we actually connect to Redis
	if len(h.subscribers[topic]) == 1 {
		pubsub := h.redis.Subscribe(context.Background(), topic)
		h.redisSubs[topic] = pubsub

		// Start the broadcast daemon for this specific channel
		go h.broadcast(topic, pubsub.Channel())
	}

	return ch
}

// Unsubscribe removes the local Go channel. If it's the last one, it closes the Redis connection.
func (h *Hub) Unsubscribe(topic string, ch chan string) {
	h.Lock()
	defer h.Unlock()

	if _, ok := h.subscribers[topic]; ok {
		delete(h.subscribers[topic], ch)
		close(ch)

		// 🔒 SCALING MAGIC: If 0 students are left watching, sever the Redis connection to save memory
		if len(h.subscribers[topic]) == 0 {
			if pubsub, exists := h.redisSubs[topic]; exists {
				pubsub.Close()
				delete(h.redisSubs, topic)
			}
			delete(h.subscribers, topic)
		}
	}
}

// broadcast reads from the single Redis channel and fans out to all connected Go clients
func (h *Hub) broadcast(topic string, redisCh <-chan *redisClient.Message) {
	for msg := range redisCh {
		h.RLock()
		subs := h.subscribers[topic]
		targets := make([]chan string, 0, len(subs))
		for ch := range subs {
			targets = append(targets, ch)
		}
		h.RUnlock()

		for _, ch := range targets {
			select {
			case ch <- msg.Payload:
			default:
				// Drop message for a slow reader so we don't block the other 499 fast readers
				log.Printf("[WARN] Dropped message for a slow reader on topic %s", topic)
			}
		}
	}
}
