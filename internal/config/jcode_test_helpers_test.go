package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yamlv3 "gopkg.in/yaml.v3"
)

func jcodeConfigFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	require.True(t, ok, "locate config fixture source")
	return filepath.Join(filepath.Dir(source), "testdata", name)
}

func requireJcodeFixture(t *testing.T, name string) JcodeConfig {
	t.Helper()
	data, err := os.ReadFile(jcodeConfigFixturePath(t, name))
	require.NoError(t, err)
	var document struct {
		Jcode JcodeConfig `yaml:"jcode"`
	}
	require.NoError(t, yamlv3.Unmarshal(data, &document))
	require.NoError(t, validateJcodeConfig(document.Jcode, jcodeConfigFixturePath(t, name)))
	return document.Jcode
}

func requireJcodeFixtureError(t *testing.T, name string) error {
	t.Helper()
	data, err := os.ReadFile(jcodeConfigFixturePath(t, name))
	require.NoError(t, err)
	var document struct {
		Jcode JcodeConfig `yaml:"jcode"`
	}
	require.NoError(t, yamlv3.Unmarshal(data, &document))
	return validateJcodeConfig(document.Jcode, jcodeConfigFixturePath(t, name))
}

func assertJcodeRunner(t *testing.T, fixture string, want JcodeRunner) {
	t.Helper()
	assert.Equal(t, want, requireJcodeFixture(t, fixture).EffectiveRunner())
}
