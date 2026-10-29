package sharepoint

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/pkg/errors"
)

const (
	graphBaseURL = "https://graph.microsoft.com/v1.0"
	graphScope   = "https://graph.microsoft.com/.default"
	// Graph requires upload chunks to be multiples of 320 KiB.
	uploadChunkSize = 16 * 320 * 1024
	maxAttempts     = 5
	requestTimeout  = 5 * time.Minute
)

var aadstsPattern = regexp.MustCompile(`AADSTS\d+`)

type tokenSource func(ctx context.Context) (string, error)

type graphClient struct {
	baseURL     string
	clientID    string
	token       tokenSource
	httpClient  *http.Client
	chunkSize   int
	maxAttempts int
	wait        func(ctx context.Context, d time.Duration) error
}

func newGraphClient(cfg Config) (Client, error) {
	if cfg.TenantID == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, errors.New("SharePoint requires a tenant ID, client ID, and client secret")
	}
	cred, err := azidentity.NewClientSecretCredential(cfg.TenantID, cfg.ClientID, cfg.ClientSecret, nil)
	if err != nil {
		return nil, errors.Wrapf(ErrCredential, "invalid application configuration: %v", err)
	}
	return &graphClient{
		baseURL:     graphBaseURL,
		clientID:    cfg.ClientID,
		token:       credentialTokenSource(cred),
		httpClient:  &http.Client{Timeout: requestTimeout},
		chunkSize:   uploadChunkSize,
		maxAttempts: maxAttempts,
		wait:        sleepContext,
	}, nil
}

func credentialTokenSource(cred azcore.TokenCredential) tokenSource {
	return func(ctx context.Context) (string, error) {
		token, err := cred.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{graphScope}})
		if err == nil {
			return token.Token, nil
		}
		var authErr *azidentity.AuthenticationFailedError
		if errors.As(err, &authErr) && (authErr.RawResponse == nil || authErr.RawResponse.StatusCode < http.StatusInternalServerError) {
			// Report only the Entra error code; the full error is verbose.
			if code := aadstsPattern.FindString(authErr.Error()); code != "" {
				return "", errors.Wrap(ErrCredential, code)
			}
			return "", errors.WithStack(ErrCredential)
		}
		return "", errors.Wrap(err, "failed to get a Microsoft Entra token")
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type driveItem struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	WebURL string    `json:"webUrl"`
	Size   int64     `json:"size"`
	Folder *struct{} `json:"folder"`
	Root   *struct{} `json:"root"`

	ParentReference struct {
		DriveID string `json:"driveId"`
		SiteID  string `json:"siteId"`
		Path    string `json:"path"`
	} `json:"parentReference"`
	CreatedBy struct {
		Application *struct {
			ID string `json:"id"`
		} `json:"application"`
	} `json:"createdBy"`
}

func (c *graphClient) Resolve(ctx context.Context, canonicalURL string) (*Location, error) {
	canonical, err := NormalizeURL(canonicalURL)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(canonical)
	if err != nil {
		return nil, errors.Wrap(err, "invalid SharePoint URL")
	}
	// /shares accepts the URL with unescaped spaces.
	shareID := "u!" + base64.RawURLEncoding.EncodeToString([]byte("https://"+u.Host+u.Path))
	var item driveItem
	if err := c.graphJSON(ctx, http.MethodGet, "/shares/"+shareID+"/driveItem?$select=id,name,webUrl,folder,root,parentReference", nil, &item); err != nil {
		return nil, errors.Wrapf(err, "failed to resolve %s", canonical)
	}
	if item.Folder == nil && item.Root == nil {
		return nil, errors.Errorf("%s is a file; use a document library or folder", canonical)
	}
	driveID := item.ParentReference.DriveID
	if item.ID == "" || driveID == "" {
		return nil, errors.Errorf("Microsoft Graph did not return the document library of %s", canonical)
	}

	var drive struct {
		Name          string `json:"name"`
		SharePointIDs struct {
			SiteID string `json:"siteId"`
			WebID  string `json:"webId"`
		} `json:"sharePointIds"`
	}
	if err := c.graphJSON(ctx, http.MethodGet, "/drives/"+escapeSegment(driveID)+"?$select=name,sharePointIds", nil, &drive); err != nil {
		return nil, errors.Wrapf(err, "failed to get the document library of %s", canonical)
	}
	siteID := item.ParentReference.SiteID
	if siteID == "" && drive.SharePointIDs.SiteID != "" && drive.SharePointIDs.WebID != "" {
		siteID = u.Host + "," + drive.SharePointIDs.SiteID + "," + drive.SharePointIDs.WebID
	}
	if siteID == "" {
		return nil, errors.Errorf("Microsoft Graph did not return the site of %s", canonical)
	}
	var site struct {
		DisplayName string `json:"displayName"`
	}
	if err := c.graphJSON(ctx, http.MethodGet, "/sites/"+escapeSiteID(siteID)+"?$select=displayName", nil, &site); err != nil {
		return nil, errors.Wrapf(err, "failed to get the site of %s", canonical)
	}

	location := &Location{
		DriveID:     driveID,
		ItemID:      item.ID,
		SiteName:    site.DisplayName,
		LibraryName: drive.Name,
		WebURL:      item.WebURL,
	}
	if item.Root == nil {
		location.FolderPath = folderPath(item.ParentReference.Path, item.Name)
	}
	return location, nil
}

