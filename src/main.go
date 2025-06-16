package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

func main() {
	config := LoadConfig()
	sourceQueueConfig, dstQueueConfig := GetQueueConfigs(config.SourceQueueName)
	if sourceQueueConfig == nil && dstQueueConfig == nil {
		errorMsg := fmt.Sprintf("Unable to find a config for %s", config.SourceQueueName)
		panic(errorMsg)
	}

	conn, err := pgx.Connect(context.Background(), config.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close(context.Background())

	repo := NewRepo(conn)

	opts, err := redis.ParseURL(config.BrokerURL)
	if err != nil {
		panic(err)
	}
	redisClient := redis.NewClient(opts)
	defer redisClient.Close()

	httpClient := &http.Client{Timeout: config.HNClientHTTPTimeout}
	client := NewHNClient(
		httpClient,
		config.HNClientBaseURL,
		config.HNClientAPIVersion,
		config.HNClientBackoff,
		config.HNClientMaxAttempts,
	)

	if sourceQueueConfig == nil && dstQueueConfig == nil {
		// Should be unreachable.
		errorMsg := fmt.Sprintf("Unable to find a config for %s", config.SourceQueueName)
		panic(errorMsg)
	} else if sourceQueueConfig == nil && dstQueueConfig != nil {
		dstQueue := NewPriorityQueue(redisClient, *dstQueueConfig, config.ConsumerTimeout)
		consumer := NewLatestStoryConsumer(client, config.ConsumerPollInterval, config.ConsumerTimeout)
		producer := NewMessageProducer(dstQueue)
	} else if sourceQueueConfig != nil && dstQueueConfig != nil {
		sourceQueue := NewPriorityQueue(redisClient, *sourceQueueConfig, config.ConsumerTimeout)
		dstQueue := NewPriorityQueue(redisClient, *dstQueueConfig, config.ConsumerTimeout)
		consumer := NewMessageConsumer(client, sourceQueue, repo)
		producer := NewMessageProducer(dstQueue)
	} else { // sourceQueueConfig != nil && dstQueueConfig == nil
		sourceQueue := NewPriorityQueue(redisClient, *sourceQueueConfig, config.ConsumerTimeout)
		consumer := NewMessageConsumer(client, sourceQueue, repo)
		producer := &NopProducer{}
	}

	for {
		time.Sleep(15 * time.Second)
	}
}
