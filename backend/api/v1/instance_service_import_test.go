package v1

import (
	"context"
	"errors"
	"math"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func neonImportEntry(id, branch string) *v1pb.InstanceImport {
	return &v1pb.InstanceImport{
		InstanceId:         id,
		Instance:           &v1pb.Instance{Title: id},
		Username:           "owner",
		ConnectionDatabase: "app",
		Source:             &v1pb.InstanceImport_NeonBranch{NeonBranch: &v1pb.NeonBranchSource{ProjectId: "project", BranchId: branch}},
	}
}

func neonImportRequest(entries ...*v1pb.InstanceImport) *v1pb.ImportInstancesRequest {
	return &v1pb.ImportInstancesRequest{
		Credential: &v1pb.ImportInstancesRequest_NeonCredential{NeonCredential: &v1pb.NeonCredential{ApiKey: "key"}},
		Imports:    entries,
	}
}

func statusErrorInfo(t *testing.T, status *statuspb.Status) *errdetails.ErrorInfo {
	t.Helper()
	for _, detail := range status.GetDetails() {
		info := &errdetails.ErrorInfo{}
		if detail.UnmarshalTo(info) == nil {
			return info
		}
	}
	return &errdetails.ErrorInfo{}
}

func violatedFields(t *testing.T, err error) []string {
	t.Helper()
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	status := importStatus(connectErr)
	require.Empty(t, statusErrorInfo(t, status).Reason)
	var fields []string
	for _, detail := range status.Details {
		badRequest := &errdetails.BadRequest{}
		if detail.UnmarshalTo(badRequest) != nil {
			continue
		}
		for _, violation := range badRequest.FieldViolations {
			fields = append(fields, violation.Field)
		}
	}
	return fields
}

func TestImportAdmission(t *testing.T) {
	require.NoError(t, validateImportRequest(neonImportRequest(neonImportEntry("one", "one"), neonImportEntry("two", "two"))))
	require.EqualError(t,
		validateImportRequest(neonImportRequest(neonImportEntry("same", "one"), neonImportEntry("same", "two"))),
		"invalid_argument: imports[1].instance_id: duplicates imports[0].instance_id")

	edited := func(edit func(*v1pb.InstanceImport)) []*v1pb.InstanceImport {
		entry := neonImportEntry("one", "one")
		edit(entry)
		return []*v1pb.InstanceImport{neonImportEntry("zero", "zero"), entry}
	}
	for _, tc := range []struct {
		name    string
		request *v1pb.ImportInstancesRequest
		field   string
	}{
		{name: "no credential", request: &v1pb.ImportInstancesRequest{Imports: []*v1pb.InstanceImport{neonImportEntry("one", "one")}}, field: "credential"},
		{name: "source without its credential", request: &v1pb.ImportInstancesRequest{Imports: []*v1pb.InstanceImport{neonImportEntry("one", "one")}}, field: "imports[0].neon_branch"},
		{name: "empty API key", request: &v1pb.ImportInstancesRequest{Credential: &v1pb.ImportInstancesRequest_NeonCredential{NeonCredential: &v1pb.NeonCredential{}}, Imports: []*v1pb.InstanceImport{neonImportEntry("one", "one")}}, field: "neon_credential.api_key"},
		{name: "no entries", request: neonImportRequest(), field: "imports"},
		{name: "too many entries", request: neonImportRequest(make([]*v1pb.InstanceImport, importEntryLimit+1)...), field: "imports"},
		{name: "nil entry", request: neonImportRequest(neonImportEntry("zero", "zero"), nil), field: "imports[1]"},
		{name: "invalid instance ID", request: neonImportRequest(edited(func(e *v1pb.InstanceImport) { e.InstanceId = "Not Valid" })...), field: "imports[1].instance_id"},
		{name: "duplicate instance ID", request: neonImportRequest(neonImportEntry("same", "one"), neonImportEntry("same", "two")), field: "imports[1].instance_id"},
		{name: "duplicate source", request: neonImportRequest(neonImportEntry("one", "same"), neonImportEntry("two", "same")), field: "imports[1].source"},
		{name: "no instance", request: neonImportRequest(edited(func(e *v1pb.InstanceImport) { e.Instance = nil })...), field: "imports[1].instance"},
		{name: "blank title", request: neonImportRequest(edited(func(e *v1pb.InstanceImport) { e.Instance.Title = " " })...), field: "imports[1].instance.title"},
		{name: "negative sync interval", request: neonImportRequest(edited(func(e *v1pb.InstanceImport) { e.Instance.SyncInterval = durationpb.New(-1) })...), field: "imports[1].instance.sync_interval"},
		{name: "engine stated", request: neonImportRequest(edited(func(e *v1pb.InstanceImport) { e.Instance.Engine = v1pb.Engine_MYSQL })...), field: "imports[1].instance.engine"},
		{name: "vendor stated", request: neonImportRequest(edited(func(e *v1pb.InstanceImport) { e.Instance.Vendor = v1pb.Vendor_NEON })...), field: "imports[1].instance.vendor"},
		{name: "data source stated", request: neonImportRequest(edited(func(e *v1pb.InstanceImport) { e.Instance.DataSources = []*v1pb.DataSource{{Host: "db.example.com"}} })...), field: "imports[1].instance.data_sources"},
		{name: "no username", request: neonImportRequest(edited(func(e *v1pb.InstanceImport) { e.Username = "" })...), field: "imports[1].username"},
		{name: "no connection database", request: neonImportRequest(edited(func(e *v1pb.InstanceImport) { e.ConnectionDatabase = "" })...), field: "imports[1].connection_database"},
		{name: "empty supplied password", request: neonImportRequest(edited(func(e *v1pb.InstanceImport) { e.Password = new("") })...), field: "imports[1].password"},
		{name: "no source", request: neonImportRequest(edited(func(e *v1pb.InstanceImport) { e.Source = nil })...), field: "imports[1].source"},
		{name: "malformed Neon project ID", request: neonImportRequest(edited(func(e *v1pb.InstanceImport) { e.GetNeonBranch().ProjectId = "../secrets" })...), field: "imports[1].neon_branch.project_id"},
		{name: "malformed Neon branch ID", request: neonImportRequest(edited(func(e *v1pb.InstanceImport) { e.GetNeonBranch().BranchId = "" })...), field: "imports[1].neon_branch.branch_id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Contains(t, violatedFields(t, validateImportRequest(tc.request)), tc.field)
		})
	}
}