// folderPath returns the path of an item within its library from its parent
// path, such as /drives/{id}/root:/a/b.
func folderPath(parentPath, name string) string {
	_, parent, _ := strings.Cut(parentPath, "root:")
	if unescaped, err := url.PathUnescape(parent); err == nil {
		parent = unescaped
	}
	return strings.Trim(parent+"/"+name, "/")
}

func (c *graphClient) Deliver(ctx context.Context, loc *Location, folderName string, files []File) (*Delivery, error) {
	if loc == nil || loc.DriveID == "" || loc.ItemID == "" {
		return nil, errors.New("the SharePoint destination is not resolved")
	}
	if folderName == "" {
		return nil, errors.New("the SharePoint folder name is empty")
	}
	seen := make(map[string]bool, len(files))
	for _, file := range files {
		key := strings.ToLower(file.Name)
		if file.Name == "" || seen[key] {
			return nil, errors.Errorf("invalid or duplicate export file name %q", file.Name)
		}
		seen[key] = true
	}

	folder, err := c.ensureFolder(ctx, loc, folderName)
	if err != nil {
		return nil, &nothingDeliveredError{err: err}
	}
	for _, file := range files {
		if err := c.uploadFile(ctx, loc.DriveID, folder.ID, file); err != nil {
			return nil, err
		}
	}
	return &Delivery{FolderName: folderName, WebURL: folder.WebURL}, nil
}

func (c *graphClient) ensureFolder(ctx context.Context, loc *Location, folderName string) (*driveItem, error) {
	parent := itemPath(loc.DriveID, loc.ItemID)
	body := map[string]any{
		"name":                              folderName,
		"folder":                            map[string]any{},
		"@microsoft.graph.conflictBehavior": "fail",
	}
	var folder driveItem
	err := c.graphJSON(ctx, http.MethodPost, parent+"/children", body, &folder)
	if statusOf(err) == http.StatusConflict {
		folder = driveItem{}
		err = c.graphJSON(ctx, http.MethodGet, parent+":/"+escapeSegment(folderName)+"?$select=id,webUrl,folder", nil, &folder)
		if err == nil && folder.Folder == nil {
			return nil, errors.Errorf("a file named %q already exists in the SharePoint destination", folderName)
		}
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create SharePoint folder %q", folderName)
	}
	if folder.ID == "" {
		return nil, errors.Errorf("Microsoft Graph did not return SharePoint folder %q", folderName)
	}
	return &folder, nil
}

func (c *graphClient) uploadFile(ctx context.Context, driveID, folderID string, file File) error {
	err := c.upload(ctx, driveID, folderID, file)
	if err == nil {
		return nil
	}
	if statusOf(err) == http.StatusConflict {
		return c.reconcile(ctx, driveID, folderID, file, false)
	}
	var unknown *unknownOutcomeError
	if errors.As(err, &unknown) {
		return c.reconcile(ctx, driveID, folderID, file, true)
	}
	return errors.Wrapf(err, "failed to upload %q", file.Name)
}

func (c *graphClient) upload(ctx context.Context, driveID, folderID string, file File) error {
	target := itemPath(driveID, folderID) + ":/" + escapeSegment(file.Name) + ":"
	if len(file.Content) == 0 {
		// Upload sessions cannot upload empty files.
		return c.graphJSON(ctx, http.MethodPut, target+"/content?@microsoft.graph.conflictBehavior=fail", nil, nil)
	}
	body := map[string]any{"item": map[string]any{"@microsoft.graph.conflictBehavior": "fail"}}
	var session struct {
		UploadURL string `json:"uploadUrl"`
	}
	if err := c.graphJSON(ctx, http.MethodPost, target+"/createUploadSession", body, &session); err != nil {
		return err
	}
	if session.UploadURL == "" {
		return errors.New("Microsoft Graph did not return an upload session")
	}
	return c.uploadChunks(ctx, session.UploadURL, file.Content)
}

