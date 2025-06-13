package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	config := LoadConfig()
	sourceQueueConfig, _ := GetQueueConfigs(config.SourceQueueName)
	if sourceQueueConfig == nil {
		errorMsg := fmt.Sprintf("Unable to find a config for %s", config.SourceQueueName)
		panic(errorMsg)
	}

	conn, err := pgx.Connect(context.Background(), config.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close(context.Background())

	repo := NewRepo(conn)
	for {
		var count int32
		err := conn.QueryRow(context.Background(), "select count(*) from stories").Scan(&count)
		if err != nil {
			panic(err)
		}

		fmt.Println(count)
		time.Sleep(15 * time.Second)
	}
}