func TestImportParallelOrderingAndOutcomes(t *testing.T) {
	var entries []*v1pb.InstanceImport
	for _, id := range []string{"one", "two", "three", "four", "five"} {
		entries = append(entries, neonImportEntry(id, id))
	}
	entered := make(chan struct{}, importWorkerCount)
	release := make(chan struct{})
	var active, maxActive atomic.Int32
	operations := importOperations{
		resolve: func(_ context.Context, entry *v1pb.InstanceImport) (*importTarget, error) {
			n := active.Add(1)
			for old := maxActive.Load(); n > old; old = maxActive.Load() {
				if maxActive.CompareAndSwap(old, n) {
					break
				}
			}
			entered <- struct{}{}
			<-release
			active.Add(-1)
			if entry.InstanceId == "two" {
				return nil, newVendorError(connect.CodeFailedPrecondition, reasonVendorPasswordRequired, "Enter the database role password", v1pb.Vendor_NEON)
			}
			return &importTarget{engine: v1pb.Engine_POSTGRES, vendor: v1pb.Vendor_NEON, dataSource: &v1pb.DataSource{Host: "ep.neon.tech"}}, nil
		},
		create: func(_ context.Context, req *connect.Request[v1pb.CreateInstanceRequest]) (*connect.Response[v1pb.Instance], error) {
			if !req.Msg.ValidateOnly && req.Msg.InstanceId == "three" {
				return nil, connect.NewError(connect.CodeInternal, errors.New("commit response lost"))
			}
			return connect.NewResponse(&v1pb.Instance{Name: "instances/" + req.Msg.InstanceId}), nil
		},
	}
	done := make(chan []*v1pb.InstanceImportResult, 1)
	go func() {
		done <- runImport(context.Background(), neonImportRequest(entries...), true, operations)
	}()
	for range importWorkerCount {
		<-entered
	}
	require.Equal(t, int32(importWorkerCount), active.Load())
	close(release)
	results := <-done
	require.LessOrEqual(t, maxActive.Load(), int32(importWorkerCount))
	require.Len(t, results, 5)
	require.Equal(t, "instances/one", results[0].GetImported().Name)
	require.Equal(t, v1pb.InstanceImportFailure_NOT_CREATED, results[1].GetFailure().CreationOutcome)
	require.Equal(t, reasonVendorPasswordRequired, statusErrorInfo(t, results[1].GetFailure().Status).Reason)
	require.Equal(t, v1pb.InstanceImportFailure_UNKNOWN, results[2].GetFailure().CreationOutcome)
	require.Equal(t, reasonInstanceCreationFailed, statusErrorInfo(t, results[2].GetFailure().Status).Reason)
	require.Equal(t, "instances/five", results[4].GetImported().Name)
}

