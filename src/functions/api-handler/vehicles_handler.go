package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maddiesch/automatic-reminders/auto"
)

func getVehiclesHandler(c *gin.Context) {
	accountID := c.GetString(contextUserIDKey)
	account, err := auto.FindAccount(accountID)
	if err != nil {
		reportError(err, false)
		respondWithError(c, err)
		return
	}

	vehicles, err := auto.VehiclesForAccount(account)
	if err != nil {
		reportError(err, false)
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"Vehicles": vehicles})
}
