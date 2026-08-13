package v1

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestValidateOnlyAuditSkipAppliesOnlyToSuccess pins the boundary of the
// validate-only skip in createAuditLog.
//
// The skip exists so a dry run — which changes nothing — does not spam the
// audit log. It said nothing about the outcome, so it also swallowed every
// FAILED attempt whose request carries validate_only. That handed an agent an
// unlogged attempt at any forbidden method taking one of those six requests:
// two identical denials of the same method, in the same session, differing only
// in that flag, produced one row between them.
//
// The three cases below are the whole rule: outcome decides, not the flag
// alone.
func TestValidateOnlyAuditSkipAppliesOnlyToSuccess(t *testing.T) {
	st := newAuditLiveStore(t)
	in := NewAuditInterceptor(st, "test-secret", &config.Profile{})

	// A retarget with the stored password left to ride along — the shape the
	// MCP class refuses. The secrets are here so the redaction assertion below
	// has something to catch.
	const storedPassword = "stored-db-secret"
	const storedKeytab = "stored-keytab-bytes"
	retargetRequest := func(validateOnly bool) *v1pb.UpdateDataSourceRequest {
		return &v1pb.UpdateDataSourceRequest{
			Name: "instances/probe",
			DataSource: &v1pb.DataSource{
				Id:       "admin-ds",
				Host:     "attacker.example.com",
				Password: storedPassword,
				SaslConfig: &v1pb.SASLConfig{
					Mechanism: &v1pb.SASLConfig_KrbConfig{
						KrbConfig: &v1pb.KerberosConfig{Keytab: []byte(storedKeytab)},
					},
				},
			},
			ValidateOnly: validateOnly,
		}
	}

	invoke := func(t *testing.T, correlationID string, request *v1pb.UpdateDataSourceRequest, rerr error) {
		t.Helper()
		authCtx := &common.AuthContext{
			Audit:          true,
			AuthMethod:     common.AuthMethodIAM,
			Resources:      []*common.Resource{{Type: common.ResourceTypeWorkspace, ID: auditTestWorkspace}},
			DelegatedGrant: &common.DelegatedGrant{CorrelationID: correlationID},
		}
		next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			if rerr != nil {
				return nil, rerr
			}
			return connect.NewResponse(&v1pb.Instance{Name: "instances/probe"}), nil
		}
		req := &specRequest{
			AnyRequest: connect.NewRequest(request),
			procedure:  "/bytebase.v1.InstanceService/UpdateDataSource",
		}
		_, err := in.WrapUnary(next)(newAuditTestContext(authCtx), req)
		if rerr == nil {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}

	t.Run("a denied validate-only call is recorded", func(t *testing.T) {
		denial := connect.NewError(connect.CodePermissionDenied,
			errors.New("InstanceService/UpdateDataSource is not available to MCP sessions"))
		invoke(t, "corr-validate-only-denied", retargetRequest(true), denial)

		rows := findRowsByCorrelation(t, st, "corr-validate-only-denied")
		require.Len(t, rows, 1,
			"a refused attempt must be auditable whether or not validate_only was set — "+
				"otherwise the flag is a switch that turns off the record")
		require.Equal(t, int32(connect.CodePermissionDenied), rows[0].Payload.GetStatus().GetCode(),
			"the row must carry the denial, not a blank status")

		// The newly-logged row goes through the same getRequestString
		// redaction as every other UpdateDataSource row — that is why
		// narrowing the skip exposes no field the plain path did not already
		// write. It pins that the path runs, not that redactDataSource covers
		// every secret: the iam_extension oneof and
		// authentication_private_key_passphrase are unmasked there today, on
		// this row and on every UpdateDataSource row that preceded it.
		request := rows[0].Payload.GetRequest()
		require.NotEmpty(t, request)
		require.NotContains(t, request, storedPassword,
			"the request payload must be redacted before it reaches the audit row")
		// The row is protojson, which renders a bytes field base64, so the
		// keytab has to be looked for in that form — searching for the ASCII
		// literal would pass even with keytab redaction deleted outright.
		require.NotContains(t, request, base64.StdEncoding.EncodeToString([]byte(storedKeytab)),
			"the keytab reaches the row base64-encoded, and must be masked before it does")
		require.Contains(t, request, "attacker.example.com",
			"the destination is what an operator reads the row for — it must survive redaction")
	})

	t.Run("a succeeding validate-only call stays silent", func(t *testing.T) {
		invoke(t, "corr-validate-only-ok", retargetRequest(true), nil)

		require.Empty(t, findRowsByCorrelation(t, st, "corr-validate-only-ok"),
			"a dry run that succeeded changed nothing — the original reason for the skip")
	})

	t.Run("a failed validate-only call is recorded, denial or not", func(t *testing.T) {
		// The rule is any failure, not any refusal, and this is the case that
		// pins it: a validate-only connection test that could not reach the
		// host. It is also the bulk of what the change adds, since the
		// instance form dials before every save.
		failedDial := connect.NewError(connect.CodeInvalidArgument,
			errors.New("failed to connect to attacker.example.com: connection refused"))
		invoke(t, "corr-validate-only-dial-failed", retargetRequest(true), failedDial)

		rows := findRowsByCorrelation(t, st, "corr-validate-only-dial-failed")
		require.Len(t, rows, 1,
			"keying on a denial code would drop every other rejected attempt — the hole this change closes")
		require.Equal(t, int32(connect.CodeInvalidArgument), rows[0].Payload.GetStatus().GetCode())
	})

	t.Run("a denied ordinary call is recorded", func(t *testing.T) {
		denial := connect.NewError(connect.CodePermissionDenied, errors.New("denied"))
		invoke(t, "corr-plain-denied", retargetRequest(false), denial)

		rows := findRowsByCorrelation(t, st, "corr-plain-denied")
		require.Len(t, rows, 1, "control: the flag is the only difference between this and the first case")
		require.True(t, strings.Contains(rows[0].Payload.GetMethod(), "UpdateDataSource"))
	})
}