func TestImportCanceledRequestDoesNotCreate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results := runImport(ctx, neonImportRequest(neonImportEntry("one", "one")), true, importOperations{})
	failure := results[0].GetFailure()
	require.Equal(t, v1pb.InstanceImportFailure_NOT_CREATED, failure.CreationOutcome)
	require.Equal(t, int32(connect.CodeCanceled), failure.Status.Code)
	require.Equal(t, reasonRequestCanceled, statusErrorInfo(t, failure.Status).Reason)
}

func TestImportActivationFollowsLicense(t *testing.T) {
	for _, tc := range []struct {
		name           string
		unified        bool
		activatedLimit int
		want           bool
	}{
		{name: "unified license", unified: true, activatedLimit: 10, want: true},
		{name: "split license with activated instances", activatedLimit: 5, want: true},
		{name: "unlimited activated instances", activatedLimit: math.MaxInt, want: true},
		{name: "no activated instances", activatedLimit: 0, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, importActivation(tc.unified, tc.activatedLimit))
		})
	}
}

func TestImportBuildsInstanceFromResolvedSource(t *testing.T) {
	entry := neonImportEntry("one", "one")
	entry.Instance.Environment = new("environments/prod")
	entry.Instance.ExternalLink = "https://console.example.com"
	entry.Instance.SyncInterval = durationpb.New(6 * time.Hour)
	entry.Instance.Activation = true
	for _, activation := range []bool{true, false} {
		var created atomic.Int32
		operations := importOperations{
			resolve: func(context.Context, *v1pb.InstanceImport) (*importTarget, error) {
				return &importTarget{engine: v1pb.Engine_POSTGRES, vendor: v1pb.Vendor_NEON, dataSource: &v1pb.DataSource{Id: "admin", Host: "ep.example.com", Username: "owner"}}, nil
			},
			create: func(_ context.Context, req *connect.Request[v1pb.CreateInstanceRequest]) (*connect.Response[v1pb.Instance], error) {
				instance := req.Msg.Instance
				require.Equal(t, v1pb.Engine_POSTGRES, instance.Engine)
				require.Equal(t, v1pb.Vendor_NEON, instance.Vendor)
				require.Equal(t, activation, instance.Activation)
				if req.Msg.InstanceId == "one" {
					require.Equal(t, "environments/prod", instance.GetEnvironment())
					require.Equal(t, "https://console.example.com", instance.ExternalLink)
					require.Equal(t, 6*time.Hour, instance.SyncInterval.AsDuration())
				} else {
					require.Nil(t, instance.SyncInterval)
				}
				require.Equal(t, "ep.example.com", instance.DataSources[0].Host)
				if !req.Msg.ValidateOnly {
					created.Add(1)
				}
				return connect.NewResponse(&v1pb.Instance{Name: "instances/" + req.Msg.InstanceId, Activation: instance.Activation}), nil
			},
		}
		results := runImport(context.Background(), neonImportRequest(entry, neonImportEntry("two", "two")), activation, operations)
		require.Equal(t, int32(2), created.Load())
		for _, result := range results {
			require.Equal(t, activation, result.GetImported().Activation)
		}
		require.Equal(t, v1pb.Engine_ENGINE_UNSPECIFIED, entry.Instance.Engine, "the request entry is left as sent")
	}
}

// An Instance field reaches CreateInstance through an import entry unless it
// is classified here as decided by the source or the license. A new field
// fails this test until it is given a role.
func TestImportInstanceFieldRoles(t *testing.T) {
	const (
		fromEntry   = "taken from the entry"
		fromSource  = "decided by the source; rejected when set"
		fromLicense = "decided by the license; overwritten"
		serverOwned = "ignored by CreateInstance"
	)
	roles := map[string]string{
		"title":          fromEntry,
		"external_link":  fromEntry,
		"environment":    fromEntry,
		"sync_interval":  fromEntry,
		"sync_databases": fromEntry,
		"labels":         fromEntry,
		"engine":         fromSource,
		"vendor":         fromSource,
		"data_sources":   fromSource,
		"activation":     fromLicense,
		"name":           serverOwned,
		"state":          serverOwned,
		"engine_version": serverOwned,
		"roles":          serverOwned,
		"last_sync_time": serverOwned,
	}
	fields := (&v1pb.Instance{}).ProtoReflect().Descriptor().Fields()
	require.Equal(t, len(roles), fields.Len())
	for i := range fields.Len() {
		require.Contains(t, roles, string(fields.Get(i).Name()))
	}
}

