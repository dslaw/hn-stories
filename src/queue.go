package main

import (
	"errors"
	"fmt"
	"time"
)

const (
	QueueKeyPrefix     = "ingestion-queue"
	DefaultGracePeriod = time.Minute
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

var queueConfigs = [...]QueueConfig{
	{Name: NewQueueName, ProcessAfter: 0 * time.Second, GracePeriod: DefaultGracePeriod},
	{Name: "15m", ProcessAfter: 15 * time.Minute, GracePeriod: DefaultGracePeriod},
	{Name: "30m", ProcessAfter: 30 * time.Minute, GracePeriod: DefaultGracePeriod},
	{Name: "1h", ProcessAfter: time.Hour, GracePeriod: DefaultGracePeriod},
	{Name: "3h", ProcessAfter: 3 * time.Hour, GracePeriod: DefaultGracePeriod},
	{Name: "6h", ProcessAfter: 6 * time.Hour, GracePeriod: DefaultGracePeriod},
}

// GetQueueConfigs returns the queue config for the given queue name, as well as
// the config for the next queue, if there is one, otherwise nil. If no queue
// config exists for the given queue name, nil is returned for both values.
func GetQueueConfigs(name string) (*QueueConfig, *QueueConfig) {
	var nextQueueConfig *QueueConfig

	for idx, queueConfig := range queueConfigs {
		if queueConfig.Name == name {
			nextIdx := idx + 1
			if nextIdx < len(queueConfigs) {
				nextQueueConfig = &queueConfigs[nextIdx]
			}

			return &queueConfig, nextQueueConfig
		}
	}

	return nil, nil
}
