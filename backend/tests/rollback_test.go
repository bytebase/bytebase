package tests

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	resourcemysql "github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestCreateRollbackIssueMySQL(t *testing.T) {
	t.Parallel()

	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer func() {
		_ = ctl.Close(ctx)
	}()

	// Create a MySQL instance.
	mysqlPort := getTestPort()
	stopInstance := resourcemysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer stopInstance()

	connCfg := getMySQLConnectionConfig(strconv.Itoa(mysqlPort), "")
	// Add MySQL instance to Bytebase.
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "mysqlInstance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: connCfg.Host, Port: connCfg.Port, Username: connCfg.Username}},
		},
	})
	a.NoError(err)
	t.Log("Instance added.")

	databaseName := t.Name()
	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, databaseName, "", nil)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)

	dbMySQL, err := connectTestMySQL(mysqlPort, "")
	a.NoError(err)
	_, err = dbMySQL.ExecContext(ctx, fmt.Sprintf(`
		USE %s;
		CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(20));
		INSERT INTO t VALUES (1, '1\n1'), (2, '2\n2'), (3, '3\n3')
	`, databaseName))
	a.NoError(err)
	t.Log("Schema initialized.")

	dmlSheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title: "migration statement sheet",
			Content: []byte(`
			DELETE FROM t WHERE id = 1;
			UPDATE t SET name = 'unknown\nunknown';
		`),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)

	// Run a DML issue.
	_, rollout, issue, err := ctl.changeDatabaseWithConfig(ctx, ctl.project, []*v1pb.Plan_Step{
		{
			Specs: []*v1pb.Plan_Spec{
				{
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Target:          database.Name,
							Sheet:           dmlSheet.Name,
							Type:            v1pb.Plan_ChangeDatabaseConfig_DATA,
							RollbackEnabled: true,
						},
					},
				},
			},
		},
	})
	a.NoError(err)

	// Check that the data is changed.
	type record struct {
		id   int
		name string
	}
	rows1, err := dbMySQL.QueryContext(ctx, "SELECT * FROM t;")
	a.NoError(err)
	defer rows1.Close()
	var records1 []record
	for rows1.Next() {
		var r record
		err = rows1.Scan(&r.id, &r.name)
		a.NoError(err)
		records1 = append(records1, r)
	}
	a.NoError(rows1.Err())
	want1 := []record{{2, "unknown\nunknown"}, {3, "unknown\nunknown"}}
	a.Equal(want1, records1)

	// wait for rollback SQL generation
	rollbackTaskName, rollbackSheet, err := waitRollbackStatement(ctx, ctl.rolloutServiceClient, rollout.Name)
	a.NoError(err)

	// Run a rollback issue.
	_, _, _, err = ctl.changeDatabaseWithConfig(ctx, ctl.project, []*v1pb.Plan_Step{
		{
			Specs: []*v1pb.Plan_Spec{
				{
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Target: database.Name,
							Sheet:  rollbackSheet,
							Type:   v1pb.Plan_ChangeDatabaseConfig_DATA,
							RollbackDetail: &v1pb.Plan_ChangeDatabaseConfig_RollbackDetail{
								RollbackFromTask:  rollbackTaskName,
								RollbackFromIssue: issue.Name,
							},
						},
					},
				},
			},
		},
	})
	a.NoError(err)

	// Check that the data is restored.
	rows2, err := dbMySQL.QueryContext(ctx, "SELECT * FROM t;")
	a.NoError(err)
	defer rows2.Close()
	var records2 []record
	for rows2.Next() {
		var r record
		err = rows2.Scan(&r.id, &r.name)
		a.NoError(err)
		records2 = append(records2, r)
	}
	a.NoError(rows2.Err())
	want2 := []record{{1, "1\n1"}, {2, "2\n2"}, {3, "3\n3"}}
	a.Equal(want2, records2)
}

