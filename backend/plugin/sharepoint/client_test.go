package sharepoint

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

const testAppID = "11111111-2222-3333-4444-555555555555"

func newTestClient(t *testing.T, handler http.Handler) (*graphClient, *[]time.Duration) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	var (
		mu    sync.Mutex
		waits []time.Duration
	)
	client := &graphClient{
		baseURL:    srv.URL,
		clientID:   testAppID,
		token:      func(context.Context) (string, error) { return "test-token", nil },
		httpClient: srv.Client(),
		chunkSize:  4,
		// Files larger than 4 bytes use upload sessions, and uploads run one
		// at a time so tests can assert their order.
		simpleUploadMaxSize:  4,
		maxConcurrentUploads: 1,
		maxAttempts:          3,
		reconcileGrace:       5 * time.Second,
		wait: func(_ context.Context, d time.Duration) error {
			mu.Lock()
			defer mu.Unlock()
			waits = append(waits, d)
			return nil
		},
	}
	return client, &waits
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeGraphError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": code}})
}

// resolveShareURL is the decoded form of the URL every resolve test asks for.
const resolveShareURL = "https://contoso.sharepoint.com/sites/finance/Shared Documents/My Exports/project a"

func resolveServer(t *testing.T, itemStatus int, item map[string]any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		shareID := "u!" + base64.RawURLEncoding.EncodeToString([]byte(resolveShareURL))
		switch r.URL.Path {
		case "/shares/" + shareID + "/driveItem":
			require.Equal(t, "id,webUrl,folder,root,parentReference", r.URL.Query().Get("$select"))
			if itemStatus != http.StatusOK {
				writeGraphError(w, itemStatus, "accessDenied")
				return
			}
			writeJSON(w, http.StatusOK, item)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			writeGraphError(w, http.StatusNotFound, "itemNotFound")
		}
	})
}

func TestResolve(t *testing.T) {
	const canonical = "https://contoso.sharepoint.com/sites/finance/Shared%20Documents/My%20Exports/project%20a"
	parent := map[string]any{"driveId": "d1"}

	t.Run("library root", func(t *testing.T) {
		client, _ := newTestClient(t, resolveServer(t, http.StatusOK, map[string]any{
			"id": "root1", "name": "root", "webUrl": "https://contoso.sharepoint.com/sites/finance/Shared%20Documents",
			"folder": map[string]any{"childCount": 2}, "root": map[string]any{}, "parentReference": parent,
		}))
		loc, err := client.Resolve(context.Background(), canonical)
		require.NoError(t, err)
		require.Equal(t, &Location{DriveID: "d1", ItemID: "root1", WebURL: "https://contoso.sharepoint.com/sites/finance/Shared%20Documents", url: canonical}, loc)
	})

	t.Run("folder", func(t *testing.T) {
		client, _ := newTestClient(t, resolveServer(t, http.StatusOK, map[string]any{
			"id": "item1", "name": "project a", "webUrl": canonical, "folder": map[string]any{"childCount": 0}, "parentReference": parent,
		}))
		loc, err := client.Resolve(context.Background(), canonical)
		require.NoError(t, err)
		require.Equal(t, &Location{DriveID: "d1", ItemID: "item1", WebURL: canonical, url: canonical}, loc)
	})

	t.Run("file", func(t *testing.T) {
		client, _ := newTestClient(t, resolveServer(t, http.StatusOK, map[string]any{
			"id": "item1", "name": "project a", "file": map[string]any{}, "parentReference": parent,
		}))
		_, err := client.Resolve(context.Background(), canonical)
		require.ErrorIs(t, err, ErrNotFolder)
	})

	t.Run("forbidden", func(t *testing.T) {
		client, _ := newTestClient(t, resolveServer(t, http.StatusForbidden, nil))
		_, err := client.Resolve(context.Background(), canonical)
		require.ErrorIs(t, err, ErrNotFoundOrNoAccess)
	})

	t.Run("unauthorized", func(t *testing.T) {
		client, _ := newTestClient(t, resolveServer(t, http.StatusUnauthorized, nil))
		_, err := client.Resolve(context.Background(), canonical)
		require.ErrorIs(t, err, ErrCredential)
	})
}

