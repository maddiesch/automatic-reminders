package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/maddiesch/automatic-reminders/auto"
)

func main() {
	lambda.Start(func(event auto.UpdateAccountEvent) (*auto.UpdateAccountEvent, error) {
		auto.UpdateCurrentUpdateState("UPDATE_VEHICLES", event)

		err := updateVehiclesForAccount(event.AccountID)
		if err != nil {
			return nil, err
		}
		return &event, nil
	})
}
