package sharepoint

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var (
	// copiedLinkPattern matches links copied from SharePoint, such as
	// /:f:/r/sites/finance/Exports (a path) or /:f:/s/finance/EaBc (a token).
	copiedLinkPattern  = regexp.MustCompile(`^/:[A-Za-z]:/([^/]*)(/.*)?$`)
	libraryViewPattern = regexp.MustCompile(`(?i)/Forms/[^/]+\.aspx$`)
)

// NormalizeURL returns the canonical URL of a SharePoint library root or folder
// from a URL pasted from a browser.
func NormalizeURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "", errors.New("enter the full SharePoint URL of a document library or folder, starting with https://")
	}
	if u.Scheme != "https" {
		return "", errors.New("the SharePoint URL must start with https://")
	}
	if u.User != nil || u.Port() != "" {
		return "", errors.New("the SharePoint URL must not contain a user name or port")
	}
	host := strings.ToLower(u.Hostname())
	tenant, ok := strings.CutSuffix(host, ".sharepoint.com")
	if !ok || tenant == "" || strings.Contains(tenant, ".") {
		return "", errors.Errorf("%q is not a SharePoint Online host; the URL must be on a *.sharepoint.com host", host)
	}
	if strings.HasSuffix(tenant, "-my") {
		return "", errors.New("OneDrive URLs are not supported; use a document library or folder on a SharePoint site")
	}

	p := u.Path
	if m := copiedLinkPattern.FindStringSubmatch(p); m != nil {
		if !strings.EqualFold(m[1], "r") {
			return "", errors.New("sharing links are not supported; open the folder in SharePoint and paste the URL from the browser's address bar")
		}
		p = m[2]
	} else if strings.HasSuffix(strings.ToLower(p), ".aspx") {
		query := u.Query()
		if id := query.Get("id"); id != "" {
			p = id
		} else if rootFolder := query.Get("RootFolder"); rootFolder != "" {
			p = rootFolder
		} else {
			p = libraryViewPattern.ReplaceAllString(p, "")
		}
	}

	var segments []string
	for _, segment := range strings.Split(p, "/") {
		switch {
		case segment == "":
		case segment == "." || segment == "..":
			return "", errors.New("the SharePoint URL must not contain . or .. path segments")
		case strings.EqualFold(segment, "_layouts") || strings.HasSuffix(strings.ToLower(segment), ".aspx"):
			return "", errors.New("the SharePoint URL is a page, not a document library or folder; open the folder in SharePoint and paste the URL from the browser's address bar")
		default:
			segments = append(segments, segment)
		}
	}
	if len(segments) == 0 {
		return "", errors.New("the SharePoint URL points to a site, not a document library or folder")
	}
	if first := strings.ToLower(segments[0]); (first == "sites" || first == "teams") && len(segments) <= 2 {
		return "", errors.New("the SharePoint URL points to a site, not a document library or folder")
	}
	canonical := url.URL{Scheme: "https", Host: host, Path: "/" + strings.Join(segments, "/")}
	return canonical.String(), nil
}
