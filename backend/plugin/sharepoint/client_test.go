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
	var waits []time.Duration
	client := &graphClient{
		baseURL:     srv.URL,
		clientID:    testAppID,
		token:       func(context.Context) (string, error) { return "test-token", nil },
		httpClient:  srv.Client(),
		chunkSize:   4,
		maxAttempts: 3,
		wait: func(_ context.Context, d time.Duration) error {
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
			require.Equal(t, "id,name,webUrl,folder,root,parentReference", r.URL.Query().Get("$select"))
			if itemStatus != http.StatusOK {
				writeGraphError(w, itemStatus, "accessDenied")
				return
			}
			writeJSON(w, http.StatusOK, item)
		case "/drives/d1":
			writeJSON(w, http.StatusOK, map[string]any{"name": "Documents", "webUrl": "https://contoso.sharepoint.com/sites/finance/Shared%20Documents"})
		case "/sites/contoso.sharepoint.com,s1,w1":
			writeJSON(w, http.StatusOK, map[string]any{"displayName": "Finance"})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			writeGraphError(w, http.StatusNotFound, "itemNotFound")
		}
	})
}

func TestResolve(t *testing.T) {
	const canonical = "https://contoso.sharepoint.com/sites/finance/Shared%20Documents/My%20Exports/project%20a"
	parent := map[string]any{"driveId": "d1", "siteId": "contoso.sharepoint.com,s1,w1"}

	t.Run("library root", func(t *testing.T) {
		client, _ := newTestClient(t, resolveServer(t, http.StatusOK, map[string]any{
			"id": "root1", "name": "root", "webUrl": "https://contoso.sharepoint.com/sites/finance/Shared%20Documents",
			"folder": map[string]any{"childCount": 2}, "root": map[string]any{}, "parentReference": parent,
		}))
		loc, err := client.Resolve(context.Background(), canonical)
		require.NoError(t, err)
		require.Equal(t, &Location{
			DriveID: "d1", ItemID: "root1", SiteName: "Finance", LibraryName: "Documents",
			WebURL: "https://contoso.sharepoint.com/sites/finance/Shared%20Documents",
		}, loc)
	})

	t.Run("folder", func(t *testing.T) {
		client, _ := newTestClient(t, resolveServer(t, http.StatusOK, map[string]any{
			"id": "item1", "name": "project a", "webUrl": canonical, "folder": map[string]any{"childCount": 0},
			"parentReference": map[string]any{"driveId": "d1", "siteId": "contoso.sharepoint.com,s1,w1", "path": "/drives/d1/root:/My%20Exports"},
		}))
		loc, err := client.Resolve(context.Background(), canonical)
		require.NoError(t, err)
		require.Equal(t, &Location{
			DriveID: "d1", ItemID: "item1", SiteName: "Finance", LibraryName: "Documents",
			FolderPath: "My Exports/project a", WebURL: canonical,
		}, loc)
	})

	t.Run("file", func(t *testing.T) {
		client, _ := newTestClient(t, resolveServer(t, http.StatusOK, map[string]any{
			"id": "item1", "name": "project a", "file": map[string]any{}, "parentReference": parent,
		}))
		_, err := client.Resolve(context.Background(), canonical)
		require.ErrorContains(t, err, "is a file")
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

func TestFolderPath(t *testing.T) {
	require.Equal(t, "a", folderPath("/drives/d1/root:", "a"))
	require.Equal(t, "a/b c/d", folderPath("/drives/d1/root:/a/b%20c", "d"))
	require.Equal(t, "a/d", folderPath("/drive/root:/a", "d"))
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
	// folderIDs maps folder IDs to names.
	folderIDs map[string]string
	// files is keyed by "folderName/fileName".
	files         map[string]fakeFile
	sessions      map[string]*fakeSession
	sessionCount  int
	contentRanges []string
	// override handles a request instead of the fake when it returns true.
	override func(w http.ResponseWriter, r *http.Request) bool
	// dropUploadResponse stores the next chunk and then drops the connection.
	dropUploadResponse bool
}

func newFakeGraph(t *testing.T) *fakeGraph {
	return &fakeGraph{t: t, folderIDs: map[string]string{}, files: map[string]fakeFile{}, sessions: map[string]*fakeSession{}}
}

func (f *fakeGraph) folderID(name string) string {
	for id, folderName := range f.folderIDs {
		if folderName == name {
			return id
		}
	}
	return ""
}

func (f *fakeGraph) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.override != nil && f.override(w, r) {
		return
	}
	if strings.HasPrefix(r.URL.Path, "/upload/") {
		f.serveUpload(w, r)
		return
	}
	require.Equal(f.t, "Bearer test-token", r.Header.Get("Authorization"))
	rest, ok := strings.CutPrefix(r.URL.Path, "/drives/d1/items/")
	if !ok {
		writeGraphError(w, http.StatusNotFound, "itemNotFound")
		return
	}
	if rest == "dest/children" && r.Method == http.MethodPost {
		var body map[string]any
		require.NoError(f.t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(f.t, "fail", body["@microsoft.graph.conflictBehavior"])
		name, ok := body["name"].(string)
		require.True(f.t, ok)
		if f.folderID(name) != "" {
			writeGraphError(w, http.StatusConflict, "nameAlreadyExists")
			return
		}
		id := fmt.Sprintf("folder%d", len(f.folderIDs)+1)
		f.folderIDs[id] = name
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "name": name, "webUrl": "https://sp/" + id, "folder": map[string]any{}})
		return
	}
	itemID, sub, _ := strings.Cut(rest, ":/")
	if itemID == "dest" && r.Method == http.MethodGet {
		id := f.folderID(sub)
		if id == "" {
			writeGraphError(w, http.StatusNotFound, "itemNotFound")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "webUrl": "https://sp/" + id, "folder": map[string]any{}})
		return
	}
	folderName, ok := f.folderIDs[itemID]
	if !ok {
		writeGraphError(w, http.StatusNotFound, "itemNotFound")
		return
	}
	switch {
	case r.Method == http.MethodPost && strings.HasSuffix(sub, ":/createUploadSession"):
		key := folderName + "/" + strings.TrimSuffix(sub, ":/createUploadSession")
		if _, exists := f.files[key]; exists {
			writeGraphError(w, http.StatusConflict, "nameAlreadyExists")
			return
		}
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
	testLocation = &Location{DriveID: "d1", ItemID: "dest"}
	testFiles    = []File{
		{Name: "statement.sql", Content: []byte("SELECT 1;\n")},
		{Name: "result #1.csv", Content: []byte("a,b\n1,2\n")},
		{Name: "empty.json", Content: []byte{}},
	}
)

func requireDelivered(t *testing.T, fake *fakeGraph) {
	t.Helper()
	for _, file := range testFiles {
		got, ok := fake.files[testFolder+"/"+file.Name]
		require.True(t, ok, file.Name)
		require.Equal(t, string(file.Content), string(got.content))
	}
}

func TestDeliver(t *testing.T) {
	fake := newFakeGraph(t)
	client, waits := newTestClient(t, fake)
	delivery, err := client.Deliver(context.Background(), testLocation, testFolder, testFiles)
	require.NoError(t, err)
	require.Equal(t, &Delivery{FolderName: testFolder, WebURL: "https://sp/folder1"}, delivery)
	requireDelivered(t, fake)
	require.Equal(t, []string{
		"bytes 0-3/10", "bytes 4-7/10", "bytes 8-9/10",
		"bytes 0-3/8", "bytes 4-7/8",
	}, fake.contentRanges)
	require.Empty(t, *waits)

	// Retrying the same export reuses the folder and the delivered files.
	delivery, err = client.Deliver(context.Background(), testLocation, testFolder, testFiles)
	require.NoError(t, err)
	require.Equal(t, &Delivery{FolderName: testFolder, WebURL: "https://sp/folder1"}, delivery)
	require.Len(t, fake.folderIDs, 1)
	require.Equal(t, 2, fake.sessionCount)
}

func TestDeliverReusesExistingFolder(t *testing.T) {
	fake := newFakeGraph(t)
	fake.folderIDs["existing"] = testFolder
	client, _ := newTestClient(t, fake)
	delivery, err := client.Deliver(context.Background(), testLocation, testFolder, testFiles)
	require.NoError(t, err)
	require.Equal(t, "https://sp/existing", delivery.WebURL)
	requireDelivered(t, fake)
}

func TestDeliverFileConflict(t *testing.T) {
	files := []File{{Name: "result.csv", Content: []byte("a,b\n")}}
	tests := []struct {
		name     string
		existing fakeFile
		wantErr  string
	}{
		{name: "delivered by this application", existing: fakeFile{content: []byte("x,y\n"), appID: testAppID}},
		{name: "different application", existing: fakeFile{content: []byte("a,b\n"), appID: "someone-else"}, wantErr: "a different file named"},
		{name: "different size", existing: fakeFile{content: []byte("a,b,c\n"), appID: testAppID}, wantErr: "a different file named"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeGraph(t)
			fake.folderIDs["existing"] = testFolder
			fake.files[testFolder+"/result.csv"] = tc.existing
			client, _ := newTestClient(t, fake)
			_, err := client.Deliver(context.Background(), testLocation, testFolder, files)
			require.Zero(t, fake.sessionCount)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
			require.NotErrorIs(t, err, ErrUnconfirmed)
		})
	}
}

