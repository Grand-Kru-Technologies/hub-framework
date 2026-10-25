package remoteforward

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePathPrefix(t *testing.T) {
	assert.Equal(t, "", NormalizePathPrefix(""))
	assert.Equal(t, "/api/", NormalizePathPrefix("api"))
	assert.Equal(t, "/api/", NormalizePathPrefix("/api"))
	assert.Equal(t, "/api/", NormalizePathPrefix("api/"))
	assert.Equal(t, "/api/", NormalizePathPrefix("/api/"))
}
