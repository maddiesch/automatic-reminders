module github.com/maddiesch/automatic-reminders/functions/update-vehicles-handler

go 1.13

require (
	github.com/aws/aws-lambda-go v1.13.2
	github.com/aws/aws-sdk-go v1.23.21
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gin-gonic/gin v1.3.0
	github.com/jarcoal/httpmock v1.0.4
	github.com/maddiesch/automatic-reminders/auto v0.0.0
	github.com/maddiesch/serverless v0.1.0
	github.com/segmentio/ksuid v1.0.2
	github.com/stretchr/testify v1.4.0
)

replace github.com/maddiesch/automatic-reminders/auto v0.0.0 => ../../auto
