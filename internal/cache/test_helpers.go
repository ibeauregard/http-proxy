package cache

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func clearIndex(t *testing.T, keys ...string) {
	for _, key := range keys {
		index.remove(key)
	}
	assert.True(t, len(index.getMap()) == 0)
}