func TestCreateRollbackIssueMySQLByPatch(t *testing.T) {
	t.Parallel()

	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer func() {
		_ = ctl.Close(ctx)
	}()

	// Create a MySQL instance.
	mysqlPort := getTestPort()
	stopInstance := resourcemysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer stopInstance()

	connCfg := getMySQLConnectionConfig(strconv.Itoa(mysqlPort), "")
	// Add MySQL instance to Bytebase.
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       t.Name(),
			Engine:      v1pb.Engine_MYSQL,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: connCfg.Host, Port: connCfg.Port, Username: connCfg.Username}},
		},
	})
	a.NoError(err)
	t.Log("Instance added.")

	databaseName := t.Name()
	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, databaseName, "", nil)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)

	dbMySQL, err := connectTestMySQL(mysqlPort, "")
	a.NoError(err)
	_, err = dbMySQL.ExecContext(ctx, fmt.Sprintf(`
		USE %s;
		CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(20));
		INSERT INTO t VALUES (1, '1\n1'), (2, '2\n2'), (3, '3\n3')
	`, databaseName))
	a.NoError(err)
	t.Log("Schema initialized.")

	dmlSheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title: "migration statement sheet",
			Content: []byte(`
			DELETE FROM t WHERE id = 1;
			UPDATE t SET name = 'unknown\nunknown';
		`),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)

	// Run a DML issue with rollbackEnabled set to false.
	plan, rollout, issue, err := ctl.changeDatabaseWithConfig(ctx, ctl.project, []*v1pb.Plan_Step{
		{
			Specs: []*v1pb.Plan_Spec{
				{
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Target: database.Name,
							Sheet:  dmlSheet.Name,
							Type:   v1pb.Plan_ChangeDatabaseConfig_DATA,
						},
					},
				},
			},
		},
	})
	a.NoError(err)

	// Patch rollbackEnabled to true.
	for _, step := range plan.Steps {
		for _, spec := range step.Specs {
			spec.GetChangeDatabaseConfig().RollbackEnabled = true
		}
	}
	_, err = ctl.rolloutServiceClient.UpdatePlan(ctx, &v1pb.UpdatePlanRequest{
		Plan:       plan,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"steps"}},
	})
	a.NoError(err)

	// Check that the data is changed.
	type record struct {
		id   int
		name string
	}
	rows1, err := dbMySQL.QueryContext(ctx, "SELECT * FROM t;")
	a.NoError(err)
	defer rows1.Close()
	var records1 []record
	for rows1.Next() {
		var r record
		err = rows1.Scan(&r.id, &r.name)
		a.NoError(err)
		records1 = append(records1, r)
	}
	a.NoError(rows1.Err())
	want1 := []record{{2, "unknown\nunknown"}, {3, "unknown\nunknown"}}
	a.Equal(want1, records1)

	// wait for rollback SQL generation
	rollbackTaskName, rollbackSheet, err := waitRollbackStatement(ctx, ctl.rolloutServiceClient, rollout.Name)
	a.NoError(err)

	// Run a rollback issue.
	_, _, _, err = ctl.changeDatabaseWithConfig(ctx, ctl.project, []*v1pb.Plan_Step{
		{
			Specs: []*v1pb.Plan_Spec{
				{
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Target: database.Name,
							Sheet:  rollbackSheet,
							Type:   v1pb.Plan_ChangeDatabaseConfig_DATA,
							RollbackDetail: &v1pb.Plan_ChangeDatabaseConfig_RollbackDetail{
								RollbackFromTask:  rollbackTaskName,
								RollbackFromIssue: issue.Name,
							},
						},
					},
				},
			},
		},
	})
	a.NoError(err)

	// Check that the data is restored.
	rows2, err := dbMySQL.QueryContext(ctx, "SELECT * FROM t;")
	a.NoError(err)
	defer rows2.Close()
	var records2 []record
	for rows2.Next() {
		var r record
		err = rows2.Scan(&r.id, &r.name)
		a.NoError(err)
		records2 = append(records2, r)
	}
	a.NoError(rows2.Err())
	want2 := []record{{1, "1\n1"}, {2, "2\n2"}, {3, "3\n3"}}
	a.Equal(want2, records2)
}

