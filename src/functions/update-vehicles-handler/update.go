package main

import (
	"github.com/maddiesch/automatic-reminders/auto"
	"github.com/maddiesch/serverless"
)

func updateVehiclesForAccount(accountID string) error {
	serverless.Log("Updating vehicles for ", accountID)

	account, err := auto.FindAccount(accountID)
	if err != nil {
		return err
	}

	token, err := auto.FindAutomaticAccessTokenForAccount(account)
	if err != nil {
		return err
	}

	return auto.UpdateVehiclesForAccount(account, token)
}
