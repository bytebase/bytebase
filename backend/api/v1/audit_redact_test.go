package v1

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// Audited RPCs write their request and response payloads to audit_log, and to
// stdout when RuntimeEnableAuditLogStdout is set. Anything with
// bb.auditLogs.search/export, or read access to the log pipeline, can read
// them. Redaction therefore owes the row two things, and the first group below
// is exactly those two:
//
//   - the secret leaves. Proven exhaustively rather than by example:
//     TestAuditRedactionCoversEveryAnnotatedField seeds a sentinel into all 567
//     annotated field paths across 321 request and response types and asserts
//     none survives. Hand-written per-credential tables used to sit here and
//     were removed — every field they covered is in that sweep, on the same
//     generated Go types, and a table is a list someone has to remember to
//     extend.
//
//   - the substance stays. TestAuditRowKeepsItsSubstance is the only guard for
//     this, and has to be written by hand: the sweep asserts absence, so a
//     redactor that blanked the entire row passes it. Over-redaction is the
//     failure mode nothing else in this package can see.
//
// The groups after that are the walker's own invariants — it terminates on the
// cyclic components, reaches an annotation at any depth, rebuilds rather than
// aliases, and never touches the caller's message — then the two payloads a
// descriptor walk cannot see into on its own, a packed Any and the streaming
// path. Fixtures and machinery are last, so the file reads as claims first.

const secretSentinel = "s3cr3t-sentinel-value"

var encodedSecretSentinel = base64.StdEncoding.EncodeToString([]byte(secretSentinel))

// ---- The contract: the secret leaves, the substance stays ----------------

// TestAuditRedactionCoversEveryAnnotatedField is the sweep. A redactor that
// stops covering a field fails here rather than silently starting to log it.
func TestAuditRedactionCoversEveryAnnotatedField(t *testing.T) {
	population := auditPopulation(t)
	names := make([]protoreflect.FullName, 0, len(population))
	for name := range population {
		names = append(names, name)
	}
	slices.SortFunc(names, func(a, b protoreflect.FullName) int { return strings.Compare(string(a), string(b)) })

	covered := 0
	for _, name := range names {
		descriptor := messageDescriptorByName(t, name)
		targets := annotatedTargets(descriptor, string(descriptor.Name()), nil, map[protoreflect.FullName]bool{})
		if len(targets) == 0 {
			continue
		}
		t.Run(string(name), func(t *testing.T) {
			for _, target := range targets {
				covered++
				filled := dynamicMessage(t, descriptor)
				require.True(t, seedTarget(filled.ProtoReflect(), target.chain),
					"%s could not be populated, so every assertion below would pass vacuously", target.path)

				redacted := redactForAudit(filled)
				assertNoAnnotatedContent(t, redacted.ProtoReflect(), string(descriptor.Name()))

				got, err := protojson.Marshal(redacted)
				require.NoError(t, err, target.path)
				require.NotContains(t, string(got), secretSentinel,
					"%s reached the audit payload of %v", target.path, population[name])
				require.NotContains(t, string(got), encodedSecretSentinel,
					"%s reached the audit payload of %v, base64-encoded", target.path, population[name])
			}
		})
	}
	require.NotZero(t, covered, "the sweep found nothing to assert on")
	t.Logf("swept %d annotated field paths across %d request/response types", covered, len(names))
}