type fakeFile struct {
	content []byte
	appID   string
}

type fakeSession struct {
	key  string
	data []byte
}

// fakeGraph serves a library root "dest" in drive "d1".
type fakeGraph struct {
	t  *testing.T
	mu sync.Mutex
	// folders holds the names of the folders under the destination.
	folders map[string]bool
	// files is keyed by "folderName/fileName".
	files         map[string]fakeFile
	sessions      map[string]*fakeSession
	sessionCount  int
	requestCount  int
	contentRanges []string
	// override handles a request instead of the fake when it returns true.
	override func(w http.ResponseWriter, r *http.Request) bool
	// dropUploadResponse stores the next chunk and then drops the connection.
	dropUploadResponse bool
}

func newFakeGraph(t *testing.T) *fakeGraph {
	return &fakeGraph{t: t, folders: map[string]bool{}, files: map[string]fakeFile{}, sessions: map[string]*fakeSession{}}
}

func (f *fakeGraph) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requestCount++
	if f.override != nil && f.override(w, r) {
		return
	}
	if strings.HasPrefix(r.URL.Path, "/upload/") {
		f.serveUpload(w, r)
		return
	}
	require.Equal(f.t, "Bearer test-token", r.Header.Get("Authorization"))
	// Files are addressed by path under the destination item "dest".
	rest, ok := strings.CutPrefix(r.URL.Path, "/drives/d1/items/dest:/")
	if !ok {
		writeGraphError(w, http.StatusNotFound, "itemNotFound")
		return
	}
	folderName, sub, ok := strings.Cut(rest, "/")
	if !ok {
		f.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		writeGraphError(w, http.StatusBadRequest, "invalidRequest")
		return
	}
	switch {
	case r.Method == http.MethodPost && strings.HasSuffix(sub, ":/createUploadSession"):
		key := folderName + "/" + strings.TrimSuffix(sub, ":/createUploadSession")
		if _, exists := f.files[key]; exists {
			writeGraphError(w, http.StatusConflict, "nameAlreadyExists")
			return
		}
		f.folders[folderName] = true
		f.sessionCount++
		id := fmt.Sprint(f.sessionCount)
		f.sessions[id] = &fakeSession{key: key}
		writeJSON(w, http.StatusOK, map[string]any{"uploadUrl": "http://" + r.Host + "/upload/" + id + "?tempauth=secret"})
	case r.Method == http.MethodPut && strings.HasSuffix(sub, ":/content"):
		key := folderName + "/" + strings.TrimSuffix(sub, ":/content")
		if _, exists := f.files[key]; exists {
			writeGraphError(w, http.StatusConflict, "nameAlreadyExists")
			return
		}
		data, _ := io.ReadAll(r.Body)
		f.folders[folderName] = true
		f.files[key] = fakeFile{content: data, appID: testAppID}
		writeJSON(w, http.StatusCreated, map[string]any{"id": key})
	case r.Method == http.MethodGet:
		file, exists := f.files[folderName+"/"+sub]
		if !exists {
			writeGraphError(w, http.StatusNotFound, "itemNotFound")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"size":      len(file.content),
			"createdBy": map[string]any{"application": map[string]any{"id": file.appID}},
		})
	default:
		f.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		writeGraphError(w, http.StatusBadRequest, "invalidRequest")
	}
}

func (f *fakeGraph) serveUpload(w http.ResponseWriter, r *http.Request) {
	require.Empty(f.t, r.Header.Get("Authorization"))
	session, ok := f.sessions[strings.TrimPrefix(r.URL.Path, "/upload/")]
	if !ok {
		writeGraphError(w, http.StatusNotFound, "itemNotFound")
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"nextExpectedRanges": []string{fmt.Sprintf("%d-", len(session.data))}})
		return
	}
	contentRange := r.Header.Get("Content-Range")
	var start, end, total int
	_, err := fmt.Sscanf(contentRange, "bytes %d-%d/%d", &start, &end, &total)
	require.NoError(f.t, err)
	data, err := io.ReadAll(r.Body)
	require.NoError(f.t, err)
	require.Len(f.t, data, end-start+1)
	if start != len(session.data) {
		writeGraphError(w, http.StatusRequestedRangeNotSatisfiable, "invalidRange")
		return
	}
	f.contentRanges = append(f.contentRanges, contentRange)
	session.data = append(session.data, data...)
	if f.dropUploadResponse {
		f.dropUploadResponse = false
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(f.t, err)
		_ = conn.Close()
		return
	}
	if len(session.data) < total {
		writeJSON(w, http.StatusAccepted, map[string]any{"nextExpectedRanges": []string{fmt.Sprintf("%d-", len(session.data))}})
		return
	}
	f.files[session.key] = fakeFile{content: session.data, appID: testAppID}
	writeJSON(w, http.StatusCreated, map[string]any{"id": session.key})
}

