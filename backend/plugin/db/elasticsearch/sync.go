package elasticsearch

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"io"
	"net/http"

	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	// version.
	version, err := d.getVerison()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch version from Elasticsearch server")
	}

	// databases.
	dbMetadata, err := d.SyncDBSchema(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch indices from Elasticsearch server")
	}

	// roles.
	instanceRoles, err := d.getInstanceRoles()
	if err != nil {
		return nil, err
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: []*storepb.DatabaseSchemaMetadata{dbMetadata},
		Metadata: &storepb.Instance{
			Roles: instanceRoles,
		},
	}, nil
}

// SyncDBSchema implements db.Driver.
func (d *Driver) SyncDBSchema(_ context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	var dbSchemaMetadata storepb.DatabaseSchemaMetadata

	// indices.
	indices, err := d.getIndices()
	if err != nil {
		return nil, err
	}

	// TODO(tommy): database name?
	dbSchemaMetadata.Name = "node"
	dbSchemaMetadata.Schemas = append(dbSchemaMetadata.Schemas, &storepb.SchemaMetadata{Tables: indices})

	return &dbSchemaMetadata, nil
}

// struct for getVersion().
type VersionResult struct {
	Version struct {
		Number string `json:"number"`
	} `json:"version"`
}

func (d *Driver) getVerison() (string, error) {
	resp, err := d.basicAuthClient.Do("GET", []byte("/"), nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err := resp.Body.Close(); err != nil {
		return "", err
	}

	var result VersionResult
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return "", err
	}

	return result.Version.Number, nil
}

type IndicesResult struct {
	IndexSize string `json:"store.size"`
	DocsCount string `json:"docs.count"`
	Index     string `json:"index"`
}

func (d *Driver) getIndices() ([]*storepb.TableMetadata, error) {
	// GET _cat/indices.
	res, err := esapi.CatIndicesRequest{Format: "json", Pretty: true}.Do(context.Background(), d.typedClient)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to list indices")
	}

	bytes, err := readBytesAndClose(res)
	if err != nil {
		return nil, err
	}

	var results []IndicesResult
	var indicesMetadata []*storepb.TableMetadata
	if err := json.Unmarshal(bytes, &results); err != nil {
		return nil, err
	}

	for _, m := range results {
		// index size.
		datasize, err := unitConversion(m.IndexSize)
		if err != nil {
			return nil, err
		}

		// document count.
		docCount, err := strconv.Atoi(m.DocsCount)
		if err != nil {
			return nil, err
		}

		indicesMetadata = append(indicesMetadata, &storepb.TableMetadata{
			Name:     m.Index,
			DataSize: datasize,
			RowCount: int64(docCount),
		})
	}
	return indicesMetadata, nil
}

func unitConversion(sizeWithUnit string) (int64, error) {
	sizeWithUnit = strings.ToLower(sizeWithUnit)
	sizeRe := regexp.MustCompile("([0-9.]+)([gmk]?b)")
	match := sizeRe.FindSubmatch([]byte(sizeWithUnit))

	unit := string(match[2])

	size, err := strconv.ParseFloat(string(match[1]), 64)
	if err != nil {
		return 0, err
	}

	switch unit {
	case "kb":
		size *= 1024
	case "mb":
		size *= 1024 * 1024
	case "gb":
		size *= 1024 * 1024 * 1024
	}

	return int64(size), nil
}

func readBytesAndClose(anyResp any) ([]byte, error) {
	var body io.ReadCloser
	// get body closer.
	switch resp := anyResp.(type) {
	case *http.Response:
		body = resp.Body

	case *esapi.Response:
		body = resp.Body

	default:
		return nil, errors.New("not supported response type")
	}

	// read bytes.
	bytes, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	if err := body.Close(); err != nil {
		return nil, err
	}

	return bytes, nil
}