// The over-redaction guard, and the half TestAuditRedactionCoversEveryAnnotatedField
// structurally cannot be: that sweep proves annotated fields are gone, and has
// no notion of what the row must still say. Every value below carries a
// sentinel, so the loop asserts both halves — the secret leaves, the substance
// stays.
//
// The rows naming a role, a user and a result count are the leaks this design
// was written for. The sweep already covers their credentials by field path;
// what only lives here is that redacting them did not take the surrounding
// record with it.
func TestAuditRowKeepsItsSubstance(t *testing.T) {
	batchParent := "projects/project-a"
	otpCode := secretSentinel
	mfaTempToken := secretSentinel
	queryResult := func() *v1pb.QueryResult {
		return &v1pb.QueryResult{
			RowsCount:   7,
			ColumnNames: []string{"card_number"},
			Rows:        []*v1pb.QueryRow{{Values: []*v1pb.RowValue{{Kind: &v1pb.RowValue_StringValue{StringValue: secretSentinel}}}}},
		}
	}
	for _, tt := range []struct {
		name     string
		value    any
		response bool
		want     []string
	}{
		{
			name:  "webhook request retains identifiers",
			value: &v1pb.AddWebhookRequest{Project: "projects/project-a", Webhook: &v1pb.Webhook{Name: "projects/project-a/webhooks/webhook-a", Title: "Webhook A", Url: secretSentinel}},
			want:  []string{"projects/project-a", "projects/project-a/webhooks/webhook-a", "Webhook A"},
		},
		{
			// The row has to stay useful: which instances were retargeted, and
			// which kind of IAM credential was supplied.
			name: "batch instance update retains targets and credential type",
			value: &v1pb.BatchUpdateInstancesRequest{
				Parent: &batchParent,
				Requests: []*v1pb.UpdateInstanceRequest{{Instance: &v1pb.Instance{
					Name:        "instances/instance-a",
					DataSources: []*v1pb.DataSource{dataSourceWithEverySecret()},
				}}},
			},
			want: []string{"projects/project-a", "instances/instance-a", "db.example.com", "gcpCredential"},
		},
		{
			name: "azure iam credential retains the principal it names",
			value: &v1pb.AddDataSourceRequest{DataSource: &v1pb.DataSource{
				IamExtension: &v1pb.DataSource_AzureCredential_{AzureCredential: &v1pb.DataSource_AzureCredential{
					TenantId: "tenant-a", ClientId: "client-a", ClientSecret: secretSentinel,
				}},
			}},
			want: []string{"azureCredential", "tenant-a", "client-a"},
		},
		{
			name: "aws iam credential records that one was supplied",
			value: &v1pb.AddDataSourceRequest{DataSource: &v1pb.DataSource{
				IamExtension: &v1pb.DataSource_AwsCredential{AwsCredential: &v1pb.DataSource_AWSCredential{
					AccessKeyId: secretSentinel, SecretAccessKey: secretSentinel,
				}},
			}},
			want: []string{"awsCredential"},
		},
		{
			name:     "audit export retains page token",
			value:    &v1pb.ExportAuditLogsResponse{Content: []byte(secretSentinel), NextPageToken: "next-page-token"},
			response: true,
			want:     []string{"next-page-token"},
		},
		{
			// INPUT_ONLY on the password, OUTPUT_ONLY only on its container, so
			// clients are meant to send it. Which role was written is the point
			// of the row; redactInstance used to pass the password with it.
			name:  "instance role update retains the role it names",
			value: &v1pb.UpdateInstanceRequest{Instance: instanceWithRolePassword()},
			want:  []string{"instances/instance-a", "role-a"},
		},
		{
			// A failed validation neither consumes the code nor suppresses the
			// row, so an intact OTP here used to be a still-live one.
			name:  "enable MFA retains who enrolled but not the code",
			value: &v1pb.EnableMFARequest{Name: "users/user@example.com", OtpCode: otpCode},
			want:  []string{"users/user@example.com"},
		},
		{
			name:  "change password retains who changed it but not the password",
			value: &v1pb.ChangePasswordRequest{Name: "users/user@example.com", NewPassword: secretSentinel},
			want:  []string{"users/user@example.com"},
		},
		{
			name:  "user update retains who was updated",
			value: &v1pb.UpdateUserRequest{User: &v1pb.User{Name: "users/user@example.com"}},
			want:  []string{"users/user@example.com"},
		},
		{
			// The two hand-written redactors disagreed here: Query kept
			// rows_count and AdminExecute dropped it. One annotation on `rows`
			// settles it, and the count is what makes the row worth reading.
			name:     "admin execute retains the shape of the result",
			value:    &v1pb.AdminExecuteResponse{Results: []*v1pb.QueryResult{queryResult()}},
			response: true,
			want:     []string{`"rowsCount":"7"`, "card_number"},
		},
		{
			name:     "query retains the shape of the result",
			value:    &v1pb.QueryResponse{Results: []*v1pb.QueryResult{queryResult()}},
			response: true,
			want:     []string{`"rowsCount":"7"`, "card_number"},
		},
		{
			// Who tried to log in is the whole point of a login row, and the
			// password, OTP and MFA token beside it are all annotated.
			name: "login retains who attempted it",
			value: &v1pb.LoginRequest{
				Email:        "alice@example.com",
				Password:     secretSentinel,
				OtpCode:      &otpCode,
				MfaTempToken: &mfaTempToken,
			},
			want: []string{"alice@example.com"},
		},
		{
			name:     "login response retains the user it authenticated",
			value:    &v1pb.LoginResponse{Token: secretSentinel, User: &v1pb.User{Name: "users/alice@example.com"}},
			response: true,
			want:     []string{"users/alice@example.com"},
		},
		{
			// The Signup password leak Codex flagged on #20024, which surfaced
			// only once SetAuditWorkspaceID re-enabled this audit path.
			name:  "signup retains the account being created",
			value: &v1pb.SignupRequest{Email: "bob@example.com", Password: secretSentinel, Title: "bob"},
			want:  []string{"bob@example.com", "bob"},
		},
		{
			// The other half of #20024. An external OIDC token can be replayed
			// against the issuing IdP, so it must not reach a row that
			// bb.auditLogs.search can read — but which workload presented it is
			// exactly what the row is for. ExchangeTokenResponse.access_token
			// leaves nothing behind to assert on, so the sweep covers it alone.
			name:  "token exchange retains the workload it correlates",
			value: &v1pb.ExchangeTokenRequest{Token: secretSentinel, Email: "ci-bot@workload.bytebase.com"},
			want:  []string{"ci-bot@workload.bytebase.com"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := marshalAuditPayload(tt.value)
			for _, want := range tt.want {
				require.Contains(t, got, want)
			}
			require.NotContains(t, got, secretSentinel, "a credential reached the audit log")
			require.NotContains(t, got, encodedSecretSentinel, "an encoded credential reached the audit log")
		})
	}
}

// ---- The walker's invariants ---------------------------------------------

