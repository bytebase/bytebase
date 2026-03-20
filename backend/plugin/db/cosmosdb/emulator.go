package cosmosdb

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// emulatorRESTClient makes direct REST API calls to the CosmosDB emulator,
// bypassing the SDK's query API which the vnext-preview emulator does not support.
type emulatorRESTClient struct {
	endpoint   string
	accountKey []byte
	httpClient *http.Client
}

func newEmulatorRESTClient(endpoint string) (*emulatorRESTClient, error) {
	key, err := base64.StdEncoding.DecodeString(cosmosDBEmulatorKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode emulator key")
	}
	return &emulatorRESTClient{
		endpoint:   strings.TrimRight(endpoint, "/"),
		accountKey: key,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: true, //nolint:gosec
				},
			},
		},
	}, nil
}

func (c *emulatorRESTClient) generateAuthHeader(verb, resourceType, resourceLink string, date string) string {
	stringToSign := strings.ToLower(verb) + "\n" +
		strings.ToLower(resourceType) + "\n" +
		resourceLink + "\n" +
		strings.ToLower(date) + "\n" +
		"" + "\n"

	h := hmac.New(sha256.New, c.accountKey)
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return url.QueryEscape("type=master&ver=1.0&sig=" + signature)
}

func (c *emulatorRESTClient) get(path, resourceType, resourceLink string) ([]byte, error) {
	reqURL := c.endpoint + "/" + path
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	date := time.Now().UTC().Format(http.TimeFormat)
	authHeader := c.generateAuthHeader("GET", resourceType, resourceLink, date)

	req.Header.Set("Authorization", authHeader)
	req.Header.Set("x-ms-date", date)
	req.Header.Set("x-ms-version", "2018-12-31")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

type listDatabasesResponse struct {
	Databases []struct {
		ID string `json:"id"`
	} `json:"Databases"`
}

type listContainersResponse struct {
	DocumentCollections []struct {
		ID string `json:"id"`
	} `json:"DocumentCollections"`
}

func (c *emulatorRESTClient) listDatabases() ([]string, error) {
	body, err := c.get("dbs", "dbs", "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to list databases")
	}

	var resp listDatabasesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, errors.Wrap(err, "failed to parse database list")
	}

	var names []string
	for _, db := range resp.Databases {
		names = append(names, db.ID)
	}
	return names, nil
}

func (c *emulatorRESTClient) listContainers(databaseName string) ([]string, error) {
	path := fmt.Sprintf("dbs/%s/colls", databaseName)
	resourceLink := fmt.Sprintf("dbs/%s", databaseName)
	body, err := c.get(path, "colls", resourceLink)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list containers")
	}

	var resp listContainersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, errors.Wrap(err, "failed to parse container list")
	}

	var names []string
	for _, container := range resp.DocumentCollections {
		names = append(names, container.ID)
	}
	return names, nil
}
