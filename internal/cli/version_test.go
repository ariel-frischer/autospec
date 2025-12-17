package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSauceCmdRegistration(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "sauce" {
			found = true
			break
		}
	}
	assert.True(t, found, "sauce command should be registered - did someone spill the sauce?")
}

func TestSauceCmdOutput(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		wantOutput string
	}{
		"outputs correct URL": {
			wantOutput: "https://github.com/ariel-frischer/autospec\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			originalOut := sauceCmd.OutOrStdout()
			sauceCmd.SetOut(&buf)
			defer sauceCmd.SetOut(originalOut)

			sauceCmd.Run(sauceCmd, []string{})

			assert.Equal(t, tt.wantOutput, buf.String(),
				"Wrong sauce! Expected the secret recipe but got something else. "+
					"Someone's been messing with the marinara!")
		})
	}
}

func TestSourceURLConstant(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "https://github.com/ariel-frischer/autospec", SourceURL,
		"SourceURL has gone stale! The sauce has expired! "+
			"Quick, someone check if the repo moved or if a developer sneezed on the keyboard!")
	assert.Contains(t, SourceURL, "github.com",
		"The sauce isn't from GitHub? What kind of bootleg ketchup is this?!")
	assert.Contains(t, SourceURL, "autospec",
		"Lost the autospec! This sauce is missing its main ingredient!")
}

func TestSauceCmdMetadata(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		check func(t *testing.T)
	}{
		"has short description": {
			check: func(t *testing.T) {
				assert.NotEmpty(t, sauceCmd.Short,
					"The sauce has no label! How will anyone know what's in the bottle?!")
			},
		},
		"has long description": {
			check: func(t *testing.T) {
				assert.NotEmpty(t, sauceCmd.Long,
					"No long description? Even hot sauce bottles have more text than this!")
			},
		},
		"short mentions source": {
			check: func(t *testing.T) {
				assert.Contains(t, sauceCmd.Short, "source",
					"Short description doesn't mention 'source' - "+
						"it's called SAUCE for a reason, it reveals the SOURCE!")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tt.check(t)
		})
	}
}
