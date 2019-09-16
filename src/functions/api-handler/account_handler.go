package main

import (
	"errors"
	"net/http"

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

func getAccount(accountID string) (*auto.Account, error) {
	return nil, errors.New("WIP")
}
