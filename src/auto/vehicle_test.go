package auto

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVehicle(t *testing.T) {
	account, token := CreateFakeAccountAndToken()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(TestingResponseVehicles))
	}

	t.Run("UpdateVehiclesForAccount", func(t *testing.T) {
		t.Run("golden path", func(t *testing.T) {

			WithStubbedRequests(t, handler, func(t *testing.T) {
				t.Run("update", func(t *testing.T) {
					err := UpdateVehiclesForAccount(account, token)

					assert.NoError(t, err)
				})
			})
		})
	})

	t.Run("VehiclesForAccount", func(t *testing.T) {
		WithStubbedRequests(t, handler, func(t *testing.T) {
			err := UpdateVehiclesForAccount(account, token)

			require.NoError(t, err)
		})

		t.Run("golden path", func(t *testing.T) {
			_, err := VehiclesForAccount(account)

			assert.NoError(t, err)
		})
	})
}
