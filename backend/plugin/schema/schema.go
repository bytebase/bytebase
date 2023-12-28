package schema

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	mux              sync.Mutex
	getDesignSchemas = make(map[storepb.Engine]getDesignSchema)
	parseToMetadatas = make(map[storepb.Engine]parseToMetadata)
)

type getDesignSchema func(string, *storepb.DatabaseSchemaMetadata) (string, error)
type parseToMetadata func(string) (*storepb.DatabaseSchemaMetadata, error)

func RegisterGetDesignSchema(engine storepb.Engine, f getDesignSchema) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getDesignSchemas[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getDesignSchemas[engine] = f
}

func GetDesignSchema(engine storepb.Engine, baselineSchema string, to *storepb.DatabaseSchemaMetadata) (string, error) {
	f, ok := getDesignSchemas[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(baselineSchema, to)
}

func RegisterParseToMetadatas(engine storepb.Engine, f parseToMetadata) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := parseToMetadatas[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	parseToMetadatas[engine] = f
}

func ParseToMetadata(engine storepb.Engine, schema string) (*storepb.DatabaseSchemaMetadata, error) {
	f, ok := parseToMetadatas[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	metadata, err := f(schema)
	if err != nil {
		return nil, err
	}

	for _, schema := range metadata.Schemas {
		for _, table := range schema.Tables {
			table.Classification, table.UserComment = common.GetClassificationAndUserComment(table.Comment)
			for _, col := range table.Columns {
				col.Classification, col.UserComment = common.GetClassificationAndUserComment(col.Comment)
			}
		}
	}
	return metadata, nil
}
