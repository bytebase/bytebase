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
	// sheetErrs fail a sheet read with an error other than a refusal.
	sheetErrs map[string]error

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
	if err := s.sheetErrs[req.Msg.GetName()]; err != nil {
		return nil, err
	}
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
		sources := readTestSources(t, s, refs)

		require.Equal(t, content, sources[refs[0].Key()])
		require.Equal(t, []byte("SELECT 2;"), sources[refs[1].Key()])
		require.Len(t, s.sheetRequests, 1)
		require.Equal(t, "projects/example/sheets/s2", s.sheetRequests[0].GetName())
		require.True(t, s.sheetRequests[0].GetRaw(), "the whole sheet is read, not its preview")
	})

	t.Run("an identity that cannot read sheets is refused once", func(t *testing.T) {
		s := &sourcesTestServer{release: release(absent)}
		sources := readTestSources(t, s, refs)

		require.Empty(t, sources)
		require.Len(t, s.sheetRequests, 1)
	})

	t.Run("a sheet read that fails for another reason does not stop the rest", func(t *testing.T) {
		s := &sourcesTestServer{
			release:   release(absent),
			sheets:    map[string][]byte{"projects/example/sheets/s2": []byte("SELECT 2;")},
			sheetErrs: map[string]error{"projects/example/sheets/s1": connect.NewError(connect.CodeInternal, errors.New("boom"))},
		}
		sources := readTestSources(t, s, refs)

		require.NotContains(t, sources, refs[0].Key())
		require.Equal(t, []byte("SELECT 2;"), sources[refs[1].Key()])
		require.Len(t, s.sheetRequests, 2)
	})

	t.Run("a sheet task reads its sheet", func(t *testing.T) {
		s := &sourcesTestServer{sheets: map[string][]byte{"projects/example/sheets/s9": content}}
		ref := result.SourceRef{Sheet: "projects/example/sheets/s9"}
		sources := readTestSources(t, s, []result.SourceRef{ref})

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

	_, ok = readCheckedOutFile(&v1pb.Release_File{Path: filepath.Dir(path), SheetSha256: hex.EncodeToString(sum[:])})
	require.False(t, ok, "only an ordinary file is read")
}

func readTestSources(t *testing.T, s *sourcesTestServer, refs []result.SourceRef) map[string][]byte {
	t.Helper()
	ctx := context.Background()
	w := world.NewWorld()
	c := newSourcesTestClient(t, s)
	return readSources(ctx, w, c, newReleaseReader(w, c), refs)
}

// resultTestServer serves one rollout, its task runs and the logs it is
// given. A log it was not given fails to read.
type resultTestServer struct {
	v1connect.UnimplementedRolloutServiceHandler

	rollout    *v1pb.Rollout
	rolloutErr error
	taskRuns   []*v1pb.TaskRun
	logs       map[string]*v1pb.TaskRunLog

	mu    sync.Mutex
	calls map[string]int
}

func (s *resultTestServer) count(method string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls == nil {
		s.calls = map[string]int{}
	}
	s.calls[method]++
}

func (s *resultTestServer) GetRollout(context.Context, *connect.Request[v1pb.GetRolloutRequest]) (*connect.Response[v1pb.Rollout], error) {
	s.count("GetRollout")
	if s.rolloutErr != nil {
		return nil, s.rolloutErr
	}
	return connect.NewResponse(s.rollout), nil
}

func (s *resultTestServer) ListTaskRuns(context.Context, *connect.Request[v1pb.ListTaskRunsRequest]) (*connect.Response[v1pb.ListTaskRunsResponse], error) {
	s.count("ListTaskRuns")
	return connect.NewResponse(&v1pb.ListTaskRunsResponse{TaskRuns: s.taskRuns}), nil
}

func (s *resultTestServer) GetTaskRunLog(_ context.Context, req *connect.Request[v1pb.GetTaskRunLogRequest]) (*connect.Response[v1pb.TaskRunLog], error) {
	s.count("GetTaskRunLog")
	log, ok := s.logs[req.Msg.GetParent()]
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("log unavailable"))
	}
	return connect.NewResponse(log), nil
}

