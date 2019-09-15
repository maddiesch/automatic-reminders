package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

// Event is the payload sent to lambda
type Event struct {
	Greeting string `json:"greeting"`
	Name     string `json:"name"`
}

func main() {
	lambda.Start(func(ctx context.Context, event Event) {
		log.Println(greeting(event))
	})
}

func greeting(event Event) string {
	return fmt.Sprintf("%s %s!", event.Greeting, event.Name)
}
