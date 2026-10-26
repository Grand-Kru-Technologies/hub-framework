package remoteforward

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinForwardURL_EmptyRequestPathKeepsPrefixSlash(t *testing.T) {
	got := JoinForwardURL("http://python:5000", "/apply/exampleTaskPython/", "")
	assert.Equal(t, "http://python:5000/apply/exampleTaskPython/", got)
}
