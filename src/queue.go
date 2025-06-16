package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	QueueKeyPrefix     = "ingestion-queue"
	DefaultGracePeriod = 1 * time.Minute
	NewQueueName       = "new"
)

var ErrTimeout = errors.New("Timeout expired")

// QueueConfig is the configuration for messaging queue.
type QueueConfig struct {
	// Name of the messaging queue.
	Name string
	// Time to wait after the initial ingestion of an item for (re)processing.
	ProcessAfter time.Duration
	// Duration of the (re)processing window.
	GracePeriod time.Duration
}

func (c QueueConfig) MakeKey() string {
	return fmt.Sprintf("%s:%s", QueueKeyPrefix, c.Name)
}

// MakeNewQueueConfig makes a QueueConfig for the "new" queue.
func MakeNewQueueConfig() QueueConfig {
	return QueueConfig{
		Name:         NewQueueName,
		ProcessAfter: 0 * time.Second,
		GracePeriod:  DefaultGracePeriod,
	}
}

// MakeQueueConfig makes a QueueConfig based on the queue's name.
func MakeQueueConfig(name string) (QueueConfig, error) {
	var (
		err          error
		processAfter = 0 * time.Second
	)

	if name != NewQueueName {
		processAfter, err = time.ParseDuration(name)
	}

	return QueueConfig{
		Name:         name,
		ProcessAfter: processAfter,
		GracePeriod:  DefaultGracePeriod,
	}, err
}

// Message is a message for communicating that a Hacker News story be processed.
type Message struct {
	// StoryID is the (external) id of the Hacker News story.
	StoryID int64 `json:"story_id"`
	// CreatedAt gives the time the story was created at, or the best guess.
	CreatedAt time.Time `json:"created_at"`
	// ProcessAt gives the time at which the message should be processed.
	ProcessAt time.Time `json:"-"`
}

func (msg *Message) Encode() (string, error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (msg *Message) Decode(data string, processAt float64) error {
	err := json.Unmarshal([]byte(data), &msg)
	if err != nil {
		return err
	}

	msg.ProcessAt = time.Unix(int64(processAt), 0).UTC()
	return nil
}

// Broker is an interface to the message broker.
type Broker interface {
	BZPopMin(context.Context, time.Duration, ...string) *redis.ZWithKeyCmd
	ZAddNX(context.Context, string, ...redis.Z) *redis.IntCmd
}

// PriorityQueue represents a persistent priority queue. Enqueued messages are
// unique and ordered by the time at which they should be processed
type PriorityQueue struct {
	client  Broker
	config  QueueConfig
	Timeout time.Duration
}

func NewPriorityQueue(client Broker, config QueueConfig, timeout time.Duration) *PriorityQueue {
	return &PriorityQueue{client: client, config: config, Timeout: timeout}
}

func (pq *PriorityQueue) QueueName() string {
	return pq.config.Name
}

func (pq *PriorityQueue) ProcessAfter() time.Duration {
	return pq.config.ProcessAfter
}

func (pq *PriorityQueue) GracePeriod() time.Duration {
	return pq.config.GracePeriod
}

func (pq *PriorityQueue) Enqueue(ctx context.Context, msg Message) error {
	score := msg.ProcessAt.Unix()
	member, err := msg.Encode()
	if err != nil {
		return err
	}

	key := pq.config.MakeKey()
	return pq.client.ZAddNX(ctx, key, redis.Z{Member: member, Score: float64(score)}).Err()
}

// Dequeue dequeues the next message to be processed, blocking until a new
// message is available or the configured timeout is reached. Note that this
// implies at-most once message delivery semantics.
func (pq *PriorityQueue) Dequeue(ctx context.Context) (Message, error) {
	msg := Message{}

	key := pq.config.MakeKey()
	item, err := pq.client.BZPopMin(ctx, pq.Timeout, key).Result()
	if err != nil {
		return msg, err
	}

	if item == nil {
		return msg, ErrTimeout
	}

	err = msg.Decode(item.Member.(string), item.Score)
	return msg, err
}