// newResultTest returns a server holding a prod stage with two finished
// tasks, a world that last saw the first of them running, and a client.
func newResultTest(t *testing.T) (*resultTestServer, *world.World, *client) {
	t.Helper()
	const stage = "projects/example/plans/101/rollout/stages/prod"
	task := func(id string, status v1pb.Task_Status) *v1pb.Task {
		return &v1pb.Task{Name: stage + "/tasks/" + id, Status: status, Target: "instances/prod/databases/" + id}
	}
	rollout := func(first v1pb.Task_Status) *v1pb.Rollout {
		return &v1pb.Rollout{
			Name: "projects/example/plans/101/rollout",
			Stages: []*v1pb.Stage{{
				Name:        stage,
				Environment: "environments/prod",
				Tasks:       []*v1pb.Task{task("hr", first), task("crm", v1pb.Task_DONE)},
			}},
		}
	}
	command := &v1pb.TaskRunLogEntry{
		Type: v1pb.TaskRunLogEntry_COMMAND_EXECUTE,
		CommandExecute: &v1pb.TaskRunLogEntry_CommandExecute{
			Statement: "UPDATE t SET a = 1",
			Response:  &v1pb.TaskRunLogEntry_CommandExecute_CommandResponse{AffectedRows: 7},
		},
	}
	s := &resultTestServer{
		rollout: rollout(v1pb.Task_DONE),
		taskRuns: []*v1pb.TaskRun{
			{Name: stage + "/tasks/hr/taskRuns/1", Status: v1pb.TaskRun_DONE},
			{Name: stage + "/tasks/crm/taskRuns/1", Status: v1pb.TaskRun_DONE},
		},
		logs: map[string]*v1pb.TaskRunLog{
			stage + "/tasks/hr/taskRuns/1":  {Entries: []*v1pb.TaskRunLogEntry{command}},
			stage + "/tasks/crm/taskRuns/1": {Entries: []*v1pb.TaskRunLogEntry{command}},
		},
	}
	mux := http.NewServeMux()
	mux.Handle(v1connect.NewRolloutServiceHandler(s))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	c, err := newClient(server.URL, credential{accessToken: "token"}, defaultClientOptions())
	require.NoError(t, err)
	t.Cleanup(c.close)

	w := world.NewWorld()
	w.URL = "https://bytebase.example.com/"
	w.TargetStage = "environments/prod"
	w.Rollout = rollout(v1pb.Task_RUNNING)
	return s, w, c
}

func TestCollectResult(t *testing.T) {
	t.Run("reads the final state and each database's log", func(t *testing.T) {
		s, w, c := newResultTest(t)

		got := collectResult(context.Background(), w, c)

		require.Equal(t, result.Done, got.Outcome)
		require.Equal(t, "https://bytebase.example.com/projects/example/plans/101/rollout", got.URL)
		tasks := got.Stages[0].Tasks
		require.Len(t, tasks, 2)
		require.Equal(t, int64(7), *tasks[0].AffectedRows)
		require.Equal(t, int64(7), *tasks[1].AffectedRows)
		require.Equal(t, 2, s.calls["GetTaskRunLog"])
	})

	t.Run("a failed refresh reports the last state seen and asks nothing more", func(t *testing.T) {
		s, w, c := newResultTest(t)
		s.rolloutErr = connect.NewError(connect.CodeInternal, errors.New("boom"))

		got := collectResult(context.Background(), w, c)

		require.Equal(t, result.Interrupted, got.Outcome, "the first task was last seen running")
		require.Len(t, got.Stages[0].Tasks, 2)
		require.Nil(t, got.Stages[0].Tasks[1].AffectedRows)
		require.Zero(t, s.calls["ListTaskRuns"])
		require.Zero(t, s.calls["GetTaskRunLog"])
	})

	t.Run("a log that cannot be read costs only that task's detail", func(t *testing.T) {
		s, w, c := newResultTest(t)
		delete(s.logs, "projects/example/plans/101/rollout/stages/prod/tasks/hr/taskRuns/1")

		got := collectResult(context.Background(), w, c)

		tasks := got.Stages[0].Tasks
		require.True(t, tasks[0].LogUnavailable)
		require.Nil(t, tasks[0].AffectedRows)
		require.False(t, tasks[1].LogUnavailable)
		require.Equal(t, int64(7), *tasks[1].AffectedRows)
	})

	t.Run("a canceled job still gets its result", func(t *testing.T) {
		_, w, c := newResultTest(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		got := collectResult(ctx, w, c)

		require.Equal(t, result.Done, got.Outcome)
		require.Equal(t, int64(7), *got.Stages[0].Tasks[0].AffectedRows)
	})

	t.Run("no rollout, no result", func(t *testing.T) {
		_, w, c := newResultTest(t)
		w.Rollout = nil

		require.Nil(t, collectResult(context.Background(), w, c))
	})
}
