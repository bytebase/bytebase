package v1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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
	// map whose values are #rrggbb hex colors. A representative subset is enough.
	tokens := func() map[string]string {
		return map[string]string{
			"--color-background": "#010203",
			"--color-accent":     "#aabbcc",
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
		// Backend is vocabulary-agnostic: any key with a valid hex color is fine.
		{"arbitrary keys ok", with(map[string]string{"--anything": "#070809"}), false},
		{"empty tokens", with(map[string]string{}), true},
		{"rgb triple rejected", bad("--color-accent", "1 2 3"), true},
		{"missing hash rejected", bad("--color-accent", "aabbcc"), true},
		{"short hex rejected", bad("--color-accent", "#abc"), true},
		{"invalid hex rejected", bad("--color-accent", "#gggggg"), true},
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

func TestValidateAnnouncementTheme(t *testing.T) {
	cases := []struct {
		name    string
		theme   *storepb.WorkspaceProfileSetting_Announcement_AnnouncementTheme
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"valid", &storepb.WorkspaceProfileSetting_Announcement_AnnouncementTheme{Background: "#2563eb", Text: "#ffffff"}, false},
		{"empty background", &storepb.WorkspaceProfileSetting_Announcement_AnnouncementTheme{Background: "", Text: "#ffffff"}, true},
		{"empty text", &storepb.WorkspaceProfileSetting_Announcement_AnnouncementTheme{Background: "#2563eb", Text: ""}, true},
		{"rgb background rejected", &storepb.WorkspaceProfileSetting_Announcement_AnnouncementTheme{Background: "37 99 235", Text: "#ffffff"}, true},
		{"short text rejected", &storepb.WorkspaceProfileSetting_Announcement_AnnouncementTheme{Background: "#2563eb", Text: "#fff"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAnnouncementTheme(tc.theme)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateEnvironmentsColor(t *testing.T) {
	service := &SettingService{}
	env := func(color string) *v1pb.EnvironmentSetting_Environment {
		return &v1pb.EnvironmentSetting_Environment{
			Id:    "test",
			Title: "Test",
			Color: color,
		}
	}
	cases := []struct {
		name    string
		color   string
		wantErr bool
	}{
		{"empty ok", "", false},
		{"valid", "#4f46e5", false},
		{"uppercase valid", "#AABBCC", false},
		{"rgb rejected", "79 70 229", true},
		{"missing hash rejected", "4f46e5", true},
		{"short hex rejected", "#fff", true},
		{"invalid hex rejected", "#zzzzzz", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.validateEnvironments(context.Background(), "workspaces/default", []*v1pb.EnvironmentSetting_Environment{env(tc.color)})
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
