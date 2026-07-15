package v1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	colorpb "google.golang.org/genproto/googleapis/type/color"
	"google.golang.org/protobuf/types/known/wrapperspb"

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

func TestValidateApprovalTemplate(t *testing.T) {
	cases := []struct {
		name     string
		template *v1pb.ApprovalTemplate
		wantErr  bool
	}{
		{
			name:    "nil template rejected",
			wantErr: true,
		},
		{
			name:     "nil flow rejected",
			template: &v1pb.ApprovalTemplate{},
			wantErr:  true,
		},
		{
			name: "empty roles allowed for skipped approval",
			template: &v1pb.ApprovalTemplate{
				Flow: &v1pb.ApprovalFlow{},
			},
		},
		{
			name: "valid role accepted",
			template: &v1pb.ApprovalTemplate{
				Flow: &v1pb.ApprovalFlow{Roles: []string{"roles/projectOwner"}},
			},
		},
		{
			name: "empty role rejected",
			template: &v1pb.ApprovalTemplate{
				Flow: &v1pb.ApprovalFlow{Roles: []string{""}},
			},
			wantErr: true,
		},
		{
			name: "blank role rejected",
			template: &v1pb.ApprovalTemplate{
				Flow: &v1pb.ApprovalFlow{Roles: []string{" "}},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateApprovalTemplate(tc.template)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateSQLEditorCustomTheme(t *testing.T) {
	// The server validates shape, not the frontend token vocabulary: a non-empty
	// map whose values are google.type.Color messages. A representative subset is enough.
	tokens := func() map[string]*colorpb.Color {
		return map[string]*colorpb.Color{
			"--color-background": colorValue(0.01, 0.02, 0.03),
			"--color-accent":     colorValue(0.6, 0.7, 0.8),
		}
	}
	with := func(m map[string]*colorpb.Color) *storepb.SQLEditorThemeSetting {
		return &storepb.SQLEditorThemeSetting{Id: "u1", Name: "Brand", MonacoBase: "vs-dark", Tokens: m}
	}
	bad := func(k string, v *colorpb.Color) *storepb.SQLEditorThemeSetting {
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
		// Backend is vocabulary-agnostic: any key with a valid Color is fine.
		{"arbitrary keys ok", with(map[string]*colorpb.Color{"--anything": colorValue(0.7, 0.8, 0.9)}), false},
		{"empty tokens", with(map[string]*colorpb.Color{}), true},
		{"nil color rejected", bad("--color-accent", nil), true},
		{"red below range rejected", bad("--color-accent", colorValue(-0.1, 0, 0)), true},
		{"green above range rejected", bad("--color-accent", colorValue(0, 1.1, 0)), true},
		{"alpha below one rejected", bad("--color-accent", colorWithAlpha(0, 0, 0, 0.5)), true},
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

func colorValue(red, green, blue float32) *colorpb.Color {
	return &colorpb.Color{Red: red, Green: green, Blue: blue}
}

func colorWithAlpha(red, green, blue, alpha float32) *colorpb.Color {
	return &colorpb.Color{Red: red, Green: green, Blue: blue, Alpha: wrapperspb.Float(alpha)}
}

func TestValidateAnnouncementTheme(t *testing.T) {
	cases := []struct {
		name    string
		theme   *storepb.WorkspaceProfileSetting_Announcement_AnnouncementTheme
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"valid", &storepb.WorkspaceProfileSetting_Announcement_AnnouncementTheme{Background: colorValue(0.2, 0.3, 0.4), Text: colorValue(1, 1, 1)}, false},
		{"nil background", &storepb.WorkspaceProfileSetting_Announcement_AnnouncementTheme{Background: nil, Text: colorValue(1, 1, 1)}, true},
		{"nil text", &storepb.WorkspaceProfileSetting_Announcement_AnnouncementTheme{Background: colorValue(0.2, 0.3, 0.4), Text: nil}, true},
		{"background out of range", &storepb.WorkspaceProfileSetting_Announcement_AnnouncementTheme{Background: colorValue(1.1, 0.3, 0.4), Text: colorValue(1, 1, 1)}, true},
		{"alpha below one", &storepb.WorkspaceProfileSetting_Announcement_AnnouncementTheme{Background: colorWithAlpha(0.2, 0.3, 0.4, 0.5), Text: colorValue(1, 1, 1)}, true},
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
	env := func(color *colorpb.Color) *v1pb.EnvironmentSetting_Environment {
		return &v1pb.EnvironmentSetting_Environment{
			Id:    "test",
			Title: "Test",
			Color: color,
		}
	}
	cases := []struct {
		name    string
		color   *colorpb.Color
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"valid", colorValue(0.31, 0.27, 0.9), false},
		{"red below range rejected", colorValue(-0.1, 0.27, 0.9), true},
		{"green above range rejected", colorValue(0.31, 1.1, 0.9), true},
		{"alpha below one rejected", colorWithAlpha(0.31, 0.27, 0.9, 0.5), true},
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
