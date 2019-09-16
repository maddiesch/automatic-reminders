package main

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticationFlow(t *testing.T) {
	t.Run("golden path", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			switch r.URL.Path {
			case "/oauth/access_token/":
				w.Write([]byte(`{"access_token":"7c503287a78fb78b278c9000b77720477e000000","scope":"scope:offline scope:public scope:trip scope:user:profile scope:vehicle:profile","expires_in":2591999,"refresh_token":"b1729476bc5e36c0000009ff6bbe0421d8000000","token_type":"bearer","user":{"id":"U_cfdca00556000000","sid":"U_cfdca005564e0000"},"user_id":"U_cfdca00556000000"}`))
			case "/user/U_cfdca00556000000":
				w.Write([]byte(`{"id":"U_cfdca00556000000","url":"https://api.automatic.com/user/U_cfdca00556000000/","username":"test@email.test","first_name":"Testy","last_name":"Mc Testerson","email":"test@email.test","email_verified":true}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}

		t.Run("first authentication", func(t *testing.T) {
			var state string

			t.Run("creates a redirect url", func(t *testing.T) {
				redirect, err := integrationCreateAutomaticAuthenticationURL()
				require.NoError(t, err)

				uri, err := url.Parse(redirect)
				require.NoError(t, err)

				state = uri.Query().Get("state")
			})

			withStubbedRequests(t, handler, func(t *testing.T) {
				t.Run("handles a response code", func(t *testing.T) {
					_, err := integrationAutomaticAuthCallback("fake-code", state)

					assert.NoError(t, err)
				})
			})
		})

		t.Run("second authentication", func(t *testing.T) {
			var state string

			t.Run("creates a redirect url", func(t *testing.T) {
				redirect, err := integrationCreateAutomaticAuthenticationURL()
				require.NoError(t, err)

				uri, err := url.Parse(redirect)
				require.NoError(t, err)

				state = uri.Query().Get("state")
			})

			withStubbedRequests(t, handler, func(t *testing.T) {
				t.Run("handles a response code", func(t *testing.T) {
					_, err := integrationAutomaticAuthCallback("fake-code", state)

					assert.NoError(t, err)
				})
			})
		})
	})
}
