package elasticsearch

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/esapi"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var instanceMetadata db.InstanceMetadata

	// version.
	version, err := d.getVerison()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch version from Elasticsearch server")
	}

	// indices.
	dbSchemaMetadata, err := d.SyncDBSchema(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch indices from Elasticsearch server")
	}

	instanceMetadata.Databases = append(instanceMetadata.Databases, dbSchemaMetadata)
	instanceMetadata.Version = version

	return &instanceMetadata, nil
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

func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.New("CheckSlowQuery() is not applicable to Elasticsearch")
}

func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.New("CheckSlowQueryLogEnabled() is not applicable to Elasticsearch")
}

// struct for getVersion().
type VersionResult struct {
	Version struct {
		Number string `json:"number"`
	} `json:"version"`
}

func (d *Driver) getVerison() (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s:%s", d.config.Host, d.config.Port), nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", d.config.AuthenticationPrivateKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read data from response")
	}
	if err = resp.Body.Close(); err != nil {
		return "", errors.Wrapf(err, "failed to close response's body")
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
	res, err := esapi.CatIndicesRequest{Format: "json", Pretty: true}.Do(context.Background(), d.client)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to list indices")
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read data from response")
	}
	if err = res.Body.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close response's body")
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
