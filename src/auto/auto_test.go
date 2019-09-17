package auto

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(SetupAndRunTestSuite(m))
}
