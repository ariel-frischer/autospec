package update

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input   string
		want    *Version
		wantErr bool
	}{
		"valid with v prefix": {
			input: "v0.6.1",
			want:  &Version{Major: 0, Minor: 6, Patch: 1, Raw: "v0.6.1"},
		},
		"valid without v prefix": {
			input: "1.2.3",
			want:  &Version{Major: 1, Minor: 2, Patch: 3, Raw: "1.2.3"},
		},
		"major version only high": {
			input: "v10.0.0",
			want:  &Version{Major: 10, Minor: 0, Patch: 0, Raw: "v10.0.0"},
		},
		"all zeros": {
			input: "v0.0.0",
			want:  &Version{Major: 0, Minor: 0, Patch: 0, Raw: "v0.0.0"},
		},
		"dev build": {
			input: "dev",
			want:  &Version{Raw: "dev"},
		},
		"empty string": {
			input: "",
			want:  &Version{Raw: ""},
		},
		"invalid - too few parts": {
			input:   "v1.2",
			wantErr: true,
		},
		"invalid - too many parts": {
			input:   "v1.2.3.4",
			wantErr: true,
		},
		"invalid - non-numeric major": {
			input:   "vX.2.3",
			wantErr: true,
		},
		"invalid - non-numeric minor": {
			input:   "v1.X.3",
			wantErr: true,
		},
		"invalid - non-numeric patch": {
			input:   "v1.2.X",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseVersion(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.Major, got.Major)
			assert.Equal(t, tt.want.Minor, got.Minor)
			assert.Equal(t, tt.want.Patch, got.Patch)
			assert.Equal(t, tt.want.Raw, got.Raw)
		})
	}
}

func TestVersion_IsDev(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		version *Version
		want    bool
	}{
		"dev build": {
			version: &Version{Raw: "dev"},
			want:    true,
		},
		"empty raw": {
			version: &Version{Raw: ""},
			want:    true,
		},
		"release version": {
			version: &Version{Major: 0, Minor: 6, Patch: 1, Raw: "v0.6.1"},
			want:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.version.IsDev())
		})
	}
}

func TestVersion_String(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		version *Version
		want    string
	}{
		"release version": {
			version: &Version{Major: 0, Minor: 6, Patch: 1, Raw: "v0.6.1"},
			want:    "v0.6.1",
		},
		"dev build": {
			version: &Version{Raw: "dev"},
			want:    "dev",
		},
		"zero version": {
			version: &Version{Major: 0, Minor: 0, Patch: 0, Raw: "v0.0.0"},
			want:    "v0.0.0",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.version.String())
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		v1   *Version
		v2   *Version
		want int
	}{
		"equal versions": {
			v1:   &Version{Major: 0, Minor: 6, Patch: 1, Raw: "v0.6.1"},
			v2:   &Version{Major: 0, Minor: 6, Patch: 1, Raw: "v0.6.1"},
			want: 0,
		},
		"v1 major greater": {
			v1:   &Version{Major: 1, Minor: 0, Patch: 0, Raw: "v1.0.0"},
			v2:   &Version{Major: 0, Minor: 9, Patch: 9, Raw: "v0.9.9"},
			want: 1,
		},
		"v1 major less": {
			v1:   &Version{Major: 0, Minor: 9, Patch: 9, Raw: "v0.9.9"},
			v2:   &Version{Major: 1, Minor: 0, Patch: 0, Raw: "v1.0.0"},
			want: -1,
		},
		"v1 minor greater": {
			v1:   &Version{Major: 0, Minor: 7, Patch: 0, Raw: "v0.7.0"},
			v2:   &Version{Major: 0, Minor: 6, Patch: 9, Raw: "v0.6.9"},
			want: 1,
		},
		"v1 minor less": {
			v1:   &Version{Major: 0, Minor: 5, Patch: 9, Raw: "v0.5.9"},
			v2:   &Version{Major: 0, Minor: 6, Patch: 0, Raw: "v0.6.0"},
			want: -1,
		},
		"v1 patch greater": {
			v1:   &Version{Major: 0, Minor: 6, Patch: 2, Raw: "v0.6.2"},
			v2:   &Version{Major: 0, Minor: 6, Patch: 1, Raw: "v0.6.1"},
			want: 1,
		},
		"v1 patch less": {
			v1:   &Version{Major: 0, Minor: 6, Patch: 0, Raw: "v0.6.0"},
			v2:   &Version{Major: 0, Minor: 6, Patch: 1, Raw: "v0.6.1"},
			want: -1,
		},
		"dev vs dev": {
			v1:   &Version{Raw: "dev"},
			v2:   &Version{Raw: "dev"},
			want: 0,
		},
		"dev vs release - dev is less": {
			v1:   &Version{Raw: "dev"},
			v2:   &Version{Major: 0, Minor: 0, Patch: 1, Raw: "v0.0.1"},
			want: -1,
		},
		"release vs dev - release is greater": {
			v1:   &Version{Major: 0, Minor: 0, Patch: 1, Raw: "v0.0.1"},
			v2:   &Version{Raw: "dev"},
			want: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.v1.Compare(tt.v2))
		})
	}
}

func TestVersion_IsNewerThan(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		v1   *Version
		v2   *Version
		want bool
	}{
		"newer version": {
			v1:   &Version{Major: 0, Minor: 7, Patch: 0, Raw: "v0.7.0"},
			v2:   &Version{Major: 0, Minor: 6, Patch: 1, Raw: "v0.6.1"},
			want: true,
		},
		"older version": {
			v1:   &Version{Major: 0, Minor: 6, Patch: 0, Raw: "v0.6.0"},
			v2:   &Version{Major: 0, Minor: 6, Patch: 1, Raw: "v0.6.1"},
			want: false,
		},
		"same version": {
			v1:   &Version{Major: 0, Minor: 6, Patch: 1, Raw: "v0.6.1"},
			v2:   &Version{Major: 0, Minor: 6, Patch: 1, Raw: "v0.6.1"},
			want: false,
		},
		"release vs dev": {
			v1:   &Version{Major: 0, Minor: 0, Patch: 1, Raw: "v0.0.1"},
			v2:   &Version{Raw: "dev"},
			want: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.v1.IsNewerThan(tt.v2))
		})
	}
}
