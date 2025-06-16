package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func LoadEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		errorMsg := fmt.Sprintf("Unable to read environment variable `%s`", key)
		panic(errorMsg)
	}
	return value
}

func LoadDurationEnv(key string) time.Duration {
	s := LoadEnv(key)
	value, err := time.ParseDuration(s)
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to parse duration environment variable `%s`: `%s`", key, s)
		panic(errorMsg)
	}
	return value
}

func LoadIntEnv(key string) int {
	s := LoadEnv(key)
	value, err := strconv.Atoi(s)
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to parse int environment variable `%s`: `%s`", key, s)
		panic(errorMsg)
	}
	return value
}

type Config struct {
	DatabaseURL          string
	BrokerURL            string
	SourceQueueName      string
	DstQueueName         string
	HNClientBaseURL      string
	HNClientAPIVersion   string
	HNClientBackoff      time.Duration
	HNClientMaxAttempts  int
	HNClientHTTPTimeout  time.Duration
	ConsumerPollInterval time.Duration
	ConsumerTimeout      time.Duration
}

func LoadConfig() *Config {
	config := &Config{}
	config.DatabaseURL = LoadEnv("DATABASE_URL")
	config.BrokerURL = LoadEnv("BROKER_URL")
	config.SourceQueueName = LoadEnv("SOURCE_QUEUE_NAME")
	config.HNClientBaseURL = LoadEnv("HN_CLIENT_BASE_URL")
	config.HNClientAPIVersion = LoadEnv("HN_CLIENT_API_VERSION")
	config.HNClientBackoff = LoadDurationEnv("HN_CLIENT_BACKOFF")
	config.HNClientMaxAttempts = LoadIntEnv("HN_CLIENT_MAX_ATTEMPTS")
	config.HNClientHTTPTimeout = LoadDurationEnv("HN_CLIENT_HTTP_TIMEOUT")
	config.ConsumerPollInterval = LoadDurationEnv("CONSUMER_POLL_INTERVAL")
	config.ConsumerTimeout = LoadDurationEnv("CONSUMER_TIMEOUT")
	return config
}