func TestDeliverHonorsRetryAfter(t *testing.T) {
	fake := newFakeGraph(t)
	throttled := false
	fake.override = func(w http.ResponseWriter, r *http.Request) bool {
		if throttled || r.URL.Path != "/drives/d1/items/dest/children" {
			return false
		}
		throttled = true
		w.Header().Set("Retry-After", "2")
		writeGraphError(w, http.StatusTooManyRequests, "activityLimitReached")
		return true
	}
	client, waits := newTestClient(t, fake)
	_, err := client.Deliver(context.Background(), testLocation, testFolder, testFiles)
	require.NoError(t, err)
	require.Equal(t, []time.Duration{2 * time.Second}, *waits)
	requireDelivered(t, fake)
}

func TestDeliverResumesAfterLostChunkResponse(t *testing.T) {
	fake := newFakeGraph(t)
	fake.override = func(_ http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPut && r.Header.Get("Content-Range") == "bytes 4-7/10" && len(fake.contentRanges) == 1 {
			fake.dropUploadResponse = true
		}
		return false
	}
	client, waits := newTestClient(t, fake)
	_, err := client.Deliver(context.Background(), testLocation, testFolder, testFiles)
	require.NoError(t, err)
	requireDelivered(t, fake)
	require.Equal(t, []time.Duration{time.Second}, *waits)
}

