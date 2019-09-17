package auto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindAccount(t *testing.T) {
	t.Run("with and existing account", func(t *testing.T) {
		account, _ := CreateFakeAccountAndToken()

		found, err := FindAccount(account.ID)

		assert.NoError(t, err)

		assert.Equal(t, account.ID, found.ID)
	})
}
