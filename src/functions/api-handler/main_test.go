package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

type envContent struct {
	Main map[string]string `json:"ApiFunctionHandler"`
}

func TestMain(m *testing.M) {
	os.Exit(runTestSuite(m))
}

func runTestSuite(m *testing.M) int {
	{ // Setup the environment from the env file
		data, err := ioutil.ReadFile(os.Getenv("TESTING_ENV_FILE"))
		if err != nil {
			panic(err)
		}
		content := envContent{}
		err = json.Unmarshal(data, &content)
		if err != nil {
			panic(err)
		}

		for key, value := range content.Main {
			err := os.Setenv(key, value)
			if err != nil {
				panic(err)
			}
		}
	}

	{ // Setup environment overrides. MUST happen after the env file
		os.Setenv("AWS_SAM_LOCAL", "true")
		os.Setenv("RETURN_FAKE_SECRETS", "true")
		os.Setenv("DYNAMODB_TABLE_NAME", os.Getenv("TEST_TABLE_NAME"))
		os.Setenv("AUTO_TEST", "true")
	}

	return m.Run()
}

func withStubbedRequests(t *testing.T, handler http.HandlerFunc, fn func(t *testing.T)) {
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
