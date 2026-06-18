package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestValidateDomains(t *testing.T) {
	a := require.New(t)

	testCases := []struct {
		domain  string
		wantErr bool
	}{
		{
			domain:  "",
			wantErr: true,
		},
		{
			domain:  "hello@world.com",
			wantErr: true,
		},
		{
			domain:  "BYTEBASE.COM",
			wantErr: true,
		},
		{
			domain:  "bytebase.com",
			wantErr: false,
		},
		{
			domain:  "x.y",
			wantErr: true,
		},
		{
			domain:  "abc.xyz",
			wantErr: false,
		},
		{
			domain:  "gmail.com",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		err := validateDomains([]string{tc.domain})
		if tc.wantErr {
			a.Error(err, tc.domain)
		} else {
			a.NoError(err, tc.domain)
		}
	}
}

func TestValidateSQLEditorCustomTheme(t *testing.T) {
	// The server validates shape, not the frontend token vocabulary: a non-empty
	// map whose values are "r g b" triples. A representative subset is enough.
	tokens := func() map[string]string {
		return map[string]string{
			"--color-background": "1 2 3",
			"--color-accent":     "4 5 6",
		}
	}
	with := func(m map[string]string) *storepb.SQLEditorThemeSetting {
		return &storepb.SQLEditorThemeSetting{Id: "u1", Name: "Brand", MonacoBase: "vs-dark", Tokens: m}
	}
	bad := func(k, v string) *storepb.SQLEditorThemeSetting {
		m := tokens()
		m[k] = v
		return with(m)
	}
	cases := []struct {
		name    string
		theme   *storepb.SQLEditorThemeSetting
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"valid", with(tokens()), false},
		// Backend is vocabulary-agnostic: any key with a valid triple is fine.
		{"arbitrary keys ok", with(map[string]string{"--anything": "7 8 9"}), false},
		{"empty tokens", with(map[string]string{}), true},
		{"bad triple", bad("--color-accent", "300 0 0"), true},
		{"empty id", &storepb.SQLEditorThemeSetting{Id: "", Name: "Brand", MonacoBase: "vs-dark", Tokens: tokens()}, true},
		{"empty name", &storepb.SQLEditorThemeSetting{Id: "u1", Name: "", MonacoBase: "vs-dark", Tokens: tokens()}, true},
		{"empty base", &storepb.SQLEditorThemeSetting{Id: "u1", Name: "Brand", MonacoBase: "", Tokens: tokens()}, true},
		// monaco_base is a frontend-owned theme id; any non-empty value is fine.
		{"non-builtin base ok", &storepb.SQLEditorThemeSetting{Id: "u1", Name: "Brand", MonacoBase: "Dark Modern", Tokens: tokens()}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSQLEditorCustomTheme(tc.theme)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
