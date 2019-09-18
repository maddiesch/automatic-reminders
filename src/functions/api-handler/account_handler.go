package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/maddiesch/automatic-reminders/auto"
)

func getAccountHandler(c *gin.Context) {
	accountID := c.GetString(contextUserIDKey)
	account, err := auto.FindAccount(accountID)
	if err != nil {
		reportError(err, false)
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, account)
}

func postAccountUpdateHandler(c *gin.Context) {
	accountID := c.GetString(contextUserIDKey)

	err := auto.UpdateAccount(auto.UpdateAccountInput{
		AccountID:          accountID,
		UpdateStateMachine: os.Getenv("UPDATE_ACCOUNT_STATE_MACHINE_ARN"),
	})

	if err != nil {
		reportError(err, false)
		respondWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
