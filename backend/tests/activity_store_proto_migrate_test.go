package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/tests/fake"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

/*
 * Compare proto/generated-go/store/activity.pb.go and backend/legacyapi/activity.go to see what fields should be migrated.
 */

const (
	MigrateActivityIssueCommentCreatePayloadStatementFmt = `
UPDATE %s
SET payload = jsonb_build_object('issueCommentCreatePayload',
	CASE
	    WHEN payload ? 'externalApprovalEvent' THEN
			-- If the key 'externalApprovalEvent' exists updates its' value of key 'type' and 'action'.
			CASE
				WHEN payload #> '{externalApprovalEvent, action}' = '"APPROVE"' THEN
					jsonb_set(jsonb_set(payload, '{externalApprovalEvent, type}', '"TYPE_FEISHU"'), '{externalApprovalEvent, action}', '"ACTION_APPROVE"')
				ELSE
					jsonb_set(jsonb_set(payload, '{externalApprovalEvent, type}', '"TYPE_FEISHU"'), '{externalApprovalEvent, action}', '"ACTION_REJECT"')
			END
		WHEN payload ? 'taskRollbackBy' THEN
			-- Proto3 maps int64 to JSON string, so we need to convert them to string.
			jsonb_set(
				jsonb_set(
					jsonb_set(
						jsonb_set(
							payload, '{taskRollbackBy, issueId}', to_jsonb(payload #>> '{taskRollbackBy, issueId}')
						), '{taskRollbackBy, taskId}', to_jsonb(payload #>> '{taskRollbackBy, taskId}')
					), '{taskRollbackBy, rollbackByIssueId}', to_jsonb(payload #>> '{taskRollbackBy, rollbackByIssueId}')
				), '{taskRollbackBy, rollbackByTaskId}', to_jsonb(payload #>> '{taskRollbackBy, rollbackByTaskId}')
			)
		ELSE
			payload
	END
)
WHERE "type"='bb.issue.comment.create';
`
)

func TestMigrateActivityIssueCreatePayload(t *testing.T) {
	t.Skip("IssueCreatePayload is no need to migrate")
}

