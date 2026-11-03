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
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

const (
	graphBaseURL = "https://graph.microsoft.com/v1.0"
	graphScope   = "https://graph.microsoft.com/.default"
	// Graph requires upload chunks to be multiples of 320 KiB and smaller than
	// 60 MiB. Each chunk pays a fixed round trip, so larger chunks upload
	// faster; chunks above 20 MiB measured no faster and resend more after a
	// stalled request.
	uploadChunkSize = 64 * 320 * 1024
	// Files up to this size upload in one request; larger files use a
	// resumable upload session.
	simpleUploadMaxSize  = 4 << 20
	maxConcurrentUploads = 4
	maxAttempts          = 5
	requestTimeout       = 5 * time.Minute
	// recentResolveTTL bounds how long ResolveRecent reuses a resolution, so a
	// batch export resolves its destination once.
	recentResolveTTL = time.Minute
)

var aadstsPattern = regexp.MustCompile(`AADSTS\d+`)

type tokenSource func(ctx context.Context) (string, error)

type graphClient struct {
	baseURL              string
	clientID             string
	token                tokenSource
	httpClient           *http.Client
	chunkSize            int
	simpleUploadMaxSize  int
	maxConcurrentUploads int
	maxAttempts          int
	wait                 func(ctx context.Context, d time.Duration) error
	// now is time.Now when nil.
	now func() time.Time

	recentMu sync.Mutex
	// recent maps canonical URLs to resolutions made within recentResolveTTL.
	recent map[string]recentLocation
}

