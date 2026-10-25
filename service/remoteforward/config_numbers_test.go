package remoteforward

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCoerceInt_AcceptsYAMLIntegerTypes(t *testing.T) {
	n, ok := CoerceInt(3)
	assert.True(t, ok)
	assert.Equal(t, 3, n)

	n, ok = CoerceInt(int64(5))
	assert.True(t, ok)
	assert.Equal(t, 5, n)
}

func TestCoercePositiveSeconds(t *testing.T) {
	d, ok := CoercePositiveSeconds(15)
	assert.True(t, ok)
	assert.Equal(t, 15*time.Second, d)
}