// Redaction must not mutate the caller's message: it runs on the live request
// and response objects, so masking in place would corrupt what the handler
// returns to the client or what a later interceptor reads.
//
// Asserted as "the message is unchanged afterwards" rather than by reading back
// the one field a case cares about. A per-field check passes while redaction
// quietly clears the neighbour nobody thought to read, and the earlier version
// of this spent forty lines of type switch picking that field out.
//
// The last three rows are the repeated-message half, and the leak this design
// opens with: Instance.roles[].password and instance_resource.data_sources[]
// both cross a repeated message field, where copy-and-share is easiest to get
// wrong.
func TestAuditRedactionDoesNotMutateTheCallersMessage(t *testing.T) {
	for name, message := range map[string]proto.Message{
		"service account response":   &v1pb.ServiceAccount{ServiceKey: secretSentinel},
		"project webhook response":   &v1pb.Project{Webhooks: []*v1pb.Webhook{{Url: secretSentinel}}},
		"release response":           &v1pb.Release{Files: []*v1pb.Release_File{{Statement: []byte(secretSentinel)}}},
		"saved query response":       &v1pb.SavedQuery{Content: []byte(secretSentinel)},
		"audit export response":      &v1pb.ExportAuditLogsResponse{Content: []byte(secretSentinel), NextPageToken: "next-page-token"},
		"ai setting request":         &v1pb.UpdateSettingRequest{Setting: &v1pb.Setting{Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_Ai{Ai: &v1pb.AISetting{ApiKey: secretSentinel}}}}},
		"add data source request":    &v1pb.AddDataSourceRequest{DataSource: dataSourceWithEverySecret()},
		"update data source request": &v1pb.UpdateDataSourceRequest{DataSource: dataSourceWithEverySecret()},
		"remove data source request": &v1pb.RemoveDataSourceRequest{DataSource: dataSourceWithEverySecret()},
		"create instance request":    &v1pb.CreateInstanceRequest{Instance: &v1pb.Instance{DataSources: []*v1pb.DataSource{dataSourceWithEverySecret()}}},
		"update instance request":    &v1pb.UpdateInstanceRequest{Instance: &v1pb.Instance{DataSources: []*v1pb.DataSource{dataSourceWithEverySecret()}}},
		"create release request":     &v1pb.CreateReleaseRequest{Release: &v1pb.Release{Files: []*v1pb.Release_File{{Statement: []byte(secretSentinel)}}}},
		"update release request":     &v1pb.UpdateReleaseRequest{Release: &v1pb.Release{Files: []*v1pb.Release_File{{Statement: []byte(secretSentinel)}}}},
		"create saved query request": &v1pb.CreateSavedQueryRequest{SavedQuery: &v1pb.SavedQuery{Content: []byte(secretSentinel)}},
		"reset password request":     &v1pb.ResetPasswordRequest{Email: "user@example.com", Code: secretSentinel, NewPassword: secretSentinel},
		"instance roles":             &v1pb.Instance{Roles: []*v1pb.InstanceRole{{Name: "r", Password: proto.String(secretSentinel)}}},
		"database instance resource": &v1pb.Database{InstanceResource: &v1pb.InstanceResource{DataSources: []*v1pb.DataSource{{Id: "admin", Password: secretSentinel}}}},
		"batch update instances": &v1pb.BatchUpdateInstancesRequest{Requests: []*v1pb.UpdateInstanceRequest{{
			Instance: &v1pb.Instance{DataSources: []*v1pb.DataSource{dataSourceWithEverySecret()}},
		}}},
	} {
		t.Run(name, func(t *testing.T) {
			before := proto.Clone(message)
			got := marshalAuditPayload(message)

			require.NotContains(t, got, secretSentinel, "a credential reached the audit log")
			require.NotContains(t, got, encodedSecretSentinel, "an encoded credential reached the audit log")
			require.True(t, proto.Equal(before, message),
				"redaction mutated the caller's message\nbefore: %v\nafter:  %v", before, message)
		})
	}
}

// TestAuditPlanHandlesCyclicDescriptors pins the fixed-point builder against
// the four cyclic components in the descriptor graph. An unguarded descend runs
// until the stack does, inside the interceptor, taking the RPC with it — and a
// guard keyed on "the field's type equals the parent's" catches only the
// self-recursive one.
//
// Termination is the easy half. The harder one is that breaking a cycle by
// provisionally answering "no plan" leaves the resulting plan with no descend
// arm for the fields that resolved that way, so an annotation inside the cycle
// would be applied at depth 0 and nowhere else — with the coverage sweep still
// passing, because the top-level copy is clean.
func TestAuditPlanHandlesCyclicDescriptors(t *testing.T) {
	for name, root := range map[string]proto.Message{
		// TablePartitionMetadata.subpartitions, the only self-recursive field,
		// and the one component not on an audited path.
		"self-recursive partitions": &v1pb.DiffMetadataRequest{},
		// ObjectSchema <-> StructKind <-> ArrayKind, whose cycle runs through a
		// map. Reached from UpdateDatabaseCatalog, which is audited.
		"object schema through a map": &v1pb.UpdateDatabaseCatalogRequest{},
		// google.protobuf.Value <-> Struct <-> ListValue, reached from
		// QueryResponse.results[].rows[].values[].value_value on every audited
		// Query and AdminExecute.
		"protobuf Value": &v1pb.QueryResponse{},
		// google.api.expr Expr and its nested types, reached from
		// SetIamPolicyRequest.policy.bindings[].parsed_expr.
		"CEL expression": &v1pb.SetIamPolicyRequest{},
	} {
		t.Run(name, func(t *testing.T) {
			done := make(chan *redactPlan, 1)
			go func() { done <- planFor(root.ProtoReflect().Descriptor()) }()
			select {
			case plan := <-done:
				// The plan may legitimately be nil; what matters is that
				// building it terminated rather than recursing forever.
				_ = plan
			case <-time.After(30 * time.Second):
				t.Fatal("plan construction did not terminate on a cyclic descriptor")
			}
			// A second call has to answer from the cache, not rebuild.
			require.NotPanics(t, func() { planFor(root.ProtoReflect().Descriptor()) })
		})
	}
}

