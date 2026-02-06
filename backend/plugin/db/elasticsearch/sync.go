package elasticsearch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := d.getVersion()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch version from Elasticsearch server")
	}

	dbMetadata, err := d.SyncDBSchema(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch indices from Elasticsearch server")
	}

	instanceRoles, err := d.getInstanceRoles(ctx)
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
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	var dbMetadataProto storepb.DatabaseSchemaMetadata

	// indices.
	indices, err := d.getIndices(ctx)
	if err != nil {
		return nil, err
	}

	// TODO(tommy): database name?
	dbMetadataProto.Name = "node"
	dbMetadataProto.Schemas = append(dbMetadataProto.Schemas, &storepb.SchemaMetadata{Tables: indices})

	return &dbMetadataProto, nil
}

// struct for getVersion().
type VersionResult struct {
	Version struct {
		Number string `json:"number"`
	} `json:"version"`
}

func (d *Driver) getVersion() (string, error) {
	if d.isOpenSearch && d.opensearchAPI != nil {
		ctx := context.Background()
		info, err := d.opensearchAPI.Info(ctx, &opensearchapi.InfoReq{})
		if err != nil {
			return "", err
		}
		return info.Version.Number, nil
	}
	resp, err := d.basicAuthClient.Do("GET", []byte("/"), nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response body")
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		// Include response body for debugging
		return "", errors.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bytes))
	}

	var result VersionResult
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		// Include response body to help debug parsing issues
		bodyPreview, truncated := common.TruncateString(string(bytes), 500)
		if truncated {
			bodyPreview += "..."
		}
		return "", errors.Wrapf(err, "failed to parse version response: %s", bodyPreview)
	}

	return result.Version.Number, nil
}

type IndicesResult struct {
	IndexSize string `json:"store.size"`
	DocsCount string `json:"docs.count"`
	Index     string `json:"index"`
}

// IndexSettingsResult represents the settings for an index.
type IndexSettingsResult struct {
	Settings struct {
		Index struct {
			Hidden string `json:"hidden"`
		} `json:"index"`
	} `json:"settings"`
}

// getHiddenIndices returns a set of index names that are marked as hidden.
func (d *Driver) getHiddenIndices(ctx context.Context) (map[string]bool, error) {
	hiddenIndices := make(map[string]bool)

	if d.isOpenSearch && d.opensearchAPI != nil {
		resp, err := d.opensearchAPI.Indices.Settings.Get(ctx, &opensearchapi.SettingsGetReq{
			Indices:  []string{"*"},
			Settings: []string{"index.hidden"},
			Params: opensearchapi.SettingsGetParams{
				ExpandWildcards: "open,hidden",
			},
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get index settings")
		}
		for indexName, indexData := range resp.Indices {
			var settings IndexSettingsResult
			if err := json.Unmarshal(indexData.Settings, &settings.Settings); err != nil {
				continue
			}
			if settings.Settings.Index.Hidden == "true" {
				hiddenIndices[indexName] = true
			}
		}
		return hiddenIndices, nil
	} else if d.typedClient != nil {
		expandWildcards := "open,hidden"
		res, err := esapi.IndicesGetSettingsRequest{
			Index:           []string{"*"},
			Name:            []string{"index.hidden"},
			ExpandWildcards: expandWildcards,
			FilterPath:      []string{"**.hidden"},
		}.Do(ctx, d.typedClient)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get index settings")
		}
		bytes, err := readBytesAndClose(res)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read index settings response")
		}
		var results map[string]IndexSettingsResult
		if err := json.Unmarshal(bytes, &results); err != nil {
			return nil, errors.Wrapf(err, "failed to parse index settings response")
		}
		for indexName, indexData := range results {
			if indexData.Settings.Index.Hidden == "true" {
				hiddenIndices[indexName] = true
			}
		}
		return hiddenIndices, nil
	}

	// Fallback to basic auth client
	resp, err := d.basicAuthClient.Do("GET", []byte("/*/_settings?filter_path=**.hidden&expand_wildcards=open,hidden"), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get index settings")
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read index settings response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed to get index settings: unexpected status code %d: %s", resp.StatusCode, string(bytes))
	}

	var results map[string]IndexSettingsResult
	if err := json.Unmarshal(bytes, &results); err != nil {
		return nil, errors.Wrapf(err, "failed to parse index settings response")
	}
	for indexName, indexData := range results {
		if indexData.Settings.Index.Hidden == "true" {
			hiddenIndices[indexName] = true
		}
	}
	return hiddenIndices, nil
}

