// +build testing

package auto

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/maddiesch/serverless"
	"github.com/segmentio/ksuid"
)

const (
	TestingResponseAccount  = `{"id":"U_cfdca00556000000","url":"https://api.automatic.com/user/U_cfdca00556000000/","username":"test@email.test","first_name":"Testy","last_name":"Mc Testerson","email":"test@email.test","email_verified":true}`
	TestingResponseVehicles = `{"_metadata":{"count":2,"next":null,"previous":null},"results":[{"active_dtcs":[],"battery_voltage":12.842,"created_at":"2017-09-22T16:38:19.994000Z","display_name":null,"fuel_grade":"diesel","fuel_level_percent":46.0,"id":"C_fd654624e3000000","make":"Ram","model":"2500","submodel":"Big Horn","updated_at":"2018-07-26T18:35:12.003000Z","url":"https://api.automatic.com/vehicle/C_fd654624e3000000/","year":2017},{"active_dtcs":null,"battery_voltage":12.535,"created_at":"2016-08-25T00:45:22.787000Z","display_name":null,"fuel_grade":"regular","fuel_level_percent":16.0,"id":"C_4a10259f5a000000","make":"Chevrolet","model":"Silverado 1500","submodel":"LT","updated_at":"2017-09-22T16:38:57.545000Z","url":"https://api.automatic.com/vehicle/C_4a10259f5a000000/","year":2016}]}`
)

func CreateFakeAccountAndToken() (*Account, *AutomaticAccessToken) {
	account := &Account{
		ID:          fmt.Sprintf("auid:%s", ksuid.New().String()),
		FirstName:   "Testy",
		LastName:    "Mc Testerson",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		AutomaticID: "U_cfdca00556000000",
	}
	token := &AutomaticAccessToken{
		ID:           ksuid.New().String(),
		IssuedAt:     time.Now(),
		UserID:       "U_cfdca00556000000",
		AccessToken:  "fake-access-token",
		ExpiresIn:    100000,
		Scope:        "scope:offline scope:public scope:trip scope:user:profile scope:vehicle:profile",
		RefreshToken: "fake-refresh-token",
		TokenType:    "bearer",
	}

	err := WriteAccountWithToken(account, token, nil)
	if err != nil {
		panic(err)
	}

	return account, token
}

// SetupAndRunTestSuite handles all the preperation
func SetupAndRunTestSuite(m *testing.M) int {
	serverless.GetLogger().SetOutput(ioutil.Discard)

	err := SetupTestingEnvironment()
	if err != nil {
		panic(err)
	}

	return m.Run()
}

// SetupTestingEnvironment performs shared test setup
func SetupTestingEnvironment() error {
	type envContent struct {
		Main map[string]string `json:"ApiFunctionHandler"`
	}

	{ // Setup the environment from the env file
		data, err := ioutil.ReadFile(os.Getenv("TESTING_ENV_FILE"))
		if err != nil {
			return err
		}
		content := envContent{}
		err = json.Unmarshal(data, &content)
		if err != nil {
			return err
		}

		for key, value := range content.Main {
			err := os.Setenv(key, value)
			if err != nil {
				return err
			}
		}
	}

	{ // Setup environment overrides. MUST happen after the env file
		os.Setenv("AWS_SAM_LOCAL", "true")
		os.Setenv("RETURN_FAKE_SECRETS", "true")
		os.Setenv("DYNAMODB_TABLE_NAME", os.Getenv("TEST_TABLE_NAME"))
		os.Setenv("AUTO_TEST", "true")
	}

	return nil
}

// WithStubbedRequests stubs all http requests made using the auto SendRequest method
func WithStubbedRequests(t *testing.T, handler http.HandlerFunc, fn func(t *testing.T)) {
	fakeServer := httptest.NewServer(handler)
	testURL, _ := url.ParseRequestURI(fakeServer.URL)

	originalClient := getHTTPStack().client
	originalSender := getHTTPStack().sender

	defer func(c *http.Client, s httpSendFunction) {
		getHTTPStack().client = c
		getHTTPStack().sender = originalSender
	}(originalClient, originalSender)
	defer fakeServer.Close()

	getHTTPStack().client = fakeServer.Client()
	getHTTPStack().sender = func(c *http.Client, r *http.Request) (*http.Response, error) {
		r.URL.Scheme = "http"
		r.URL.Host = testURL.Host
		return c.Do(r)
	}

	t.Run("with subbed requests", fn)
}
