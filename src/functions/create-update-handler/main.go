package main

import (
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/maddiesch/automatic-reminders/auto"
)

func main() {
	lambda.Start(func(event auto.UpdateAccountEvent) (*auto.UpdateAccountEvent, error) {
		response := &event
		updateID, err := auto.CreateAccountUpdate(event.AccountID)
		if err != nil {
			return nil, err
		}

		response.UpdateID = updateID
		response.StartTime = time.Now().Unix()

		return response, nil
	})
}
