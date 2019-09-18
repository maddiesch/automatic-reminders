package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/maddiesch/automatic-reminders/auto"
)

func main() {
	lambda.Start(func(event auto.UpdateAccountEvent) error {
		return auto.FinishAccountUpdate(event)
	})
}