const testFolder = "20261022T010203Z alice@example.com db 42"

var (
	testLocation = &Location{DriveID: "d1", ItemID: "dest", WebURL: "https://sp/dest"}
	testFiles    = []File{
		{Name: "statement.sql", Content: []byte("SELECT 1;\n")},
		{Name: "result #1.csv", Content: []byte("a,b\n1,2\n")},
		{Name: "empty.json", Content: []byte{}},
	}
)

func requireUploaded(t *testing.T, fake *fakeGraph) {
	t.Helper()
	for _, file := range testFiles {
		got, ok := fake.files[testFolder+"/"+file.Name]
		require.True(t, ok, file.Name)
		require.Equal(t, string(file.Content), string(got.content))
	}
}

const testFolderURL = "https://sp/dest/20261022T010203Z%20alice@example.com%20db%2042"

func TestUpload(t *testing.T) {
	fake := newFakeGraph(t)
	client, waits := newTestClient(t, fake)
	folder, err := client.Upload(context.Background(), testLocation, testFolder, testFiles)
	require.NoError(t, err)
	require.Equal(t, &Folder{Name: testFolder, WebURL: testFolderURL}, folder)
	requireUploaded(t, fake)
	require.Equal(t, []string{
		"bytes 0-3/10", "bytes 4-7/10", "bytes 8-9/10",
		"bytes 0-3/8", "bytes 4-7/8",
	}, fake.contentRanges)
	require.Empty(t, *waits)

	// Retrying the same export reuses the folder and the uploaded files.
	folder, err = client.Upload(context.Background(), testLocation, testFolder, testFiles)
	require.NoError(t, err)
	require.Equal(t, &Folder{Name: testFolder, WebURL: testFolderURL}, folder)
	require.Len(t, fake.folders, 1)
	require.Equal(t, 2, fake.sessionCount)
}

func TestUploadFileConflict(t *testing.T) {
	files := []File{{Name: "result.csv", Content: []byte("a,b\n")}}
	tests := []struct {
		name     string
		existing fakeFile
		wantErr  string
	}{
		{name: "uploaded by this application", existing: fakeFile{content: []byte("x,y\n"), appID: testAppID}},
		{name: "different application", existing: fakeFile{content: []byte("a,b\n"), appID: "someone-else"}, wantErr: "a different file named"},
		{name: "different size", existing: fakeFile{content: []byte("a,b,c\n"), appID: testAppID}, wantErr: "a different file named"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeGraph(t)
			fake.folders[testFolder] = true
			fake.files[testFolder+"/result.csv"] = tc.existing
			client, _ := newTestClient(t, fake)
			_, err := client.Upload(context.Background(), testLocation, testFolder, files)
			require.Zero(t, fake.sessionCount)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
			require.NotErrorIs(t, err, ErrUnconfirmed)
			require.NotErrorIs(t, err, ErrNothingUploaded)
		})
	}
}

func TestUploadHonorsRetryAfter(t *testing.T) {
	fake := newFakeGraph(t)
	throttled := false
	fake.override = func(w http.ResponseWriter, _ *http.Request) bool {
		if throttled {
			return false
		}
		throttled = true
		w.Header().Set("Retry-After", "2")
		writeGraphError(w, http.StatusTooManyRequests, "activityLimitReached")
		return true
	}
	client, waits := newTestClient(t, fake)
	_, err := client.Upload(context.Background(), testLocation, testFolder, testFiles)
	require.NoError(t, err)
	require.Equal(t, []time.Duration{2 * time.Second}, *waits)
	requireUploaded(t, fake)
}

