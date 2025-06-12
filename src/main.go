package main

import (
	"fmt"
	"time"
)

func main() {
	config := LoadConfig()
	sourceQueueConfig, dstQueueConfig := GetQueueConfigs(config.SourceQueueName)
	if sourceQueueConfig == nil {
		errorMsg := fmt.Sprintf("Unable to find a config for %s", config.SourceQueueName)
		panic(errorMsg)
	}

	for {
		fmt.Println(sourceQueueConfig, dstQueueConfig)
		time.Sleep(15 * time.Second)
	}
}
