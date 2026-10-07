package main

import (
	"context"
	"fmt"

	sentinelflow "github.com/lecodev-26/sentinelflow/sdk/go/sentinelflow"
)

func main() {
	client, err := sentinelflow.New(sentinelflow.Config{BaseURL: "http://localhost:8080", APIKey: "your-api-key"})
	if err != nil {
		panic(err)
	}
	response, err := client.Chat(context.Background(), sentinelflow.ChatRequest{Model: "default", Input: "Hello SentinelFlow"})
	if err != nil {
		panic(err)
	}
	fmt.Println(response)
}