// TestAuditRedactionFollowsACycleToAnyDepth is the half termination alone does
// not give: that an annotation INSIDE a cycle is still reached. Breaking a
// cycle by provisionally answering "no plan" leaves the plan with no descend
// arm for the fields that resolved that way, so the annotation is applied at
// depth 0 and nowhere else — with the coverage sweep still green, because the
// top-level copy is clean.
//
// An earlier build did something subtler: it cached plan VALUES in the descend
// arm, freezing a finite chain, so everything below its end was shared
// unredacted. A 10-deep chain leaked 8 of its 11 secrets while a one-level
// fixture stayed green — which is why the depth here is far past three rather
// than the single hop that would read as sufficient. Descend arms hold a
// DESCRIPTOR now and resolve their plan at each depth.
//
// The cycle is built rather than found in the surface, because no annotation
// sits inside one today — exactly why either bug would go unnoticed until one
// did.
func TestAuditRedactionFollowsACycleToAnyDepth(t *testing.T) {
	descriptor := cyclicSensitiveDescriptor(t)
	secret := descriptor.Fields().ByName("secret")
	child := descriptor.Fields().ByName("child")

	plan := planFor(descriptor)
	require.NotNil(t, plan, "a message with an annotated field needs a plan")
	require.Equal(t, actionDescend, plan.fields[child.Number()].kind,
		"the recursive field has to carry a descend arm, or the annotation is applied at depth 0 and nowhere else")

	// Far deeper than any chain a value-caching builder could freeze, which was
	// three links.
	const depth = 12
	root := dynamicpb.NewMessage(descriptor)
	level := root
	for range depth {
		level.Set(secret, protoreflect.ValueOfString(secretSentinel))
		next := dynamicpb.NewMessage(descriptor)
		level.Set(child, protoreflect.ValueOfMessage(next))
		level = next
	}
	level.Set(secret, protoreflect.ValueOfString(secretSentinel))

	got, err := protojson.Marshal(redactForAudit(proto.Message(root)))
	require.NoError(t, err)
	require.NotContains(t, string(got), secretSentinel,
		"an annotated field deep inside a cycle survived redaction")
	require.Equal(t, depth, strings.Count(string(got), `"child"`),
		"the fixture has to actually reach that depth for the assertion above to mean anything")
	require.Equal(t, secretSentinel, root.Get(secret).String(), "redaction mutated the caller's message")
}

// TestAuditRedactionRebuildsMapsRatherThanSharing pins the map rule. Setting a
// map field from the value Range yields shares the Go map itself and its
// message values, so clearing a field in an entry writes through to the
// caller's live message — measured, the source loses the field.
//
// The fixture is built rather than found: no map in the v1 surface has a value
// type with a plan today, so this violation of "redaction does not mutate the
// caller's message" would otherwise go uncaught until one appeared.
func TestAuditRedactionRebuildsMapsRatherThanSharing(t *testing.T) {
	descriptor := mapOfSensitiveDescriptor(t)
	entryValue := dynamicpb.NewMessage(descriptor.Fields().ByName("entries").MapValue().Message())
	entryValue.Set(entryValue.Descriptor().Fields().ByName("secret"), protoreflect.ValueOfString(secretSentinel))

	message := dynamicpb.NewMessage(descriptor)
	entries := message.Mutable(descriptor.Fields().ByName("entries")).Map()
	entries.Set(protoreflect.ValueOfString("k").MapKey(), protoreflect.ValueOfMessage(entryValue))

	plan := planFor(descriptor)
	require.NotNil(t, plan)
	redacted := redactMessage(message, plan)

	got := redacted.Get(descriptor.Fields().ByName("entries")).Map().
		Get(protoreflect.ValueOfString("k").MapKey()).Message()
	require.Empty(t, got.Get(got.Descriptor().Fields().ByName("secret")).String(),
		"a credential inside a map value survived redaction")
	require.Equal(t, secretSentinel, entryValue.Get(entryValue.Descriptor().Fields().ByName("secret")).String(),
		"redaction wrote through the shared map and cleared the caller's own message")
}

// TestAuditRedactionKeepsSensitiveOneofArmsPresent pins the blank-versus-clear
// rule. Clearing a scalar arm unsets the oneof and erases which arm was
// supplied: DataSourceExternalSecret.token is a string arm whose sibling
// app_role is a message that would survive as {}, so clearing would make token
// auth indistinguishable from unconfigured while AppRole stayed legible.
func TestAuditRedactionKeepsSensitiveOneofArmsPresent(t *testing.T) {
	request := &v1pb.AddDataSourceRequest{DataSource: &v1pb.DataSource{
		ExternalSecret: &v1pb.DataSourceExternalSecret{
			Url:        "https://vault.example.com",
			AuthOption: &v1pb.DataSourceExternalSecret_Token{Token: secretSentinel},
		},
	}}
	got := marshalAuditPayload(request)
	require.NotContains(t, got, secretSentinel)
	require.Contains(t, got, `"token":""`,
		"the arm must stay present with no value, so the row still records that token auth was configured")
	require.Contains(t, got, "vault.example.com", "the secret store the caller named is the point of the row")

	// The message arm keeps its own shape, minus the credentials inside it.
	appRole := &v1pb.AddDataSourceRequest{DataSource: &v1pb.DataSource{
		ExternalSecret: &v1pb.DataSourceExternalSecret{
			AuthOption: &v1pb.DataSourceExternalSecret_AppRole{
				AppRole: &v1pb.DataSourceExternalSecret_AppRoleAuthOption{
					RoleId: secretSentinel, SecretId: secretSentinel, MountPath: "approle",
				},
			},
		},
	}}
	got = marshalAuditPayload(appRole)
	require.NotContains(t, got, secretSentinel)
	require.Contains(t, got, "approle", "which auth method was configured stays readable")
}

// TestAuditRedactionSharesUnannotatedSubtrees pins the performance property the
// whole design rests on: a subtree holding nothing annotated is shared by
// pointer, never copied. A 5 MB sheet inside an audited batch used to be cloned
// only to have its content nulled.
func TestAuditRedactionSharesUnannotatedSubtrees(t *testing.T) {
	labels := map[string]string{"env": "prod"}
	instance := &v1pb.Instance{
		Name:        "instances/instance-a",
		Labels:      labels,
		Roles:       []*v1pb.InstanceRole{{Name: "instances/instance-a/roles/r", Password: proto.String(secretSentinel)}},
		DataSources: []*v1pb.DataSource{{Id: "admin", Password: secretSentinel}},
	}
	redacted := redactForAudit(instance)
	require.NotSame(t, instance, redacted, "a message with a plan is copied, not returned as itself")
	require.Equal(t, labels, redacted.GetLabels(), "an unannotated map is shared, not rebuilt")

	// A message with nothing annotated anywhere beneath it is returned as
	// itself, which is what makes the sheet case free.
	sheet := &v1pb.CreateSheetRequest{Parent: "projects/project-a", Sheet: &v1pb.Sheet{Content: []byte(secretSentinel)}}
	require.NotSame(t, sheet, redactForAudit(sheet), "the sheet content is annotated, so the request is copied")
	plain := &v1pb.GetSheetRequest{Name: "projects/project-a/sheets/sheet-a"}
	require.Same(t, plain, redactForAudit(plain), "nothing under this message is annotated, so it is not copied")
}

