package main

import (
	"fmt"
	"os"
	"time"
)

func MakeMessage(workerType string) string {
	return fmt.Sprintf("Hi! from %s...", workerType)
}

func main() {
	workerType, _ := os.LookupEnv("WORKER_TYPE")
	waitFor := 15 * time.Second
	for {
		fmt.Println(MakeMessage(workerType))
		time.Sleep(waitFor)
	}
}
