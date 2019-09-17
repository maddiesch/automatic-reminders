package auto

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/maddiesch/serverless"
)

type httpSendFunction func(*http.Client, *http.Request) (*http.Response, error)

type httpStack struct {
	client *http.Client
	sender httpSendFunction
}

var (
	httpStackInstance      *httpStack
	httpStackInstanceSetup sync.Once
)

func getHTTPStack() *httpStack {
	httpStackInstanceSetup.Do(func() {
		httpStackInstance = &httpStack{
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
			sender: func(c *http.Client, r *http.Request) (*http.Response, error) {
				return c.Do(r)
			},
		}
	})
	return httpStackInstance
}

// SendRequest performs a request through the auto library's http client
func SendRequest(r *http.Request) (*http.Response, error) {
	serverless.GetLogger().Printf("SUB-REQUEST: [%s] %s", r.Method, r.URL)

	response, err := getHTTPStack().sender(getHTTPStack().client, r)
	if err != nil {
		return response, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response, fmt.Errorf("invalid HTTP response: %s", response.Status)
	}
	return response, err
}