// A rejected request is audited too, so redaction runs on messages the handler
// was about to refuse. A hand-written redactor used to read through the nil
// data source in one of these and panic the interceptor; the descriptor walk
// visits only populated fields, so there is nothing to read through — these
// pin that.
func TestAuditRedactionSurvivesAnIncompleteRequest(t *testing.T) {
	for _, tt := range []struct {
		name    string
		request any
	}{
		{"add data source without one", &v1pb.AddDataSourceRequest{Name: "instances/instance-a"}},
		{"update data source without one", &v1pb.UpdateDataSourceRequest{Name: "instances/instance-a"}},
		{"remove data source without one", &v1pb.RemoveDataSourceRequest{Name: "instances/instance-a"}},
		{"create instance without one", &v1pb.CreateInstanceRequest{InstanceId: "instance-a"}},
		{"update instance without one", &v1pb.UpdateInstanceRequest{}},
		{"batch update instances without any", &v1pb.BatchUpdateInstancesRequest{
			Requests: []*v1pb.UpdateInstanceRequest{{}},
		}},
		{"instance carrying an empty data source", &v1pb.UpdateInstanceRequest{Instance: &v1pb.Instance{
			DataSources: []*v1pb.DataSource{{}},
		}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := marshalAuditPayload(tt.request)
			require.NotEmpty(t, got, "a rejected request still gets a row")
		})
	}
}

// ---- The two payloads a descriptor walk cannot see on its own ------------

// TestAuditRedactsPackedAny pins the Any half. The descriptor walk sees only
// type_url and value, so a credential inside a packed message is invisible to
// it: the packed type is resolved and redacted at runtime, against the
// registry, or the Any is dropped.
func TestAuditRedactsPackedAny(t *testing.T) {
	setting, err := anypb.New(&v1pb.Setting{Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_Ai{
		Ai: &v1pb.AISetting{Endpoint: "https://ai.example.com", ApiKey: secretSentinel},
	}}})
	require.NoError(t, err)

	row := &storepb.AuditLog{
		Method:      "/bytebase.v1.SettingService/UpdateSetting",
		ServiceData: redactAuditServiceData(setting),
	}
	got, err := protojson.Marshal(row)
	require.NoError(t, err)
	require.NotContains(t, string(got), secretSentinel, "a credential inside the service_data before-image was recorded")
	require.Contains(t, string(got), "ai.example.com", "the rest of the before-image is the point of the row")
	require.Contains(t, string(setting.GetValue()), secretSentinel, "redaction mutated the caller's Any")

	t.Run("an unregistered packed type is dropped, not logged", func(t *testing.T) {
		// protojson.Marshal fails the ENTIRE row on an unresolvable Any, so
		// dropping is what keeps the record rather than losing it.
		stranger, err := anypb.New(&v1pb.Sheet{Content: []byte(secretSentinel)})
		require.NoError(t, err)
		row := &storepb.AuditLog{
			Method:      "/bytebase.v1.SheetService/CreateSheet",
			ServiceData: redactAuditServiceData(stranger),
		}
		require.Nil(t, row.GetServiceData())
		got, err := protojson.Marshal(row)
		require.NoError(t, err)
		require.NotContains(t, string(got), encodedSecretSentinel)
	})

	t.Run("a registered type with nothing to redact keeps its type_url verbatim", func(t *testing.T) {
		// connect's ErrorDetail.Type() returns a bare full name, so details are
		// stored without the type.googleapis.com/ prefix. A round-trip through
		// anypb.New would rewrite it and silently change what SearchAuditLogs
		// emits as @type.
		detail, err := connect.NewErrorDetail(&v1pb.PermissionDeniedDetail{Method: "/bytebase.v1.UserService/GetUser"})
		require.NoError(t, err)
		connectErr := connect.NewError(connect.CodePermissionDenied, errors.New("denied"))
		connectErr.AddDetail(detail)

		row := &storepb.AuditLog{Status: redactAuditStatus(convertErrToStatus(connectErr))}
		require.Len(t, row.GetStatus().GetDetails(), 1)
		require.Equal(t, "bytebase.v1.PermissionDeniedDetail", row.GetStatus().GetDetails()[0].GetTypeUrl())
	})
}