func TestUploadResumesAfterLostChunkResponse(t *testing.T) {
	fake := newFakeGraph(t)
	fake.override = func(_ http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPut && r.Header.Get("Content-Range") == "bytes 4-7/10" && len(fake.contentRanges) == 1 {
			fake.dropUploadResponse = true
		}
		return false
	}
	client, waits := newTestClient(t, fake)
	_, err := client.Upload(context.Background(), testLocation, testFolder, testFiles)
	require.NoError(t, err)
	requireUploaded(t, fake)
	require.Equal(t, []time.Duration{time.Second}, *waits)
}

func TestUploadUnconfirmed(t *testing.T) {
	fake := newFakeGraph(t)
	fake.override = func(w http.ResponseWriter, r *http.Request) bool {
		if !strings.HasPrefix(r.URL.Path, "/upload/") {
			return false
		}
		writeGraphError(w, http.StatusGatewayTimeout, "gatewayTimeout")
		return true
	}
	client, waits := newTestClient(t, fake)
	_, err := client.Upload(context.Background(), testLocation, testFolder, testFiles)
	require.ErrorIs(t, err, ErrUnconfirmed)
	require.ErrorContains(t, err, "statement.sql")
	require.NotContains(t, err.Error(), "tempauth")
	require.Equal(t, []time.Duration{time.Second, 2 * time.Second}, *waits)
}

func TestUploadForbiddenUploadsNothing(t *testing.T) {
	fake := newFakeGraph(t)
	fake.override = func(w http.ResponseWriter, _ *http.Request) bool {
		writeGraphError(w, http.StatusForbidden, "accessDenied")
		return true
	}
	client, _ := newTestClient(t, fake)
	client.maxConcurrentUploads = len(testFiles)
	_, err := client.Upload(context.Background(), testLocation, testFolder, testFiles)
	require.ErrorIs(t, err, ErrNotFoundOrNoAccess)
	require.ErrorIs(t, err, ErrNothingUploaded)
	require.Empty(t, fake.folders)
}

func TestUploadStorageFull(t *testing.T) {
	fake := newFakeGraph(t)
	fake.override = func(w http.ResponseWriter, _ *http.Request) bool {
		writeGraphError(w, http.StatusInsufficientStorage, "quotaLimitReached")
		return true
	}
	client, waits := newTestClient(t, fake)
	_, err := client.Upload(context.Background(), testLocation, testFolder, testFiles)
	require.ErrorIs(t, err, ErrStorageFull)
	require.ErrorIs(t, err, ErrNothingUploaded)
	require.ErrorContains(t, err, "quotaLimitReached")
	require.Empty(t, *waits)
}

func TestUploadRejectionDoesNotDenyAConcurrentUpload(t *testing.T) {
	files := []File{{Name: "a.sql", Content: []byte("a")}, {Name: "b.csv", Content: []byte("b")}}
	rejected := make(chan struct{})
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "b.csv") {
			writeGraphError(w, http.StatusForbidden, "accessDenied")
			close(rejected)
			return
		}
		<-rejected
		writeJSON(w, http.StatusCreated, map[string]any{"id": "a"})
	}))
	client.simpleUploadMaxSize = simpleUploadMaxSize
	client.maxConcurrentUploads = len(files)
	_, err := client.Upload(context.Background(), testLocation, testFolder, files)
	require.ErrorIs(t, err, ErrNotFoundOrNoAccess)
	require.NotErrorIs(t, err, ErrNothingUploaded)
}

// recreatableDestination serves a destination URL that resolves to the item
// named by current, under drive d1. Only the item "dest" holds files.
type recreatableDestination struct {
	fake    *fakeGraph
	current string
	resolve int
}

func newRecreatableDestination(t *testing.T, current string) *recreatableDestination {
	d := &recreatableDestination{fake: newFakeGraph(t), current: current}
	d.fake.override = func(w http.ResponseWriter, r *http.Request) bool {
		if !strings.HasPrefix(r.URL.Path, "/shares/") {
			return false
		}
		d.resolve++
		if d.current == "" {
			writeGraphError(w, http.StatusForbidden, "accessDenied")
			return true
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id": d.current, "webUrl": "https://sp/" + d.current, "folder": map[string]any{},
			"parentReference": map[string]any{"driveId": "d1"},
		})
		return true
	}
	return d
}