func TestImportEntryErrorModel(t *testing.T) {
	connectionFailure := connect.NewError(connect.CodeInvalidArgument, errors.New("failed to connect to database: password authentication failed for user \"owner\""))
	connectionFailure.Meta().Set(connectionCategoryHeader, connectionCategoryAuthFailed)
	for _, tc := range []struct {
		name string
		// The validation pass fails when validate is set; otherwise the create pass does.
		validate bool
		err      error
		code     connect.Code
		reason   string
		outcome  v1pb.InstanceImportFailure_CreationOutcome
		metadata map[string]string
	}{
		{name: "connection test", validate: true, err: connectionFailure, code: connect.CodeFailedPrecondition, reason: reasonInstanceConnectionFailed, outcome: v1pb.InstanceImportFailure_NOT_CREATED, metadata: map[string]string{"category": connectionCategoryAuthFailed}},
		{name: "settings rejected", validate: true, err: connect.NewError(connect.CodeInvalidArgument, errors.New("invalid label")), code: connect.CodeInvalidArgument, reason: reasonInstanceSettingsInvalid, outcome: v1pb.InstanceImportFailure_NOT_CREATED},
		{name: "environment missing", validate: true, err: connect.NewError(connect.CodeNotFound, &environmentNotFoundError{id: "nope"}), code: connect.CodeInvalidArgument, reason: reasonInstanceSettingsInvalid, outcome: v1pb.InstanceImportFailure_NOT_CREATED},
		{name: "destination deleted", validate: true, err: connect.NewError(connect.CodeNotFound, errors.New("project not found")), code: connect.CodeNotFound, reason: reasonDestinationNotFound, outcome: v1pb.InstanceImportFailure_NOT_CREATED},
		{name: "validation hit a server error", validate: true, err: connect.NewError(connect.CodeInternal, errors.New("metadata read failed")), code: connect.CodeInternal, reason: reasonInstanceValidationFailed, outcome: v1pb.InstanceImportFailure_NOT_CREATED},
		{name: "canceled during validation", validate: true, err: connect.NewError(connect.CodeCanceled, context.Canceled), code: connect.CodeCanceled, reason: reasonRequestCanceled, outcome: v1pb.InstanceImportFailure_NOT_CREATED},
		{name: "instance limit", err: connect.NewError(connect.CodeResourceExhausted, errors.New("limit reached")), code: connect.CodeResourceExhausted, reason: reasonInstanceLimitExceeded, outcome: v1pb.InstanceImportFailure_NOT_CREATED},
		{name: "instance ID taken", err: connect.NewError(connect.CodeAlreadyExists, errors.New("exists")), code: connect.CodeAlreadyExists, reason: reasonInstanceIDExists, outcome: v1pb.InstanceImportFailure_NOT_CREATED},
		{name: "write failure", err: connect.NewError(connect.CodeInternal, errors.New("connection reset")), code: connect.CodeInternal, reason: reasonInstanceCreationFailed, outcome: v1pb.InstanceImportFailure_UNKNOWN},
		{name: "canceled during the write", err: connect.NewError(connect.CodeCanceled, context.Canceled), code: connect.CodeCanceled, reason: reasonInstanceCreationFailed, outcome: v1pb.InstanceImportFailure_UNKNOWN},
		{name: "deadline during the write", err: connect.NewError(connect.CodeDeadlineExceeded, context.DeadlineExceeded), code: connect.CodeDeadlineExceeded, reason: reasonInstanceCreationFailed, outcome: v1pb.InstanceImportFailure_UNKNOWN},
	} {
		t.Run(tc.name, func(t *testing.T) {
			operations := importOperations{
				resolve: func(context.Context, *v1pb.InstanceImport) (*importTarget, error) {
					return &importTarget{dataSource: &v1pb.DataSource{}}, nil
				},
				create: func(_ context.Context, req *connect.Request[v1pb.CreateInstanceRequest]) (*connect.Response[v1pb.Instance], error) {
					if req.Msg.ValidateOnly == tc.validate {
						return nil, tc.err
					}
					return connect.NewResponse(&v1pb.Instance{}), nil
				},
			}
			failure := runImport(context.Background(), neonImportRequest(neonImportEntry("one", "one")), true, operations)[0].GetFailure()
			require.Equal(t, tc.outcome, failure.CreationOutcome)
			require.Equal(t, int32(tc.code), failure.Status.Code)
			info := statusErrorInfo(t, failure.Status)
			require.Equal(t, tc.reason, info.Reason)
			require.Equal(t, importErrorDomain, info.Domain)
			require.Equal(t, tc.outcome == v1pb.InstanceImportFailure_UNKNOWN, info.Reason == reasonInstanceCreationFailed, "INSTANCE_CREATION_FAILED and UNKNOWN appear only together")
			for key, value := range tc.metadata {
				require.Equal(t, value, info.Metadata[key])
			}
			require.NotContains(t, failure.Status.Message, "owner")
		})
	}
}