// TestStreamingAuditRedactsRows is the end-to-end assertion the streaming path
// needs. AdminExecute is the only streaming RPC, it is audited, and its
// response carries every row of an admin-mode query — and Send builds its own
// auditEntry and calls createAuditLog directly, so a redaction walk that lived
// in WrapUnary would silently skip it. Everything else about streaming
// persistence is exercised through the createAuditLogFunc stub, which bypasses
// the real path; this one writes a real row and reads it back.
func TestStreamingAuditRedactsRows(t *testing.T) {
	st := newAuditLiveStore(t)
	t.Cleanup(func() { require.NoError(t, st.Close()) })
	interceptor := NewAuditInterceptor(st, "test-secret", &config.Profile{})

	handler := interceptor.WrapStreamingHandler(func(_ context.Context, conn connect.StreamingHandlerConn) error {
		if err := conn.Receive(&v1pb.AdminExecuteRequest{
			Name:      "instances/instance-a/databases/db",
			Statement: "SELECT card_number FROM payments",
		}); err != nil {
			return err
		}
		return conn.Send(&v1pb.AdminExecuteResponse{Results: []*v1pb.QueryResult{{
			ColumnNames: []string{"card_number"},
			RowsCount:   1,
			Statement:   "SELECT card_number FROM payments",
			Rows: []*v1pb.QueryRow{{Values: []*v1pb.RowValue{
				{Kind: &v1pb.RowValue_StringValue{StringValue: secretSentinel}},
			}}},
		}}})
	})

	ctx := context.WithValue(context.Background(), common.AuthContextKey, &common.AuthContext{
		Audit:     true,
		Resources: []*common.Resource{{Type: common.ResourceTypeWorkspace, ID: auditTestWorkspace}},
	})
	require.NoError(t, handler(ctx, &auditStreamingConn{}))

	rows, err := st.SearchAuditLogs(ctx, &store.AuditLogFind{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NotContains(t, rows[0].Payload.Response, secretSentinel, "an admin-mode result row reached the audit log")
	require.Contains(t, rows[0].Payload.Response, "card_number", "the column names and the statement are the point of the row")
	require.Contains(t, rows[0].Payload.Response, `"rowsCount":"1"`,
		"how many rows the query returned survives; only the rows themselves are dropped")
}

// ---- Guards on the fixtures the sweep builds for itself ------------------

// TestNoAnnotationOutsideBytebaseProtos is what makes the fill's well-known-type
// skip safe. The skip exists because google.protobuf.Value rejects an empty
// instance at marshal time; it would be a hole if an annotation could ever sit
// inside one.
func TestNoAnnotationOutsideBytebaseProtos(t *testing.T) {
	var annotated []string
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if strings.HasPrefix(string(fd.Package()), "bytebase.") {
			return true
		}
		var walk func(protoreflect.MessageDescriptors)
		walk = func(messages protoreflect.MessageDescriptors) {
			for i := range messages.Len() {
				md := messages.Get(i)
				fields := md.Fields()
				for j := range fields.Len() {
					if field := fields.Get(j); auditBehaviorOf(field) != v1pb.AuditBehavior_AUDIT_BEHAVIOR_UNSPECIFIED {
						annotated = append(annotated, string(field.FullName()))
					}
				}
				walk(md.Messages())
			}
		}
		walk(fd.Messages())
		return true
	})
	require.Empty(t, annotated,
		"the coverage sweep skips non-bytebase messages when building fixtures; an annotation inside one would go untested")
}

// ---- Fixtures and machinery ----------------------------------------------

func instanceWithRolePassword() *v1pb.Instance {
	password := secretSentinel
	return &v1pb.Instance{
		Name:  "instances/instance-a",
		Roles: []*v1pb.InstanceRole{{Name: "instances/instance-a/roles/role-a", RoleName: "role-a", Password: &password}},
	}
}

// dataSourceWithEverySecret is the readable fixture: every credential a data
// source can carry, on one message, so a case reads as the request a caller
// sends. It is a hand-kept list and cannot see a field added later —
// TestAuditRedactionCoversEveryAnnotatedField is the net that can.
func dataSourceWithEverySecret() *v1pb.DataSource {
	return &v1pb.DataSource{
		Id:                                 "admin-ds",
		Host:                               "db.example.com",
		Username:                           "admin",
		Password:                           secretSentinel,
		SslCa:                              secretSentinel,
		SslCaPath:                          secretSentinel,
		SslCert:                            secretSentinel,
		SslCertPath:                        secretSentinel,
		SslKey:                             secretSentinel,
		SslKeyPath:                         secretSentinel,
		SshPassword:                        secretSentinel,
		SshPrivateKey:                      secretSentinel,
		AuthenticationPrivateKey:           secretSentinel,
		AuthenticationPrivateKeyPassphrase: secretSentinel,
		MasterPassword:                     secretSentinel,
		ExternalSecret: &v1pb.DataSourceExternalSecret{
			AuthOption: &v1pb.DataSourceExternalSecret_Token{Token: secretSentinel},
		},
		SaslConfig: &v1pb.SASLConfig{Mechanism: &v1pb.SASLConfig_KrbConfig{
			KrbConfig: &v1pb.KerberosConfig{Keytab: []byte(secretSentinel)},
		}},
		IamExtension: &v1pb.DataSource_GcpCredential{
			GcpCredential: &v1pb.DataSource_GCPCredential{Content: secretSentinel},
		},
	}
}

// auditStreamingConn is a minimal AdminExecute stream.
type auditStreamingConn struct {
	connect.StreamingHandlerConn
}

func (*auditStreamingConn) Spec() connect.Spec {
	return connect.Spec{Procedure: v1connect.SQLServiceAdminExecuteProcedure}
}
func (*auditStreamingConn) Peer() connect.Peer         { return connect.Peer{} }
func (*auditStreamingConn) RequestHeader() http.Header { return http.Header{} }
func (*auditStreamingConn) Receive(any) error          { return nil }
func (*auditStreamingConn) Send(any) error             { return nil }

// cyclicSensitiveDescriptor builds `message Node { string secret = 1
// [SENSITIVE]; Node child = 2; }` — a self-recursive message with an
// annotation inside the cycle.
func cyclicSensitiveDescriptor(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()
	return buildTestMessage(t, "audit_cycle.proto", "audittest.cycle", &descriptorpb.DescriptorProto{
		Name: proto.String("Node"),
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:    proto.String("secret"),
				Number:  proto.Int32(1),
				Label:   descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Options: sensitiveFieldOptions(),
			},
			{
				Name:     proto.String("child"),
				Number:   proto.Int32(2),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".audittest.cycle.Node"),
			},
		},
	})
}

// mapOfSensitiveDescriptor builds `message Holder { map<string, Entry> entries
// = 1; }` where Entry carries a SENSITIVE field.
func mapOfSensitiveDescriptor(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()
	return buildTestMessage(t, "audit_map.proto", "audittest.maps", &descriptorpb.DescriptorProto{
		Name: proto.String("Holder"),
		NestedType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("EntriesEntry"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("key"),
						Number: proto.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:     proto.String("value"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".audittest.maps.Entry"),
					},
				},
				Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
			},
		},
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:     proto.String("entries"),
				Number:   proto.Int32(1),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".audittest.maps.Holder.EntriesEntry"),
			},
		},
	}, &descriptorpb.DescriptorProto{
		Name: proto.String("Entry"),
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:    proto.String("secret"),
				Number:  proto.Int32(1),
				Label:   descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Options: sensitiveFieldOptions(),
			},
		},
	})
}