func TestMigrateActivityIssueCommentCreatePayload(t *testing.T) {
	testCases := []struct {
		legacyPayload *api.ActivityIssueCommentCreatePayload
		protoPayload  *storepb.ActivityPayload
	}{
		{
			legacyPayload: &api.ActivityIssueCommentCreatePayload{
				IssueName: "issue1",
			},
			protoPayload: &storepb.ActivityPayload{
				Payload: &storepb.ActivityPayload_IssueCommentCreatePayload{
					IssueCommentCreatePayload: &storepb.ActivityIssueCommentCreatePayload{
						IssueName: "issue1",
					},
				},
			},
		},
		{
			legacyPayload: &api.ActivityIssueCommentCreatePayload{
				IssueName: "issue1",
				ExternalApprovalEvent: &api.ExternalApprovalEvent{
					Type:      api.ExternalApprovalTypeFeishu,
					Action:    api.ExternalApprovalEventActionApprove,
					StageName: "stage1",
				},
			},
			protoPayload: &storepb.ActivityPayload{
				Payload: &storepb.ActivityPayload_IssueCommentCreatePayload{
					IssueCommentCreatePayload: &storepb.ActivityIssueCommentCreatePayload{
						IssueName: "issue1",
						Event: &storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_{
							ExternalApprovalEvent: &storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent{
								Type:      storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_TYPE_FEISHU,
								Action:    storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_ACTION_APPROVE,
								StageName: "stage1",
							},
						},
					},
				},
			},
		},
		{
			legacyPayload: &api.ActivityIssueCommentCreatePayload{
				IssueName: "issue1",
				ExternalApprovalEvent: &api.ExternalApprovalEvent{
					Type:      api.ExternalApprovalTypeFeishu,
					Action:    api.ExternalApprovalEventActionReject,
					StageName: "stage1",
				},
			},
			protoPayload: &storepb.ActivityPayload{
				Payload: &storepb.ActivityPayload_IssueCommentCreatePayload{
					IssueCommentCreatePayload: &storepb.ActivityIssueCommentCreatePayload{
						IssueName: "issue1",
						Event: &storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_{
							ExternalApprovalEvent: &storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent{
								Type:      storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_TYPE_FEISHU,
								Action:    storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_ACTION_REJECT,
								StageName: "stage1",
							},
						},
					},
				},
			},
		},
		{
			legacyPayload: &api.ActivityIssueCommentCreatePayload{
				IssueName: "issue1",
				TaskRollbackBy: &api.TaskRollbackBy{
					IssueID:           1,
					TaskID:            2,
					RollbackByIssueID: 3,
					RollbackByTaskID:  4,
				},
			},
			protoPayload: &storepb.ActivityPayload{
				Payload: &storepb.ActivityPayload_IssueCommentCreatePayload{
					IssueCommentCreatePayload: &storepb.ActivityIssueCommentCreatePayload{
						IssueName: "issue1",
						Event: &storepb.ActivityIssueCommentCreatePayload_TaskRollbackBy_{
							TaskRollbackBy: &storepb.ActivityIssueCommentCreatePayload_TaskRollbackBy{
								IssueId:           1,
								TaskId:            2,
								RollbackByIssueId: 3,
								RollbackByTaskId:  4,
							},
						},
					},
				},
			},
		},
	}

	a := require.New(t)
	log.SetLevel(zapcore.DebugLevel)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)
	metaDB, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	legacyTableName := fmt.Sprintf("legacy_%s", t.Name())
	_, err = metaDB.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE %s(payload JSONB, "type" TEXT NOT NULL DEFAULT 'bb.issue.comment.create');`, legacyTableName))
	a.NoError(err)
	protoTableName := fmt.Sprintf("proto_%s", t.Name())
	_, err = metaDB.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE %s(payload JSONB);`, protoTableName))
	a.NoError(err)

	for _, tc := range testCases {
		{
			// Apply migrate SQL to legacy table and unmarshal to compare with proto payload.
			payload, err := json.Marshal(tc.legacyPayload)
			a.NoError(err)
			_, err = metaDB.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s(payload) VALUES($1);", legacyTableName), payload)
			a.NoError(err)
			_, err = metaDB.ExecContext(ctx, fmt.Sprintf(MigrateActivityIssueCommentCreatePayloadStatementFmt, legacyTableName))
			a.NoError(err)
			var storePayload string
			err = metaDB.QueryRowContext(ctx, fmt.Sprintf("SELECT payload FROM %s;", legacyTableName)).Scan(&storePayload)
			a.NoError(err)
			var protoPayload storepb.ActivityPayload
			err = protojson.Unmarshal([]byte(storePayload), &protoPayload)
			a.NoError(err)
			a.True(proto.Equal(tc.protoPayload, &protoPayload))

			// Remove data
			_, err = metaDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s;", legacyTableName))
			a.NoError(err)
			_, err = metaDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s;", protoTableName))
			a.NoError(err)
		}

		{
			// Seed data to proto table and legacy table, and execute migrate SQL to legacy table and unmarshal to compare with proto payload.
			payload, err := json.Marshal(tc.legacyPayload)
			a.NoError(err)
			_, err = metaDB.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s(payload) VALUES($1);", legacyTableName), payload)
			a.NoError(err)
			payload, err = protojson.Marshal(tc.protoPayload)
			a.NoError(err)
			_, err = metaDB.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s(payload) VALUES($1);", protoTableName), payload)
			a.NoError(err)
			_, err = metaDB.ExecContext(ctx, fmt.Sprintf(MigrateActivityIssueCommentCreatePayloadStatementFmt, legacyTableName))
			a.NoError(err)
			var legacyPayload string
			err = metaDB.QueryRowContext(ctx, fmt.Sprintf("SELECT payload FROM %s;", legacyTableName)).Scan(&legacyPayload)
			a.NoError(err)
			var protoPayload string
			err = metaDB.QueryRowContext(ctx, fmt.Sprintf("SELECT payload FROM %s;", protoTableName)).Scan(&protoPayload)
			a.NoError(err)
			a.Equal(protoPayload, legacyPayload)

			// Remove data
			_, err = metaDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s;", legacyTableName))
			a.NoError(err)
			_, err = metaDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s;", protoTableName))
			a.NoError(err)
		}
	}
}
