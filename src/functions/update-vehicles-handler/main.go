package main

import (
	"github.com/aws/aws-lambda-go/lambda"
)

// Event inbound event handler.
type Event struct {
	AccountID string
}

func main() {
	lambda.Start(func(event Event) error {
		return updateVehiclesForAccount(event.AccountID)
	})
}
