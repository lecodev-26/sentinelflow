package main

import (
	"context"
	"fmt"

	sentinelflow "github.com/lecodev-26/sentinelflow/sdk/go/sentinelflow"
)

func main() {
	client := sentinelflow.New("http://localhost:8080", "your-api-key")
	if err != nil {
		panic(err)
	}
	response, err := client.Chat(context.Background(), sentinelflow.ChatRequest{Model: "default", Messages: []sentinelflow.Message{{Role: "user", Content: "Hello SentinelFlow"}}})
	if err != nil {
		panic(err)
	}
	fmt.Println(response)
}
