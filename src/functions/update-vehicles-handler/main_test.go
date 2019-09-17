package main

import (
	"os"
	"testing"

	"github.com/maddiesch/automatic-reminders/auto"
)

func TestMain(m *testing.M) {
	os.Exit(auto.SetupAndRunTestSuite(m))
}
