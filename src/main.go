package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

func main() {
	config := LoadConfig()

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

	var (
		consumer Consumer
		producer Producer
	)

	if config.SourceQueueName == "" && config.DstQueueName == NewQueueName {
		// Fetch new stories and put them on the "new" queue as messages.
		dstQueueConfig := MakeNewQueueConfig()
		dstQueue := NewPriorityQueue(redisClient, dstQueueConfig, config.ConsumerTimeout)
		consumer = NewLatestStoryConsumer(client, config.ConsumerPollInterval, config.ConsumerTimeout)
		producer = NewMessageProducer(dstQueue)
	} else if config.SourceQueueName != "" && config.DstQueueName != "" {
		// Consume messages from source queue and put new messages onto
		// destination queue.
		sourceQueueConfig, err := MakeQueueConfig(config.SourceQueueName)
		if err != nil {
			panic(err)
		}
		dstQueueConfig, err := MakeQueueConfig(config.DstQueueName)
		if err != nil {
			panic(err)
		}

		sourceQueue := NewPriorityQueue(redisClient, sourceQueueConfig, config.ConsumerTimeout)
		dstQueue := NewPriorityQueue(redisClient, dstQueueConfig, config.ConsumerTimeout)
		consumer = NewMessageConsumer(client, sourceQueue, repo)
		producer = NewMessageProducer(dstQueue)
	} else if config.SourceQueueName != "" && config.DstQueueName == "" {
		// Consume messages from last source queue and do not produce any new
		// messages.
		sourceQueueConfig, err := MakeQueueConfig(config.SourceQueueName)
		if err != nil {
			panic(err)
		}

		sourceQueue := NewPriorityQueue(redisClient, sourceQueueConfig, config.ConsumerTimeout)
		consumer = NewMessageConsumer(client, sourceQueue, repo)
		producer = &NopProducer{}
	} else {
		errorMsg := fmt.Sprintf(
			"Invalid queue configuration: source=%s dst=%s",
			config.SourceQueueName,
			config.DstQueueName,
		)
		panic(errorMsg)
	}

	Run(context.Background(), consumer, producer)
}
