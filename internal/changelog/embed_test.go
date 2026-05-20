package changelog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbedded(t *testing.T) {
	content := Embedded()
	assert.NotEmpty(t, content, "embedded changelog should not be empty")
	assert.Contains(t, string(content), "project: autospec", "embedded content should contain project field")
}

func TestLoadEmbedded(t *testing.T) {
	tests := map[string]struct {
		assertion func(t *testing.T, cl *Changelog, err error)
	}{
		"loads without error": {
			assertion: func(t *testing.T, cl *Changelog, err error) {
				require.NoError(t, err)
				assert.NotNil(t, cl)
			},
		},
		"has correct project": {
			assertion: func(t *testing.T, cl *Changelog, err error) {
				require.NoError(t, err)
				assert.Equal(t, "autospec", cl.Project)
			},
		},
		"has versions": {
			assertion: func(t *testing.T, cl *Changelog, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, cl.Versions, "changelog should have at least one version")
			},
		},
		"latest release has changes": {
			assertion: func(t *testing.T, cl *Changelog, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, cl.Versions)
				latest := cl.GetLatestRelease()
				require.NotNil(t, latest)
				assert.False(t, latest.Changes.IsEmpty(),
					"latest released version should have at least one change entry")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cl, err := LoadEmbedded()
			tt.assertion(t, cl, err)
		})
	}
}
