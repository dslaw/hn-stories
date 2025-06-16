package main

import (
	// "context"
	"fmt"
	"net/http"
	"time"
	// "github.com/jackc/pgx/v5"
	// "github.com/redis/go-redis/v9"
)

func main() {
	config := LoadConfig()
	sourceQueueConfig, _ := GetQueueConfigs(config.SourceQueueName)
	if sourceQueueConfig == nil {
		errorMsg := fmt.Sprintf("Unable to find a config for %s", config.SourceQueueName)
		panic(errorMsg)
	}

	// conn, err := pgx.Connect(context.Background(), config.DatabaseURL)
	// if err != nil {
	// 	panic(err)
	// }
	// defer conn.Close(context.Background())

	// repo := NewRepo(conn)

	// opts, err := redis.ParseURL(config.BrokerURL)
	// if err != nil {
	// 	panic(err)
	// }
	// redisClient := redis.NewClient(opts)
	// defer redisClient.Close()

	httpClient := &http.Client{Timeout: config.HNClientHTTPTimeout}
	client := NewHNClient(
		httpClient,
		config.HNClientBaseURL,
		config.HNClientAPIVersion,
		config.HNClientBackoff,
		config.HNClientMaxAttempts,
	)

	for {
		fmt.Println(client.FetchNewStories())
		time.Sleep(15 * time.Second)
	}
}
