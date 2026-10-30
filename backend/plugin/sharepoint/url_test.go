package sharepoint

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr string
	}{
		{
			name: "site folder",
			raw:  "https://contoso.sharepoint.com/sites/finance/Exports/project-a",
			want: "https://contoso.sharepoint.com/sites/finance/Exports/project-a",
		},
		{
			name: "root site folder",
			raw:  "https://contoso.sharepoint.com/Exports/project-a",
			want: "https://contoso.sharepoint.com/Exports/project-a",
		},
		{
			name: "root site library",
			raw:  "https://contoso.sharepoint.com/Shared%20Documents",
			want: "https://contoso.sharepoint.com/Shared%20Documents",
		},
		{
			name: "teams site library",
			raw:  "https://contoso.sharepoint.com/teams/data/Exports",
			want: "https://contoso.sharepoint.com/teams/data/Exports",
		},
		{
			name: "multi-geo host and uppercase host",
			raw:  "https://Contoso-EU.SharePoint.com/sites/finance/Exports",
			want: "https://contoso-eu.sharepoint.com/sites/finance/Exports",
		},
		{
			name: "query, fragment, duplicate and trailing slashes",
			raw:  "  https://contoso.sharepoint.com//sites/finance//Exports/project-a/?web=1#top ",
			want: "https://contoso.sharepoint.com/sites/finance/Exports/project-a",
		},
		{
			name: "spaces and unicode",
			raw:  "https://contoso.sharepoint.com/sites/finance/Shared Documents/Q1 报表",
			want: "https://contoso.sharepoint.com/sites/finance/Shared%20Documents/Q1%20%E6%8A%A5%E8%A1%A8",
		},
		{
			name: "library view with id",
			raw:  "https://contoso.sharepoint.com/sites/finance/Exports/Forms/AllItems.aspx?id=%2Fsites%2Ffinance%2FExports%2Fproject%20a&viewid=6c2b3f0e-0000-0000-0000-000000000000",
			want: "https://contoso.sharepoint.com/sites/finance/Exports/project%20a",
		},
		{
			name: "root site library view with id",
			raw:  "https://contoso.sharepoint.com/Exports/Forms/AllItems.aspx?id=%2FExports%2Fproject-a",
			want: "https://contoso.sharepoint.com/Exports/project-a",
		},
		{
			name: "library view without id",
			raw:  "https://contoso.sharepoint.com/sites/finance/Shared%20Documents/Forms/AllItems.aspx",
			want: "https://contoso.sharepoint.com/sites/finance/Shared%20Documents",
		},
		{
			name: "copied folder link",
			raw:  "https://contoso.sharepoint.com/:f:/r/sites/finance/Exports/project-a?csf=1&web=1&e=AbCdEf",
			want: "https://contoso.sharepoint.com/sites/finance/Exports/project-a",
		},
		{
			name: "copied link with another type",
			raw:  "https://contoso.sharepoint.com/:u:/r/sites/finance/Exports",
			want: "https://contoso.sharepoint.com/sites/finance/Exports",
		},
		{
			name: "canonical URL is stable",
			raw:  "https://contoso.sharepoint.com/sites/finance/Shared%20Documents/Q1%20%E6%8A%A5%E8%A1%A8",
			want: "https://contoso.sharepoint.com/sites/finance/Shared%20Documents/Q1%20%E6%8A%A5%E8%A1%A8",
		},
		{name: "token sharing link", raw: "https://contoso.sharepoint.com/:x:/s/finance/EaBcDeF", wantErr: "sharing links are not supported"},
		{name: "guest sharing link", raw: "https://contoso.sharepoint.com/:x:/g/finance/EaBcDeF", wantErr: "browser's address bar"},
		{name: "http", raw: "http://contoso.sharepoint.com/sites/finance/Exports", wantErr: "https://"},
		{name: "not sharepoint", raw: "https://contoso.example.com/sites/finance/Exports", wantErr: "not a SharePoint Online host"},
		{name: "lookalike host", raw: "https://contoso.sharepoint.com.evil.com/sites/finance/Exports", wantErr: "not a SharePoint Online host"},
		{name: "bare sharepoint host", raw: "https://sharepoint.com/sites/finance/Exports", wantErr: "not a SharePoint Online host"},
		{name: "onedrive", raw: "https://contoso-my.sharepoint.com/personal/alice_contoso_com/Documents", wantErr: "OneDrive"},
		{name: "port", raw: "https://contoso.sharepoint.com:8443/sites/finance/Exports", wantErr: "port"},
		{name: "host only", raw: "https://contoso.sharepoint.com/", wantErr: "points to a site"},
		{name: "site only", raw: "https://contoso.sharepoint.com/sites/finance/", wantErr: "points to a site"},
		{name: "teams site only", raw: "https://contoso.sharepoint.com/teams/data", wantErr: "points to a site"},
		{name: "sites only", raw: "https://contoso.sharepoint.com/sites", wantErr: "points to a site"},
		{name: "site page", raw: "https://contoso.sharepoint.com/sites/finance/SitePages/Home.aspx", wantErr: "is a page"},
		{name: "layouts page", raw: "https://contoso.sharepoint.com/sites/finance/_layouts/15/viewlsts.aspx?view=14", wantErr: "is a page"},
		{name: "dot segment", raw: "https://contoso.sharepoint.com/sites/finance/Exports/..", wantErr: "path segments"},
		{name: "not a URL", raw: "sites/finance/Exports", wantErr: "full SharePoint URL"},
		{name: "empty", raw: "", wantErr: "full SharePoint URL"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeURL(tc.raw)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			again, err := NormalizeURL(got)
			require.NoError(t, err)
			require.Equal(t, got, again)
		})
	}
}
