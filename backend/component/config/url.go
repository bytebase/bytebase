package config

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// NormalizeExternalURL will format the external url.
func NormalizeExternalURL(externalURL string) (string, error) {
	r := strings.TrimSpace(externalURL)
	r = strings.TrimSuffix(r, "/")
	u, err := url.Parse(r)
	if err != nil {
		return "", errors.Wrapf(err, "%s malformed", externalURL)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errors.Errorf("%s must start with http:// or https://", externalURL)
	}
	if u.Host == "" {
		return "", errors.Errorf("%s must name a host", externalURL)
	}
	if u.User != nil {
		return "", errors.Errorf("%s must not carry userinfo", externalURL)
	}
	if u.RawQuery != "" || u.ForceQuery {
		return "", errors.Errorf("%s must not carry a query string", externalURL)
	}
	if u.Fragment != "" || u.RawFragment != "" {
		return "", errors.Errorf("%s must not carry a fragment", externalURL)
	}

	host := strings.ToLower(u.Host)
	port := u.Port()
	if port != "" {
		// The external URL is used as the redirectURL in the get token process of OAuth, and the
		// RedirectURL needs to be consistent with the RedirectURL in the get code process.
		// The frontend gets it through window.location.origin in the get code
		// process, so port 80/443 need to be cropped.
		if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
			host = strings.ToLower(u.Hostname())
			if strings.Contains(host, ":") {
				host = "[" + host + "]"
			}
		}
	}
	return scheme + "://" + host + strings.TrimSuffix(u.EscapedPath(), "/"), nil
}
