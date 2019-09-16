package auto

import (
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/maddiesch/serverless"
	"github.com/maddiesch/serverless/amazon"
)

// Secret contains the app secrets
type Secret struct {
	ClientID     string
	ClientSecret string
	Signing      string
}

var (
	secretsInstance Secret
	secretsSetup    sync.Once
)

// Secrets returns the shared secrets for the app
func Secrets() Secret {
	secretsSetup.Do(func() {
		if os.Getenv("RETURN_FAKE_SECRETS") == "true" {
			secretsInstance = Secret{
				ClientID:     "fake-client-id",
				ClientSecret: "fake-client-secret",
				Signing:      "super-sekret",
			}
			return
		}

		client := ssm.New(amazon.BaseSession())

		output, err := client.GetParameters(&ssm.GetParametersInput{
			Names: aws.StringSlice([]string{
				os.Getenv("SECRETS_CLIENT_ID_PARAMETER_NAME"),
				os.Getenv("SECRETS_CLIENT_SECRET_PARAMETER_NAME"),
				os.Getenv("SECRETS_PRODUCTION_SIGNING_SECRET_PARAMETER_NAME"),
			}),
		})
		if err != nil {
			serverless.GetLogger().Fatal(err)
		}

		secrets := Secret{}

		for _, param := range output.Parameters {
			switch aws.StringValue(param.Name) {
			case os.Getenv("SECRETS_CLIENT_ID_PARAMETER_NAME"):
				secrets.ClientID = aws.StringValue(param.Value)
			case os.Getenv("SECRETS_CLIENT_SECRET_PARAMETER_NAME"):
				secrets.ClientSecret = aws.StringValue(param.Value)
			case os.Getenv("SECRETS_PRODUCTION_SIGNING_SECRET_PARAMETER_NAME"):
				secrets.Signing = aws.StringValue(param.Value)
			default:
			}
		}

		secretsInstance = secrets
	})
	return secretsInstance
}