func TestDeliverUnconfirmed(t *testing.T) {
	fake := newFakeGraph(t)
	fake.override = func(w http.ResponseWriter, r *http.Request) bool {
		if !strings.HasPrefix(r.URL.Path, "/upload/") {
			return false
		}
		writeGraphError(w, http.StatusGatewayTimeout, "gatewayTimeout")
		return true
	}
	client, waits := newTestClient(t, fake)
	_, err := client.Deliver(context.Background(), testLocation, testFolder, testFiles)
	require.ErrorIs(t, err, ErrUnconfirmed)
	require.ErrorContains(t, err, "statement.sql")
	require.NotContains(t, err.Error(), "tempauth")
	require.Equal(t, []time.Duration{time.Second, 2 * time.Second}, *waits)
}

func TestDeliverForbidden(t *testing.T) {
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeGraphError(w, http.StatusForbidden, "accessDenied")
	}))
	_, err := client.Deliver(context.Background(), testLocation, testFolder, testFiles)
	require.ErrorIs(t, err, ErrNotFoundOrNoAccess)
}

func TestTransportErrorOmitsURL(t *testing.T) {
	client, _ := newTestClient(t, http.NotFoundHandler())
	client.maxAttempts = 1
	_, err := client.send(context.Background(), http.MethodPut, "http://127.0.0.1:1/upload/1?tempauth=secret", []byte("x"), nil, false)
	var unknown *unknownOutcomeError
	require.True(t, errors.As(err, &unknown))
	require.NotContains(t, err.Error(), "tempauth")
}
