package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gin-gonic/gin"
	"github.com/maddiesch/automatic-reminders/auto"
	"github.com/maddiesch/serverless"
	"github.com/maddiesch/serverless/sam"
)

func main() {
	lambda.Start(serverless.LambdaHandler(func() {
		if !sam.IsLocal() && !auto.IsTest() {
			gin.SetMode(gin.ReleaseMode)
		}

		serverless.SharedApp().ConfigureGin(func(e *gin.Engine) {
			installRoutes(e)
		})
	}))
}
