package command

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/action/result"
	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

const testRelease = "projects/example/releases/r1"

// sourcesTestServer serves one release and the sheets it is given. A sheet
// it was not given is refused, as for an identity without bb.sheets.get.
type sourcesTestServer struct {
	v1connect.UnimplementedReleaseServiceHandler
	v1connect.UnimplementedSheetServiceHandler

	release *v1pb.Release
	sheets  map[string][]byte

	mu            sync.Mutex
	sheetRequests []*v1pb.GetSheetRequest
}

func (s *sourcesTestServer) GetRelease(context.Context, *connect.Request[v1pb.GetReleaseRequest]) (*connect.Response[v1pb.Release], error) {
	return connect.NewResponse(s.release), nil
}

func (s *sourcesTestServer) GetSheet(_ context.Context, req *connect.Request[v1pb.GetSheetRequest]) (*connect.Response[v1pb.Sheet], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sheetRequests = append(s.sheetRequests, req.Msg)
	content, ok := s.sheets[req.Msg.GetName()]
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}
	return connect.NewResponse(&v1pb.Sheet{Name: req.Msg.GetName(), Content: content}), nil
}

func newSourcesTestClient(t *testing.T, s *sourcesTestServer) *client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(v1connect.NewReleaseServiceHandler(s))
	mux.Handle(v1connect.NewSheetServiceHandler(s))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	c, err := newClient(server.URL, credential{accessToken: "token"}, defaultClientOptions())
	require.NoError(t, err)
	t.Cleanup(c.close)
	return c
}

func TestReadSources(t *testing.T) {
	content := []byte("UPDATE account SET tier = 'basic';")
	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:])
	checkedOut := filepath.Join(t.TempDir(), "0001_backfill.sql")
	require.NoError(t, os.WriteFile(checkedOut, content, 0644))
	absent := filepath.Join(t.TempDir(), "absent.sql")

	refs := []result.SourceRef{
		{Release: testRelease, Version: "0001"},
		{Release: testRelease, Version: "0002"},
	}
	release := func(path string) *v1pb.Release {
		return &v1pb.Release{Name: testRelease, Files: []*v1pb.Release_File{
			{Version: "0001", Path: path, Sheet: "projects/example/sheets/s1", SheetSha256: hash},
			{Version: "0002", Path: absent, Sheet: "projects/example/sheets/s2", SheetSha256: hash},
		}}
	}

	t.Run("a checked-out file needs no sheet read", func(t *testing.T) {
		s := &sourcesTestServer{release: release(checkedOut), sheets: map[string][]byte{"projects/example/sheets/s2": []byte("SELECT 2;")}}
		sources := readSources(context.Background(), world.NewWorld(), newSourcesTestClient(t, s), refs)

		require.Equal(t, content, sources[refs[0].Key()])
		require.Equal(t, []byte("SELECT 2;"), sources[refs[1].Key()])
		require.Len(t, s.sheetRequests, 1)
		require.Equal(t, "projects/example/sheets/s2", s.sheetRequests[0].GetName())
		require.True(t, s.sheetRequests[0].GetRaw(), "the whole sheet is read, not its preview")
	})

	t.Run("an identity that cannot read sheets is refused once", func(t *testing.T) {
		s := &sourcesTestServer{release: release(absent)}
		sources := readSources(context.Background(), world.NewWorld(), newSourcesTestClient(t, s), refs)

		require.Empty(t, sources)
		require.Len(t, s.sheetRequests, 1)
	})

	t.Run("a sheet task reads its sheet", func(t *testing.T) {
		s := &sourcesTestServer{sheets: map[string][]byte{"projects/example/sheets/s9": content}}
		ref := result.SourceRef{Sheet: "projects/example/sheets/s9"}
		sources := readSources(context.Background(), world.NewWorld(), newSourcesTestClient(t, s), []result.SourceRef{ref})

		require.Equal(t, content, sources[ref.Key()])
	})
}

func TestReadCheckedOutFile(t *testing.T) {
	content := []byte("UPDATE account SET tier = 'basic';")
	sum := sha256.Sum256(content)
	path := filepath.Join(t.TempDir(), "0001_backfill.sql")
	require.NoError(t, os.WriteFile(path, content, 0644))

	got, ok := readCheckedOutFile(&v1pb.Release_File{Path: path, SheetSha256: hex.EncodeToString(sum[:])})
	require.True(t, ok)
	require.Equal(t, content, got)

	_, ok = readCheckedOutFile(&v1pb.Release_File{Path: path, SheetSha256: hex.EncodeToString(sha256.New().Sum(nil))})
	require.False(t, ok, "a file that differs from the release is not used")

	_, ok = readCheckedOutFile(&v1pb.Release_File{Path: filepath.Join(t.TempDir(), "missing.sql"), SheetSha256: hex.EncodeToString(sum[:])})
	require.False(t, ok)

	_, ok = readCheckedOutFile(&v1pb.Release_File{Path: path})
	require.False(t, ok, "a release file with no hash cannot be verified")
}
