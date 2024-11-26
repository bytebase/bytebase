package databricks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	schemaStmtFmt = "" +
		"--\n" +
		"-- %s structure for %s\n" +
		"--\n" +
		"%s;\n"
)

var (
	sysCatalog = "system"
	infoSchema = "information_schema"
	dftSchema  = "default"
)

func (d *Driver) Dump(ctx context.Context, writer io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	catalogMap, err := d.listCatologTables(ctx, "")
	if err != nil {
		return err
	}

	for catalogName, schemaMap := range catalogMap {
		if catalogName == sysCatalog {
			continue
		}
		if _, err := writer.Write([]byte(fmt.Sprintf("CREATE CATALOG %s\n", catalogName))); err != nil {
			return err
		}

		for schemaName, tableList := range schemaMap {
			if schemaName == infoSchema {
				continue
			}
			if schemaName != dftSchema {
				_, err := writer.Write([]byte(fmt.Sprintf("CREATE SCHEMA `%s`.`%s`\n", catalogName, schemaName)))
				if err != nil {
					return err
				}
			}

			viewDDL := strings.Builder{}
			mtViewDDL := strings.Builder{}
			extTblDDL := strings.Builder{}
			mngTblDDL := strings.Builder{}
			for _, tblUnion := range tableList {
				qualifiedName, err := getQualifiedTblName(catalogName, schemaName, tblUnion.name)
				if err != nil {
					return err
				}

				switch tblUnion.typeName {
				case catalog.TableTypeView:
					ddl, err := d.showCreateTable(ctx, qualifiedName)
					if err != nil {
						return err
					}
					formatDDL := fmt.Sprintf(schemaStmtFmt, "VIEW", qualifiedName, ddl)
					if _, err := viewDDL.WriteString(formatDDL); err != nil {
						return err
					}
				case catalog.TableTypeMaterializedView:
					tblAddInfo, err := d.descTbl(ctx, qualifiedName)
					if err != nil {
						return err
					}
					tblProperties, err := formatTblProperties(tblAddInfo)
					if err != nil {
						return err
					}
					formatDDL, err := genMaterializedViewDDL(qualifiedName, tblUnion.materialView, tblProperties)
					if err != nil {
						return err
					}
					if _, err := mtViewDDL.WriteString(formatDDL); err != nil {
						return err
					}
				case catalog.TableTypeExternal:
					ddl, err := d.showCreateTable(ctx, qualifiedName)
					if err != nil {
						return err
					}
					formatDDL := fmt.Sprintf(schemaStmtFmt, "EXTERNAL TABLE", qualifiedName, ddl)
					if _, err := extTblDDL.WriteString(formatDDL); err != nil {
						return err
					}
				case catalog.TableTypeManaged:
					ddl, err := d.showCreateTable(ctx, qualifiedName)
					if err != nil {
						return err
					}
					formatDDL := fmt.Sprintf(schemaStmtFmt, "MANAGED TABLE", qualifiedName, ddl)
					if _, err := mngTblDDL.WriteString(formatDDL); err != nil {
						return err
					}
				default:
					// we do not sync streaming table.
					continue
				}
			}
			if _, err := writer.Write([]byte(mngTblDDL.String())); err != nil {
				return err
			}
			if _, err := writer.Write([]byte(extTblDDL.String())); err != nil {
				return err
			}
			if _, err := writer.Write([]byte(viewDDL.String())); err != nil {
				return err
			}
			if _, err := writer.Write([]byte(mtViewDDL.String())); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *Driver) showCreateTable(ctx context.Context, qualifiedTblName string) (string, error) {
	rows, colInfo, err := d.execSingleSQLSync(ctx, fmt.Sprintf("SHOW CREATE TABLE %s", qualifiedTblName))
	if err != nil {
		return "", err
	}

	if len(rows) != 1 || len(colInfo) != 1 {
		return "", errors.New("invalid ddl format")
	}

	return rows[0][0], nil
}

func genMaterializedViewDDL(qualifiedName string, mtViewMeta *storepb.MaterializedViewMetadata, tblProperties string) (string, error) {
	builder := strings.Builder{}
	if _, err := builder.WriteString(""); err != nil {
		return "", err
	}

	if _, err := builder.WriteString(fmt.Sprintf("CREATE MATERIALIZED VIEW %s\n", qualifiedName)); err != nil {
		return "", err
	}

	if _, err := builder.WriteString(fmt.Sprintf("COMMENT %s\n", mtViewMeta.Comment)); err != nil {
		return "", err
	}

	if _, err := builder.WriteString(fmt.Sprintf("TBLPROPERTIES %s\n", tblProperties)); err != nil {
		return "", err
	}

	return builder.String(), nil
}

type tableAdditionalInfo struct {
	tblProps map[string]string
}

func (d *Driver) descTbl(ctx context.Context, tblName string) (*tableAdditionalInfo, error) {
	tblAddInfo := &tableAdditionalInfo{
		tblProps: make(map[string]string),
	}
	rows, _, err := d.execSingleSQLSync(ctx, fmt.Sprintf("DESC FORMATTED %s", tblName))
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		infoKey := row[0]
		infoVal := row[1]
		if infoKey == "Table Properties" {
			pattern := regexp.MustCompile("([a-zA-Z0-9-.]*)=([a-zA-Z0-9-.]*)")
			matches := pattern.FindAllStringSubmatch(infoVal, -1)
			if matches == nil {
				continue
			}
			for _, match := range matches {
				tblAddInfo.tblProps[match[1]] = match[2]
			}
		}
	}

	return tblAddInfo, nil
}

func formatTblProperties(tblAddInfo *tableAdditionalInfo) (string, error) {
	if tblAddInfo == nil || tblAddInfo.tblProps == nil {
		return "", errors.New("properties cannot be nil value")
	}

	builder := strings.Builder{}
	idx := 0
	for key, val := range tblAddInfo.tblProps {
		comma := ","
		if idx == len(tblAddInfo.tblProps)-1 {
			comma = ""
		}
		if _, err := builder.WriteString(fmt.Sprintf("%s = %s%s\n", key, val, comma)); err != nil {
			return "", err
		}
		idx++
	}

	return builder.String(), nil
}