func TestResolveRecent(t *testing.T) {
	dest := newRecreatableDestination(t, "dest")
	client, _ := newTestClient(t, dest.fake)
	now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	client.now = func() time.Time { return now }
	ctx := context.Background()

	loc, err := client.ResolveRecent(ctx, resolveShareURL)
	require.NoError(t, err)
	require.Equal(t, "dest", loc.ItemID)
	_, err = client.ResolveRecent(ctx, resolveShareURL)
	require.NoError(t, err)
	require.Equal(t, 1, dest.resolve)

	// Resolve always asks Graph.
	_, err = client.Resolve(ctx, resolveShareURL)
	require.NoError(t, err)
	require.Equal(t, 2, dest.resolve)

	now = now.Add(recentResolveTTL)
	_, err = client.ResolveRecent(ctx, resolveShareURL)
	require.NoError(t, err)
	require.Equal(t, 3, dest.resolve)
}

func TestUploadResolvesReusedDestinationAgain(t *testing.T) {
	ctx := context.Background()
	t.Run("recreated destination receives the export", func(t *testing.T) {
		dest := newRecreatableDestination(t, "deleted")
		client, _ := newTestClient(t, dest.fake)
		_, err := client.ResolveRecent(ctx, resolveShareURL)
		require.NoError(t, err)
		dest.current = "dest"
		loc, err := client.ResolveRecent(ctx, resolveShareURL)
		require.NoError(t, err)
		require.Equal(t, "deleted", loc.ItemID)

		folder, err := client.Upload(ctx, loc, testFolder, testFiles)
		require.NoError(t, err)
		require.Equal(t, testFolderURL, folder.WebURL)
		requireUploaded(t, dest.fake)
		require.Equal(t, 2, dest.resolve)
	})
	t.Run("deleted destination fails without uploading", func(t *testing.T) {
		dest := newRecreatableDestination(t, "deleted")
		client, _ := newTestClient(t, dest.fake)
		_, err := client.ResolveRecent(ctx, resolveShareURL)
		require.NoError(t, err)
		dest.current = ""
		loc, err := client.ResolveRecent(ctx, resolveShareURL)
		require.NoError(t, err)

		_, err = client.Upload(ctx, loc, testFolder, testFiles)
		require.ErrorIs(t, err, ErrNotFoundOrNoAccess)
		require.ErrorIs(t, err, ErrNothingUploaded)
		require.Equal(t, 2, dest.resolve)
	})
	t.Run("fresh resolution is not resolved again", func(t *testing.T) {
		dest := newRecreatableDestination(t, "deleted")
		client, _ := newTestClient(t, dest.fake)
		loc, err := client.ResolveRecent(ctx, resolveShareURL)
		require.NoError(t, err)

		_, err = client.Upload(ctx, loc, testFolder, testFiles)
		require.ErrorIs(t, err, ErrNotFoundOrNoAccess)
		require.Equal(t, 1, dest.resolve)
		// The failure drops the resolution, so the next export asks Graph.
		_, err = client.ResolveRecent(ctx, resolveShareURL)
		require.NoError(t, err)
		require.Equal(t, 2, dest.resolve)
	})
}

func TestTransportErrorOmitsURL(t *testing.T) {
	client, _ := newTestClient(t, http.NotFoundHandler())
	client.maxAttempts = 1
	_, err := client.send(context.Background(), http.MethodPut, "http://127.0.0.1:1/upload/1?tempauth=secret", []byte("x"), nil, false)
	var unknown *unknownOutcomeError
	require.True(t, errors.As(err, &unknown))
	require.NotContains(t, err.Error(), "tempauth")
}

func TestUploadSmallFilesInOneRequestEach(t *testing.T) {
	fake := newFakeGraph(t)
	client, _ := newTestClient(t, fake)
	client.simpleUploadMaxSize = simpleUploadMaxSize
	_, err := client.Upload(context.Background(), testLocation, testFolder, testFiles)
	require.NoError(t, err)
	requireUploaded(t, fake)
	require.Zero(t, fake.sessionCount)
	// One request uploads each file; the first also creates the folder.
	require.Equal(t, len(testFiles), fake.requestCount)
}

