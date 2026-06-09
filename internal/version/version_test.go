package version_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/t0mer/galactica/internal/version"
)

func TestConstants(t *testing.T) {
	assert.Equal(t, "Galactica", version.AppName)
	assert.Equal(t, "galactica", version.BinaryName)
	assert.Equal(t, "GALACTICA", version.EnvPrefix)
}

func TestVersionVarsHaveDefaults(t *testing.T) {
	assert.NotEmpty(t, version.Version)
	assert.NotEmpty(t, version.Commit)
	assert.NotEmpty(t, version.Date)
}
