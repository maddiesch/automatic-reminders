package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/maddiesch/automatic-reminders/auto"
)

func main() {
	lambda.Start(func(event auto.UpdateAccountEvent) (*auto.UpdateAccountEvent, error) {
		response := &event

		auto.UpdateCurrentUpdateState("UPDATING_TRIPS", event)

		err := auto.UpdateTripsForAccountUpdate(response)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}