func sensitiveFieldOptions() *descriptorpb.FieldOptions {
	options := &descriptorpb.FieldOptions{}
	proto.SetExtension(options, v1pb.E_AuditBehavior, v1pb.AuditBehavior_SENSITIVE)
	return options
}

// buildTestMessage compiles a throwaway file descriptor and returns its first
// message. Building the fixture rather than borrowing one from the surface is
// the point: the shapes it covers — an annotation inside a cycle, and one
// inside a map value — do not exist in the v1 protos today, which is precisely
// why nothing would notice if the redactor stopped handling them.
func buildTestMessage(t *testing.T, path, pkg string, messages ...*descriptorpb.DescriptorProto) protoreflect.MessageDescriptor {
	t.Helper()
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:        proto.String(path),
		Package:     proto.String(pkg),
		Syntax:      proto.String("proto3"),
		MessageType: messages,
	}, nil)
	require.NoError(t, err)
	return file.Messages().Get(0)
}

// auditPopulation is every (procedure, direction) pair whose message can reach
// marshalAuditPayload, keyed by message type: 210 methods resolve to far fewer
// distinct types, and the redactor is per type.
func auditPopulation(t *testing.T) map[protoreflect.FullName][]string {
	t.Helper()
	byType := map[protoreflect.FullName][]string{}
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Package() != "bytebase.v1" {
			return true
		}
		services := fd.Services()
		for i := range services.Len() {
			sd := services.Get(i)
			methods := sd.Methods()
			for j := range methods.Len() {
				md := methods.Get(j)
				procedure := fmt.Sprintf("/%s/%s", sd.FullName(), md.Name())
				byType[md.Input().FullName()] = append(byType[md.Input().FullName()], procedure+" request")
				byType[md.Output().FullName()] = append(byType[md.Output().FullName()], procedure+" response")
			}
		}
		return true
	})
	require.NotEmpty(t, byType)
	return byType
}

// messageDescriptorByName resolves a name the population produced.
func messageDescriptorByName(t *testing.T, name protoreflect.FullName) protoreflect.MessageDescriptor {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
	require.NoError(t, err)
	md, ok := descriptor.(protoreflect.MessageDescriptor)
	require.True(t, ok, "%s is not a message", name)
	return md
}

// isWellKnown reports whether a message comes from outside our own protos.
// The fill skips these: google.protobuf.Value rejects an empty instance at
// marshal time, and building one just to prove it holds no credential is
// wasted work. TestNoAnnotationOutsideBytebaseProtos is what makes the skip
// safe rather than assumed.
func isWellKnown(descriptor protoreflect.MessageDescriptor) bool {
	return !strings.HasPrefix(string(descriptor.FullName()), "bytebase.")
}

// auditTarget is one annotated field and the chain of fields that reaches it
// from a top-level request or response message.
//
// One message per FIELD rather than one message with every field set: setting a
// second arm of a oneof clears the first, so a single-message sweep would
// exercise only the last-numbered arm while every other assertion passed
// vacuously. Driving off the target path also reaches a oneof nested inside
// another oneof's arm — every AppIMSetting payload arm sits under SettingValue's
// app_im arm.
type auditTarget struct {
	path  string
	chain []protoreflect.FieldDescriptor
}

// annotatedTargets enumerates every annotated field reachable from a
// descriptor, through repeated fields, map values and every arm of every oneof
// at once. Recursion into a type already on the path stops, which bounds the
// four cyclic components in the descriptor graph.
func annotatedTargets(descriptor protoreflect.MessageDescriptor, prefix string, chain []protoreflect.FieldDescriptor, onPath map[protoreflect.FullName]bool) []auditTarget {
	if onPath[descriptor.FullName()] || isWellKnown(descriptor) {
		return nil
	}
	onPath[descriptor.FullName()] = true
	defer delete(onPath, descriptor.FullName())

	var targets []auditTarget
	for i := range descriptor.Fields().Len() {
		field := descriptor.Fields().Get(i)
		path := fmt.Sprintf("%s.%s", prefix, field.Name())
		next := append(append([]protoreflect.FieldDescriptor{}, chain...), field)
		if auditBehaviorOf(field) != v1pb.AuditBehavior_AUDIT_BEHAVIOR_UNSPECIFIED {
			targets = append(targets, auditTarget{path: path, chain: next})
			continue
		}
		if sub := submessageOf(field); sub != nil {
			targets = append(targets, annotatedTargets(sub, path, next, onPath)...)
		}
	}
	return targets
}

// seedTarget populates exactly one annotated field, walking the chain and
// creating the containers on the way. It reports whether a sentinel value
// actually landed: a target it cannot seed is a target the assertions below
// would pass vacuously, so the caller fails instead of skipping.
func seedTarget(m protoreflect.Message, chain []protoreflect.FieldDescriptor) bool {
	field := chain[0]
	if len(chain) == 1 {
		return seedAnnotatedField(m, field)
	}
	switch {
	case field.IsMap():
		entry := m.Mutable(field).Map().NewValue()
		if !seedTarget(entry.Message(), chain[1:]) {
			return false
		}
		m.Mutable(field).Map().Set(protoreflect.ValueOfString("key").MapKey(), entry)
		return true
	case field.IsList():
		list := m.Mutable(field).List()
		element := list.NewElement()
		if !seedTarget(element.Message(), chain[1:]) {
			return false
		}
		list.Append(element)
		return true
	default:
		return seedTarget(m.Mutable(field).Message(), chain[1:])
	}
}