func (d *Driver) getIndices(ctx context.Context) ([]*storepb.TableMetadata, error) {
	var indicesMetadata []*storepb.TableMetadata

	// Get hidden indices to filter them out.
	hiddenIndices, err := d.getHiddenIndices(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get hidden indices")
	}

	if d.isOpenSearch && d.opensearchAPI != nil {
		resp, err := d.opensearchAPI.Cat.Indices(ctx, &opensearchapi.CatIndicesReq{})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list indices")
		}

		for _, idx := range resp.Indices {
			if hiddenIndices[idx.Index] {
				continue
			}

			var datasize int64
			if idx.StoreSize != nil {
				datasize, err = unitConversion(*idx.StoreSize)
				if err != nil {
					return nil, err
				}
			}

			var docCount int64
			if idx.DocsCount != nil {
				docCount = int64(*idx.DocsCount)
			}

			indicesMetadata = append(indicesMetadata, &storepb.TableMetadata{
				Name:     idx.Index,
				DataSize: datasize,
				RowCount: docCount,
			})
		}

		return indicesMetadata, nil
	} else if d.typedClient != nil {
		res, err := esapi.CatIndicesRequest{Format: "json", Pretty: true}.Do(ctx, d.typedClient)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list indices")
		}

		bytes, err := readBytesAndClose(res)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list Elasticsearch indices")
		}

		var results []IndicesResult
		if err := json.Unmarshal(bytes, &results); err != nil {
			// Include response body to help debug parsing issues
			bodyPreview, truncated := common.TruncateString(string(bytes), 500)
			if truncated {
				bodyPreview += "..."
			}
			return nil, errors.Wrapf(err, "failed to parse Elasticsearch indices response: %s", bodyPreview)
		}

		for _, m := range results {
			if hiddenIndices[m.Index] {
				continue
			}

			datasize, err := unitConversion(m.IndexSize)
			if err != nil {
				return nil, err
			}
			docCount, err := parseDocCount(m.DocsCount)
			if err != nil {
				return nil, err
			}
			indicesMetadata = append(indicesMetadata, &storepb.TableMetadata{
				Name:     m.Index,
				DataSize: datasize,
				RowCount: docCount,
			})
		}
		return indicesMetadata, nil
	}

	// Fallback to basic auth client
	resp, err := d.basicAuthClient.Do("GET", []byte("/_cat/indices?format=json&pretty"), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list indices")
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read indices response body")
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		// Include response body for debugging
		return nil, errors.Errorf("failed to list indices: unexpected status code %d: %s", resp.StatusCode, string(bytes))
	}

	var results []IndicesResult
	if err := json.Unmarshal(bytes, &results); err != nil {
		// Include response body to help debug parsing issues
		bodyPreview, truncated := common.TruncateString(string(bytes), 500)
		if truncated {
			bodyPreview += "..."
		}
		return nil, errors.Wrapf(err, "failed to parse indices response: %s", bodyPreview)
	}

	for _, m := range results {
		if hiddenIndices[m.Index] {
			continue
		}

		datasize, err := unitConversion(m.IndexSize)
		if err != nil {
			return nil, err
		}
		docCount, err := parseDocCount(m.DocsCount)
		if err != nil {
			return nil, err
		}
		indicesMetadata = append(indicesMetadata, &storepb.TableMetadata{
			Name:     m.Index,
			DataSize: datasize,
			RowCount: docCount,
		})
	}
	return indicesMetadata, nil
}

// parseDocCount parses document count string, returning 0 for empty values (e.g., closed indices).
func parseDocCount(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	count, err := strconv.Atoi(s)
	return int64(count), err
}

func unitConversion(sizeWithUnit string) (int64, error) {
	// Empty size is expected for closed indices (ES returns null → unmarshals to "")
	if sizeWithUnit == "" {
		return 0, nil
	}

	sizeWithUnit = strings.ToLower(sizeWithUnit)
	sizeRe := regexp.MustCompile("([0-9.]+)([kmgtpe]?b)")
	match := sizeRe.FindSubmatch([]byte(sizeWithUnit))

	// Non-empty string that doesn't match expected format is unexpected
	if len(match) < 3 {
		return 0, errors.Errorf("unexpected size format: %q", sizeWithUnit)
	}

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
	case "tb":
		size *= 1024 * 1024 * 1024 * 1024
	case "pb":
		size *= 1024 * 1024 * 1024 * 1024 * 1024
	case "eb":
		size *= 1024 * 1024 * 1024 * 1024 * 1024 * 1024
	default:
		// For "b" (bytes) or any other unit, keep the size as-is
	}

	return int64(size), nil
}

func readBytesAndClose(anyResp any) ([]byte, error) {
	var body io.ReadCloser
	var statusCode int
	var isError bool

	// get body closer and status code.
	switch resp := anyResp.(type) {
	case *http.Response:
		body = resp.Body
		statusCode = resp.StatusCode
		isError = resp.StatusCode >= 400

	case *esapi.Response:
		body = resp.Body
		statusCode = resp.StatusCode
		isError = resp.IsError()

	default:
		return nil, errors.New("not supported response type")
	}

	// read bytes.
	bytes, err := io.ReadAll(body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	if err := body.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close response body")
	}

	// Check for error status codes
	if isError {
		return bytes, errors.Errorf("request failed with status code %d: %s", statusCode, string(bytes))
	}

	return bytes, nil
}
