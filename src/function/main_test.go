package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventGreeting(t *testing.T) {
	t.Run("given a greeting and name", func(t *testing.T) {
		event := Event{Greeting: "Hello", Name: "World"}
		greeting := greeting(event)

		assert.Equal(t, greeting, "Hello World!")
	})
}