// uploadChunks uploads content to an upload session. The upload URL is
// pre-authenticated, so it is sent without a token and kept out of errors.
func (c *graphClient) uploadChunks(ctx context.Context, uploadURL string, content []byte) error {
	total := len(content)
	// Bound the loop in case the session keeps asking for the same range.
	maxRequests := 2*(total/c.chunkSize+1) + c.maxAttempts
	offset := 0
	for range maxRequests {
		end := min(offset+c.chunkSize, total)
		header := http.Header{}
		header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, end-1, total))
		resp, err := c.send(ctx, http.MethodPut, uploadURL, content[offset:end], header, false)
		if err != nil {
			return err
		}
		switch resp.status {
		case http.StatusOK, http.StatusCreated:
			return nil
		case http.StatusAccepted:
			offset, err = nextExpectedOffset(resp.body, end)
		case http.StatusRequestedRangeNotSatisfiable:
			// The previous attempt of this chunk may have been received.
			offset, err = c.uploadSessionOffset(ctx, uploadURL)
		case http.StatusNotFound:
			// The session is gone, possibly because the last chunk completed it.
			return &unknownOutcomeError{cause: errors.New("the upload session no longer exists")}
		default:
			return newGraphError(resp)
		}
		if err != nil {
			return err
		}
		if offset < 0 || offset >= total {
			return &unknownOutcomeError{cause: errors.Errorf("the upload session expects an invalid offset %d", offset)}
		}
	}
	return &unknownOutcomeError{cause: errors.New("the upload session did not complete")}
}

func (c *graphClient) uploadSessionOffset(ctx context.Context, uploadURL string) (int, error) {
	resp, err := c.send(ctx, http.MethodGet, uploadURL, nil, nil, false)
	if err != nil {
		return 0, err
	}
	if resp.status != http.StatusOK {
		return 0, &unknownOutcomeError{cause: newGraphError(resp)}
	}
	offset, err := nextExpectedOffset(resp.body, -1)
	if err != nil || offset < 0 {
		return 0, &unknownOutcomeError{cause: errors.New("the upload session status is invalid")}
	}
	return offset, nil
}

// nextExpectedOffset returns the start of the first range in an upload
// session's nextExpectedRanges, or fallback when there is none.
func nextExpectedOffset(body []byte, fallback int) (int, error) {
	var status struct {
		NextExpectedRanges []string `json:"nextExpectedRanges"`
	}
	if err := json.Unmarshal(body, &status); err != nil {
		return 0, errors.Wrap(err, "failed to parse the upload session status")
	}
	if len(status.NextExpectedRanges) == 0 {
		return fallback, nil
	}
	start, _, _ := strings.Cut(status.NextExpectedRanges[0], "-")
	var offset int
	if _, err := fmt.Sscanf(start, "%d", &offset); err != nil {
		return 0, errors.Errorf("invalid upload range %q", status.NextExpectedRanges[0])
	}
	return offset, nil
}

// reconcile accepts an existing file as delivered only if this application
// created it with the same size; export folder names are unique per export.
func (c *graphClient) reconcile(ctx context.Context, driveID, folderID string, file File, outcomeUnknown bool) error {
	var item driveItem
	err := c.graphJSON(ctx, http.MethodGet, itemPath(driveID, folderID)+":/"+escapeSegment(file.Name)+"?$select=size,folder,createdBy", nil, &item)
	if err == nil && item.Folder == nil && item.Size == int64(len(file.Content)) &&
		item.CreatedBy.Application != nil && strings.EqualFold(item.CreatedBy.Application.ID, c.clientID) {
		return nil
	}
	var unknown *unknownOutcomeError
	if outcomeUnknown || errors.As(err, &unknown) {
		return errors.Wrapf(ErrUnconfirmed, "file %q", file.Name)
	}
	if err != nil {
		return errors.Wrapf(err, "failed to check the existing file %q", file.Name)
	}
	return errors.Errorf("a different file named %q already exists in the SharePoint folder", file.Name)
}

func itemPath(driveID, itemID string) string {
	return "/drives/" + escapeSegment(driveID) + "/items/" + escapeSegment(itemID)
}

// escapeSegment escapes one path segment. Graph treats ":" as the start of
// path-based addressing, so it is escaped too.
func escapeSegment(s string) string {
	return strings.ReplaceAll(url.PathEscape(s), ":", "%3A")
}

// escapeSiteID escapes a site ID such as "host,siteGUID,webGUID", keeping the
// commas Graph uses to separate its parts.
func escapeSiteID(s string) string {
	parts := strings.Split(s, ",")
	for i, part := range parts {
		parts[i] = escapeSegment(part)
	}
	return strings.Join(parts, ",")
}