func TestRollbackCanceled(t *testing.T) {
	t.Parallel()

	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer func() {
		_ = ctl.Close(ctx)
	}()

	// Create a MySQL instance.
	mysqlPort := getTestPort()
	stopInstance := resourcemysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer stopInstance()

	connCfg := getMySQLConnectionConfig(strconv.Itoa(mysqlPort), "")
	// Add MySQL instance to Bytebase.
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       t.Name(),
			Engine:      v1pb.Engine_MYSQL,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: connCfg.Host, Port: connCfg.Port, Username: connCfg.Username}},
		},
	})
	a.NoError(err)
	t.Log("Instance added.")

	databaseName := t.Name()
	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, databaseName, "", nil)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)

	dbMySQL, err := connectTestMySQL(mysqlPort, "")
	a.NoError(err)
	_, err = dbMySQL.ExecContext(ctx, fmt.Sprintf(`
		USE %s;
		CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(20));
		INSERT INTO t VALUES (1, '1\n1'), (2, '2\n2'), (3, '3\n3')
	`, databaseName))
	a.NoError(err)
	t.Log("Schema initialized.")

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title: "delete statement sheet",
			Content: []byte(`
			DELETE FROM t WHERE id = 1;
			UPDATE t SET name = 'unknown\nunknown';
		`),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)

	// Run a DML issue.
	plan, rollout, _, err := ctl.changeDatabaseWithConfig(ctx, ctl.project, []*v1pb.Plan_Step{
		{
			Specs: []*v1pb.Plan_Spec{
				{
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Target:          database.Name,
							Sheet:           sheet.Name,
							Type:            v1pb.Plan_ChangeDatabaseConfig_DATA,
							RollbackEnabled: true,
						},
					},
				},
			},
		},
	})
	a.NoError(err)

	// Cancel rollback SQL generation.
	for _, step := range plan.Steps {
		for _, spec := range step.Specs {
			spec.GetChangeDatabaseConfig().RollbackEnabled = false
		}
	}
	_, err = ctl.rolloutServiceClient.UpdatePlan(ctx, &v1pb.UpdatePlanRequest{
		Plan:       plan,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"steps"}},
	})
	a.NoError(err)

	// Check that the data is changed.
	type record struct {
		id   int
		name string
	}
	rows1, err := dbMySQL.QueryContext(ctx, "SELECT * FROM t;")
	a.NoError(err)
	defer rows1.Close()
	var records1 []record
	for rows1.Next() {
		var r record
		err = rows1.Scan(&r.id, &r.name)
		a.NoError(err)
		records1 = append(records1, r)
	}
	a.NoError(rows1.Err())
	want1 := []record{{2, "unknown\nunknown"}, {3, "unknown\nunknown"}}
	a.Equal(want1, records1)

	rollout, err = ctl.rolloutServiceClient.GetRollout(ctx, &v1pb.GetRolloutRequest{Name: rollout.Name})
	a.NoError(err)
	a.Len(rollout.Stages, 1)
	a.Len(rollout.Stages[0].Tasks, 1)
	task := rollout.Stages[0].Tasks[0]
	a.Equal(false, task.GetDatabaseDataUpdate().RollbackEnabled)
}

func waitRollbackStatement(ctx context.Context, rolloutServiceClient v1pb.RolloutServiceClient, rolloutName string) (string, string, error) {
	for i := 0; i < 30; i++ {
		rollout, err := rolloutServiceClient.GetRollout(ctx, &v1pb.GetRolloutRequest{Name: rolloutName})
		if err != nil {
			return "", "", err
		}
		for _, stage := range rollout.Stages {
			for _, task := range stage.Tasks {
				dataUpdate := task.GetDatabaseDataUpdate()
				if dataUpdate.RollbackSqlStatus == v1pb.Task_DatabaseDataUpdate_DONE {
					return task.Name, dataUpdate.RollbackSheet, nil
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	return "", "", errors.Errorf("failed to generate rollback statement")
}
