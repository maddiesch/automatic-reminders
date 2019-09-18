package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func installRoutes(e *gin.Engine) {
	e.Handle("GET", "/", redirectToV1Handler)

	v1 := e.Group("/v1")
	{
		v1.Handle("GET", "/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })

		integration := v1.Group("/integration")
		{
			automatic := integration.Group("/automatic")
			{
				automatic.Handle("GET", "/authenticate", integrationAutomaticAuthHandler)
				automatic.Handle("GET", "/authenticate/callback", integrationAutomaticAuthCallbackHandler)
				automatic.Handle("POST", "/hookshot", integrationAutomaticHookshotHandler)
			}
		}

		account := v1.Group("/account", Authenticate)
		{
			account.Handle("GET", "", getAccountHandler)
			account.Handle("POST", "", postAccountUpdateHandler)

			account.Handle("GET", "/vehicles", getVehiclesHandler)
		}
	}
}

func redirectToV1Handler(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/v1")
}