func TestImportDestinationError(t *testing.T) {
	missing := importDestinationError(connect.NewError(connect.CodeNotFound, errors.New("project \"projects/gone\" not found")))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(missing))
	require.Equal(t, reasonDestinationNotFound, statusErrorInfo(t, importStatus(missing)).Reason)

	malformed := importDestinationError(connect.NewError(connect.CodeInvalidArgument, errors.New("invalid instance parent")))
	require.Equal(t, []string{"parent"}, violatedFields(t, malformed))

	internal := connect.NewError(connect.CodeInternal, errors.New("store failure"))
	require.Same(t, internal, importDestinationError(internal))
}

func TestImportResourceAuthorization(t *testing.T) {
	for _, parent := range []string{"", "projects/payments"} {
		for _, entry := range []struct {
			request any
			method  string
		}{
			{&v1pb.DiscoverNeonBranchesRequest{Parent: new(parent)}, "/bytebase.v1.InstanceService/DiscoverNeonBranches"},
			{&v1pb.ImportInstancesRequest{Parent: new(parent), Imports: []*v1pb.InstanceImport{neonImportEntry("one", "foreign")}}, "/bytebase.v1.InstanceService/ImportInstances"},
		} {
			resources, err := getResourceFromRequest(context.Background(), entry.request, entry.method)
			require.NoError(t, err)
			require.Equal(t, []string{parent}, resources)
		}
	}
}

// Every other create caller shares this path, so the check is asserted where
// it is enforced and not only through the import.
func TestCreateInstanceRejectsMissingEnvironment(t *testing.T) {
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `INSERT INTO workspace(resource_id) VALUES ('default')`)
	require.NoError(t, err)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{Workspace: "default", Name: storepb.SettingName_ENVIRONMENT, Value: &storepb.EnvironmentSetting{Environments: []*storepb.EnvironmentSetting_Environment{{Id: "prod", Title: "Prod"}}}})
	require.NoError(t, err)
	service := &InstanceService{store: stores, profile: &config.Profile{}, licenseService: newInstanceServiceTestLicenseService(t, stores)}
	request := func(id, environment string) *connect.Request[v1pb.CreateInstanceRequest] {
		return connect.NewRequest(&v1pb.CreateInstanceRequest{InstanceId: id, Instance: &v1pb.Instance{
			Title:       id,
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new(environment),
			DataSources: []*v1pb.DataSource{{Id: "admin", Type: v1pb.DataSourceType_ADMIN, Host: "db.example.com"}},
		}})
	}

	_, err = service.createInstanceWithSync(ctx, request("missing-env", "environments/nope"), false)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	var missing *environmentNotFoundError
	require.ErrorAs(t, err, &missing)
	instance, err := stores.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: new("missing-env"), Workspace: "default"})
	require.NoError(t, err)
	require.Nil(t, instance)

	created, err := service.createInstanceWithSync(ctx, request("known-env", "environments/prod"), false)
	require.NoError(t, err)
	require.Equal(t, "environments/prod", created.Msg.GetEnvironment())

	unset, err := service.createInstanceWithSync(ctx, request("no-env", ""), false)
	require.NoError(t, err)
	require.Empty(t, unset.Msg.GetEnvironment())
}

func TestSampleQuotaFailureStatus(t *testing.T) {
	err := sampleProjectInstanceConnectError(errors.Join(errors.New("sample preparation failed"), &store.InstanceLimitError{Limit: 1}))
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
}