// seedAnnotatedField puts the sentinel into the annotated field itself. A
// scalar takes it directly; an annotated message or container takes it on the
// first scalar reachable inside, so an OMIT subtree like User.profile or
// QueryResult.rows is populated rather than merely present.
func seedAnnotatedField(m protoreflect.Message, field protoreflect.FieldDescriptor) bool {
	if sub := submessageOf(field); sub != nil {
		seeded := false
		switch {
		case field.IsMap():
			entry := m.Mutable(field).Map().NewValue()
			seeded = seedFirstScalar(entry.Message(), map[protoreflect.FullName]bool{})
			m.Mutable(field).Map().Set(protoreflect.ValueOfString("key").MapKey(), entry)
		case field.IsList():
			list := m.Mutable(field).List()
			element := list.NewElement()
			seeded = seedFirstScalar(element.Message(), map[protoreflect.FullName]bool{})
			list.Append(element)
		default:
			seeded = seedFirstScalar(m.Mutable(field).Message(), map[protoreflect.FullName]bool{})
		}
		return seeded
	}
	value, ok := sentinelValue(field)
	if !ok {
		return false
	}
	if field.IsList() {
		m.Mutable(field).List().Append(value)
		return true
	}
	m.Set(field, value)
	return true
}

// seedFirstScalar puts the sentinel on the first string or bytes field
// reachable inside a message, descending until it finds one.
func seedFirstScalar(m protoreflect.Message, onPath map[protoreflect.FullName]bool) bool {
	descriptor := m.Descriptor()
	if descriptor.FullName() == anyFullName {
		// An Any carries no scalar of its own worth seeding, but its type_url
		// is a string and is what an annotated Any field has to be shown
		// dropping. It never reaches protojson, because the annotation that
		// made it a target is what removes it.
		m.Set(descriptor.Fields().ByName("type_url"), protoreflect.ValueOfString(secretSentinel))
		return true
	}
	if onPath[descriptor.FullName()] || isWellKnown(descriptor) {
		return false
	}
	onPath[descriptor.FullName()] = true
	defer delete(onPath, descriptor.FullName())

	for i := range descriptor.Fields().Len() {
		field := descriptor.Fields().Get(i)
		if submessageOf(field) != nil {
			continue
		}
		if value, ok := sentinelValue(field); ok {
			if field.IsList() {
				m.Mutable(field).List().Append(value)
			} else {
				m.Set(field, value)
			}
			return true
		}
	}
	for i := range descriptor.Fields().Len() {
		field := descriptor.Fields().Get(i)
		sub := submessageOf(field)
		if sub == nil {
			continue
		}
		switch {
		case field.IsMap():
			entry := m.Mutable(field).Map().NewValue()
			if seedFirstScalar(entry.Message(), onPath) {
				m.Mutable(field).Map().Set(protoreflect.ValueOfString("key").MapKey(), entry)
				return true
			}
		case field.IsList():
			list := m.Mutable(field).List()
			element := list.NewElement()
			if seedFirstScalar(element.Message(), onPath) {
				list.Append(element)
				return true
			}
		default:
			if seedFirstScalar(m.Mutable(field).Message(), onPath) {
				return true
			}
			m.Clear(field)
		}
	}
	return false
}

func sentinelValue(field protoreflect.FieldDescriptor) (protoreflect.Value, bool) {
	switch field.Kind() {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(secretSentinel), true
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes([]byte(secretSentinel)), true
	default:
		return protoreflect.Value{}, false
	}
}

// assertNoAnnotatedContent walks every populated field of a redacted message and
// requires that annotated ones carry no content. It is the whole-annotation
// assertion the string check below cannot make: a field the fixture never
// populated passes a "does not contain" check for free.
//
// Blank rather than absent, because a SENSITIVE oneof arm stays present with an
// empty value — that is how the row still records which credential type the
// caller supplied.
func assertNoAnnotatedContent(t *testing.T, m protoreflect.Message, path string) {
	t.Helper()
	m.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		fieldPath := fmt.Sprintf("%s.%s", path, field.Name())
		if auditBehaviorOf(field) != v1pb.AuditBehavior_AUDIT_BEHAVIOR_UNSPECIFIED {
			require.True(t, isBlankValue(field, value), "annotated field %s survived redaction", fieldPath)
			return true
		}
		switch {
		case field.IsMap():
			if isMessageKind(field.MapValue().Kind()) {
				value.Map().Range(func(key protoreflect.MapKey, entry protoreflect.Value) bool {
					assertNoAnnotatedContent(t, entry.Message(), fmt.Sprintf("%s[%v]", fieldPath, key))
					return true
				})
			}
		case isMessageKind(field.Kind()) && field.IsList():
			for i := range value.List().Len() {
				assertNoAnnotatedContent(t, value.List().Get(i).Message(), fmt.Sprintf("%s[%d]", fieldPath, i))
			}
		case isMessageKind(field.Kind()):
			assertNoAnnotatedContent(t, value.Message(), fieldPath)
		default:
		}
		return true
	})
}

func isBlankValue(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
	switch {
	case field.IsMap():
		return value.Map().Len() == 0
	case field.IsList():
		return value.List().Len() == 0
	case field.Kind() == protoreflect.BytesKind:
		return len(value.Bytes()) == 0
	case field.Kind() == protoreflect.StringKind:
		return value.String() == ""
	case isMessageKind(field.Kind()):
		return !value.Message().IsValid() || proto.Size(value.Message().Interface()) == 0
	default:
		return false
	}
}

// dynamicMessage builds an empty instance of a descriptor through the global
// type registry, so the sweep needs no per-type Go literal.
func dynamicMessage(t *testing.T, descriptor protoreflect.MessageDescriptor) proto.Message {
	t.Helper()
	messageType, err := protoregistry.GlobalTypes.FindMessageByName(descriptor.FullName())
	require.NoError(t, err)
	return messageType.New().Interface()
}
