package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/alexmullins/zip"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestSQLExportMatchesQueryMasking checks that an export masks a column the
// same way a query does. MD5 masking is not idempotent, so masking a result
// twice changes the exported value.
func TestSQLExportMatchesQueryMasking(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctl, ctx := startWorkspace(context.Background(), t)

	_, err := ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_SEMANTIC_TYPES.String(),
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_SemanticType{
					SemanticType: &v1pb.SemanticTypeSetting{
						Types: []*v1pb.SemanticTypeSetting_SemanticType{{
							Id:    "hashed",
							Title: "Hashed",
							Algorithm: &v1pb.Algorithm{
								Mask: &v1pb.Algorithm_Md5Mask{Md5Mask: &v1pb.Algorithm_MD5Mask{Salt: "salt"}},
							},
						}},
					},
				},
			},
		},
	}))
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("export-masking"),
		Instance: &v1pb.Instance{
			SyncDatabases: &v1pb.SyncDatabases{},
			Title:         "Export masking",
			Engine:        v1pb.Engine_POSTGRES,
			Environment:   new("environments/prod"),
			Activation:    true,
			DataSources:   []*v1pb.DataSource{sharedPgTarget(t).adminDataSource()},
		},
	}))
	a.NoError(err)
	databaseName := generateRandomString("maskdb")
	a.NoError(ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, databaseName, ""))
	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	setup, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte("CREATE TABLE employee(id INT PRIMARY KEY, secret TEXT); INSERT INTO employee VALUES (1, 'plain');")},
	}))
	a.NoError(err)
	a.NoError(ctl.changeDatabase(ctx, ctl.project, database, setup.Msg, false))
	_, err = ctl.databaseCatalogServiceClient.UpdateDatabaseCatalog(ctx, connect.NewRequest(&v1pb.UpdateDatabaseCatalogRequest{
		Catalog: &v1pb.DatabaseCatalog{
			Name: database.Name + "/catalog",
			Schemas: []*v1pb.SchemaCatalog{{
				Name: "public",
				Tables: []*v1pb.TableCatalog{{
					Name: "employee",
					Kind: &v1pb.TableCatalog_Columns_{Columns: &v1pb.TableCatalog_Columns{
						Columns: []*v1pb.ColumnCatalog{{Name: "secret", SemanticType: "hashed"}},
					}},
				}},
			}},
		},
	}))
	a.NoError(err)

	const statement = "SELECT secret FROM employee"
	queryResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{Name: database.Name, Statement: statement}))
	a.NoError(err)
	a.Len(queryResp.Msg.Results, 1)
	a.Empty(queryResp.Msg.Results[0].Error)
	a.Len(queryResp.Msg.Results[0].Rows, 1)
	queried := queryResp.Msg.Results[0].Rows[0].Values[0].GetStringValue()
	a.NotEqual("plain", queried, "precondition: the semantic type must mask the column")

	exportResp, err := ctl.sqlServiceClient.Export(ctx, connect.NewRequest(&v1pb.ExportRequest{
		Name:      database.Name,
		Format:    v1pb.ExportFormat_CSV,
		Statement: statement,
	}))
	a.NoError(err)
	content := exportResp.Msg.Content
	zipReader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	a.NoError(err)
	var exported string
	for _, file := range zipReader.File {
		if !strings.HasSuffix(file.Name, ".result.csv") {
			continue
		}
		r, err := file.Open()
		a.NoError(err)
		data, err := io.ReadAll(r)
		a.NoError(err)
		exported = string(data)
	}
	// CSV export quotes text values.
	a.Equal("secret\n\""+queried+"\"", exported)
}
