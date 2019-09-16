package auto

import "os"

// IsTest returns true if running the test suite
func IsTest() bool {
	return os.Getenv("AUTO_TEST") == "true"
}
