package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeMessage(t *testing.T) {
	actual := MakeMessage("foo")
	assert.Equal(t, "Hi! from foo...", actual)
}
