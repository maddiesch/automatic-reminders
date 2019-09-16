package main

import (
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gin-gonic/gin"
	"github.com/maddiesch/serverless"
)

func main() {
	lambda.Start(serverless.LambdaHandler(func() {
		serverless.SharedApp().ConfigureGin(func(e *gin.Engine) {
			v1 := e.Group("/v1")
			{
				v1.Handle("GET", "/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })

				integration := v1.Group("/integration")
				{
					integration.Handle("GET", "/automatic/authenticate", integrationAutomaticAuthHandler)
					integration.Handle("GET", "/automatic/authenticate/callback", integrationAutomaticAuthCallbackHandler)
				}

				private := v1.Group("/private", Authenticate)
				{
					private.Handle("GET", "/", getAccountHandler)
				}
			}
		})
	}))
}