func TestUploadInParallel(t *testing.T) {
	files := []File{{Name: "a.sql", Content: []byte("a")}, {Name: "b.csv", Content: []byte("b")}, {Name: "c.json", Content: []byte("c")}}
	var mu sync.Mutex
	inFlight := 0
	allInFlight := make(chan struct{})
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		inFlight++
		if inFlight == len(files) {
			close(allInFlight)
		}
		mu.Unlock()
		select {
		case <-allInFlight:
			writeJSON(w, http.StatusCreated, map[string]any{"id": r.URL.Path})
		case <-time.After(5 * time.Second):
			writeGraphError(w, http.StatusBadRequest, "uploadsNotConcurrent")
		}
	}))
	client.simpleUploadMaxSize = simpleUploadMaxSize
	client.maxConcurrentUploads = len(files)
	_, err := client.Upload(context.Background(), testLocation, testFolder, files)
	require.NoError(t, err)
}

func TestUploadRejectedAfterLostResponseIsUnconfirmed(t *testing.T) {
	puts := 0
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			puts++
			if puts == 1 {
				// The first attempt is committed but its response is lost.
				conn, _, err := w.(http.Hijacker).Hijack()
				require.NoError(t, err)
				_ = conn.Close()
				return
			}
		}
		writeGraphError(w, http.StatusForbidden, "accessDenied")
	}))
	client.simpleUploadMaxSize = simpleUploadMaxSize
	_, err := client.Upload(context.Background(), testLocation, testFolder, []File{{Name: "a.sql", Content: []byte("a")}})
	require.ErrorIs(t, err, ErrUnconfirmed)
	require.NotErrorIs(t, err, ErrNothingUploaded)
	require.Equal(t, 2, puts)
}

// cancelOnUpload cancels the export while the server holds an upload, so the
// client never learns the upload's outcome.
func cancelOnUpload(t *testing.T, cancel context.CancelFunc) func(r *http.Request) {
	return func(r *http.Request) {
		// Reading the body lets the server notice the client going away.
		_, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		cancel()
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
			t.Error("the canceled upload was not abandoned")
		}
	}
}

func TestUploadConfirmsTheFileInFlightWhenCanceled(t *testing.T) {
	files := []File{{Name: "a.sql", Content: []byte("a")}, {Name: "b.csv", Content: []byte("b")}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var (
		mu       sync.Mutex
		requests []string
	)
	upload := cancelOnUpload(t, cancel)
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()
		if r.Method == http.MethodPut {
			upload(r)
			return
		}
		// Graph committed the upload whose response the cancel discarded.
		writeJSON(w, http.StatusOK, map[string]any{
			"size":      1,
			"createdBy": map[string]any{"application": map[string]any{"id": testAppID}},
		})
	}))
	client.simpleUploadMaxSize = simpleUploadMaxSize
	_, err := client.Upload(ctx, testLocation, testFolder, files)
	require.ErrorIs(t, err, context.Canceled)
	require.NotErrorIs(t, err, ErrUnconfirmed)
	require.NotErrorIs(t, err, ErrNothingUploaded)
	folderPath := "/drives/d1/items/dest:/" + testFolder + "/"
	require.Equal(t, []string{
		"PUT " + folderPath + "a.sql:/content",
		"GET " + folderPath + "a.sql",
	}, requests)
}

func TestUploadReconcileAfterCancelIsBounded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	upload := cancelOnUpload(t, cancel)
	client, _ := newTestClient(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			upload(r)
			return
		}
		// The lookup never answers.
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
			t.Error("the reconciliation lookup was not bounded")
		}
	}))
	client.simpleUploadMaxSize = simpleUploadMaxSize
	client.reconcileGrace = 50 * time.Millisecond
	_, err := client.Upload(ctx, testLocation, testFolder, []File{{Name: "a.sql", Content: []byte("a")}})
	require.ErrorIs(t, err, ErrUnconfirmed)
}