type recentLocation struct {
	location Location
	expires  time.Time
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
		baseURL:              graphBaseURL,
		clientID:             cfg.ClientID,
		token:                credentialTokenSource(cred),
		httpClient:           &http.Client{Timeout: requestTimeout},
		chunkSize:            uploadChunkSize,
		simpleUploadMaxSize:  simpleUploadMaxSize,
		maxConcurrentUploads: maxConcurrentUploads,
		maxAttempts:          maxAttempts,
		wait:                 sleepContext,
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
	WebURL string    `json:"webUrl"`
	Size   int64     `json:"size"`
	Folder *struct{} `json:"folder"`
	Root   *struct{} `json:"root"`

	ParentReference struct {
		DriveID string `json:"driveId"`
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
	return c.resolveCanonical(ctx, canonical)
}

func (c *graphClient) ResolveRecent(ctx context.Context, canonicalURL string) (*Location, error) {
	canonical, err := NormalizeURL(canonicalURL)
	if err != nil {
		return nil, err
	}
	c.recentMu.Lock()
	entry, ok := c.recent[canonical]
	c.recentMu.Unlock()
	if ok && c.clock().Before(entry.expires) {
		loc := entry.location
		loc.reused = true
		return &loc, nil
	}
	return c.resolveCanonical(ctx, canonical)
}

// resolveCanonical resolves a URL NormalizeURL returned and records the
// outcome for ResolveRecent.
func (c *graphClient) resolveCanonical(ctx context.Context, canonical string) (*Location, error) {
	loc, err := c.lookup(ctx, canonical)
	if err != nil {
		c.forget(canonical)
		return nil, err
	}
	c.recentMu.Lock()
	if c.recent == nil {
		c.recent = map[string]recentLocation{}
	}
	c.recent[canonical] = recentLocation{location: *loc, expires: c.clock().Add(recentResolveTTL)}
	c.recentMu.Unlock()
	return loc, nil
}

func (c *graphClient) forget(canonical string) {
	c.recentMu.Lock()
	defer c.recentMu.Unlock()
	delete(c.recent, canonical)
}

func (c *graphClient) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func (c *graphClient) lookup(ctx context.Context, canonical string) (*Location, error) {
	u, err := url.Parse(canonical)
	if err != nil {
		return nil, errors.Wrap(err, "invalid SharePoint URL")
	}
	// /shares accepts the URL with unescaped spaces.
	shareID := "u!" + base64.RawURLEncoding.EncodeToString([]byte("https://"+u.Host+u.Path))
	var item driveItem
	if err := c.graphJSON(ctx, http.MethodGet, "/shares/"+shareID+"/driveItem?$select=id,webUrl,folder,root,parentReference", nil, &item); err != nil {
		return nil, errors.Wrapf(err, "failed to resolve %s", canonical)
	}
	if item.Folder == nil && item.Root == nil {
		return nil, errors.Errorf("%s is a file; use a document library or folder", canonical)
	}
	if item.ID == "" || item.ParentReference.DriveID == "" {
		return nil, errors.Errorf("Microsoft Graph did not return the document library of %s", canonical)
	}
	return &Location{DriveID: item.ParentReference.DriveID, ItemID: item.ID, WebURL: item.WebURL, url: canonical}, nil
}

func (c *graphClient) Upload(ctx context.Context, loc *Location, folderName string, files []File) (*Folder, error) {
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

	folder, err := c.uploadFiles(ctx, loc, folderName, files)
	if err == nil || !errors.Is(err, ErrNotFoundOrNoAccess) {
		return folder, err
	}
	c.forget(loc.url)
	if !loc.reused || !errors.Is(err, ErrNothingUploaded) {
		return nil, err
	}
	// The destination may have been deleted or recreated at the same URL since
	// the reused resolution.
	fresh, err := c.resolveCanonical(ctx, loc.url)
	if err != nil {
		return nil, &nothingUploadedError{err: err}
	}
	return c.uploadFiles(ctx, fresh, folderName, files)
}

// uploadFiles addresses files by path, so the first file to arrive creates the
// export folder.
func (c *graphClient) uploadFiles(ctx context.Context, loc *Location, folderName string, files []File) (*Folder, error) {
	folderPath := itemPath(loc.DriveID, loc.ItemID) + ":/" + escapeSegment(folderName) + "/"
	var uploaded atomic.Int32
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(max(c.maxConcurrentUploads, 1))
	for _, file := range files {
		g.Go(func() error {
			if err := c.uploadFile(gctx, folderPath, file); err != nil {
				return err
			}
			uploaded.Add(1)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		// Graph creates nothing for a request it rejects, the folder included.
		if uploaded.Load() == 0 && (errors.Is(err, ErrNotFoundOrNoAccess) || errors.Is(err, ErrCredential)) {
			return nil, &nothingUploadedError{err: err}
		}
		return nil, err
	}
	return &Folder{Name: folderName, WebURL: strings.TrimRight(loc.WebURL, "/") + "/" + url.PathEscape(folderName)}, nil
}

func (c *graphClient) uploadFile(ctx context.Context, folderPath string, file File) error {
	err := c.putFile(ctx, folderPath, file)
	if err == nil {
		return nil
	}
	if statusOf(err) == http.StatusConflict {
		return c.reconcile(ctx, folderPath, file, false)
	}
	var unknown *unknownOutcomeError
	if errors.As(err, &unknown) {
		return c.reconcile(ctx, folderPath, file, true)
	}
	return errors.Wrapf(err, "failed to upload %q", file.Name)
}

func (c *graphClient) putFile(ctx context.Context, folderPath string, file File) error {
	target := folderPath + escapeSegment(file.Name) + ":"
	if len(file.Content) <= c.simpleUploadMaxSize {
		return c.graphRaw(ctx, http.MethodPut, target+"/content?@microsoft.graph.conflictBehavior=fail", file.Content)
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

// reconcile accepts an existing file as uploaded only if this application
// created it with the same size; export folder names are unique per export.
func (c *graphClient) reconcile(ctx context.Context, folderPath string, file File, outcomeUnknown bool) error {
	var item driveItem
	err := c.graphJSON(ctx, http.MethodGet, folderPath+escapeSegment(file.Name)+"?$select=size,folder,createdBy", nil, &item)
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
