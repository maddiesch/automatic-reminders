package auto

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tripResponseBody = `{"_metadata":{"count":6136,"previous":"https://api.automatic.com/trip/?started_at__gte=1325376000&started_at__lte=1568752060&page=1&cursor=db9d3520-dd00-4ee1-8d74-5cf05ec77c38","next":"https://api.automatic.com/trip/?started_at__gte=1325376000&started_at__lte=1568752060&page=3&cursor=478bcc28-61b5-4468-85a0-e276934bcd37"},"results":[{"id":"T_68283fd78200000","url":"https://api.automatic.com/trip/T_68283fd782ddda5a/","driver":"https://api.automatic.com/user/U_cfdca00556000000/","vehicle":"https://api.automatic.com/vehicle/C_fd654624e3000000/","duration_s":415.0,"distance_m":2158.1,"started_at":"2019-09-15T16:32:48.200000Z","ended_at":"2019-09-15T16:39:43.200000Z","start_timezone":"America/Denver","end_timezone":"America/Denver","tags":[],"idling_time_s":0,"user":"https://api.automatic.com/user/U_cfdca00556000000/"}]}`
)

func TestTrip(t *testing.T) {
	t.Run("UpdateTripsForAccount", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(tripResponseBody))
		}

		account, token := CreateFakeAccountAndToken()

		WithStubbedRequests(t, handler, func(t *testing.T) {
			t.Run("golden path", func(t *testing.T) {
				uri, err := url.Parse("https://api.automatic.com/trip/?started_at__gte=1325376000&started_at__lte=1568752060")

				require.NoError(t, err)

				next, err := UpdateTripsForAccount(account, token, uri)

				require.NoError(t, err)

				assert.Equal(t, "https://api.automatic.com/trip/?started_at__gte=1325376000&started_at__lte=1568752060&page=3&cursor=478bcc28-61b5-4468-85a0-e276934bcd37", next.String())
			})
		})
	})
}
