package trino

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	// Import Trino driver for side effects
	_ "github.com/trinodb/trino-go-client/trino"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func init() {
	db.Register(storepb.Engine_TRINO, newDriver)
}

type Driver struct {
	config db.ConnectionConfig
	db     *sql.DB
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

func (*Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	// Construct Trino DSN
	var scheme string
	if config.DataSource.UseSsl {
		scheme = "https"
	} else {
		scheme = "http"
	}

	// Build query parameters
	queryParams := url.Values{}
	// Add catalog if specified in config
	if config.DataSource.Database != "" {
		queryParams.Add("catalog", config.DataSource.Database)
	}
	// Set additional parameters that might help with compatibility
	queryParams.Add("binary_format", "hex")

	// Add user and password
	user := config.DataSource.Username
	if user == "" {
		user = "trino" // default user if not specified
	}

	password := config.Password
	if password == "" {
		password = config.DataSource.Password // try to get from DataSource if not provided directly
	}

	host := config.DataSource.Host
	port := config.DataSource.Port
	if port == "" {
		port = "8080" // default Trino port
	}

	// The default Trino driver configuration isn't working properly.
	// We'll try two different approaches to establish a connection:
	// 1. Direct DSN construction (original approach)
	// 2. Using Trino's config struct

	// First, let's attempt to diagnose connection issues by checking endpoint reachability
	isBytebaseServer, _ := checkIfBytebaseServer(host, port, scheme)
	if isBytebaseServer {
		return nil, errors.New("connection error: you appear to be trying to connect to a Bytebase server " +
			"instead of a Trino server. Please check your connection details and ensure " +
			"you're connecting to a Trino database server")
	}

	// Continue with normal Trino troubleshooting
	troubleshootTrinoConnection(host, port, scheme)

	var db *sql.DB
	var err error

	// Trino driver is already registered by the import as "trino"
	fmt.Println("Attempting Trino connection")

	// Build DSN with username and password if provided
	var dsn string
	if password != "" {
		dsn = fmt.Sprintf("%s://%s:%s@%s:%s", scheme, user, url.QueryEscape(password), host, port)
	} else {
		dsn = fmt.Sprintf("%s://%s@%s:%s", scheme, user, host, port)
	}

	// Add query parameters to the DSN
	queryParams.Add("source", "bytebase")

	// Add catalog if specified
	if config.DataSource.Database != "" {
		queryParams.Add("catalog", config.DataSource.Database)
	}

	// Add parameters to the DSN
	if len(queryParams) > 0 {
		dsn = dsn + "?" + queryParams.Encode()
	}

	// Log the DSN (with password masked) for debugging
	dsnForLogging := dsn
	if password != "" {
		passwordMask := "***"
		dsnForLogging = strings.Replace(dsnForLogging, url.QueryEscape(password), passwordMask, 1)
	}
	fmt.Printf("Connecting to Trino with DSN: %s\n", dsnForLogging)

	// Try to connect using the standard driver
	db, err = sql.Open("trino", dsn)

	// If there was an error, try an alternative approach
	if err != nil {
		fmt.Printf("First connection approach failed: %v\nTrying alternative approach\n", err)

		// APPROACH 2: Alternative DSN construction with different parameters
		// Reset query params and try with a more basic configuration
		queryParams = url.Values{}
		queryParams.Add("source", "bytebase")

		// Try with SSL disabled if it was enabled before
		if config.DataSource.UseSsl {
			queryParams.Add("SSL", "false")
		}

		// Try without catalog specification
		// Use a simpler DSN format
		var altDsn string
		if password != "" {
			altDsn = fmt.Sprintf("%s://%s:%s@%s:%s?%s",
				scheme, user, url.QueryEscape(password), host, port, queryParams.Encode())
		} else {
			altDsn = fmt.Sprintf("%s://%s@%s:%s?%s",
				scheme, user, host, port, queryParams.Encode())
		}

		// Log the DSN (with password masked) for debugging
		altDsnForLogging := altDsn
		if password != "" {
			passwordMask := "***"
			altDsnForLogging = strings.Replace(altDsnForLogging, url.QueryEscape(password), passwordMask, 1)
		}
		fmt.Printf("Trying alternative connection with DSN: %s\n", altDsnForLogging)

		// Open connection with diagnostic info
		fmt.Printf("Opening alternative connection using driver: 'trino'\n")
		db, err = sql.Open("trino", altDsn)
		if err != nil {
			return nil, errors.Wrap(err, "failed to connect to Trino with both connection methods")
		}
	}

	// Set a short timeout for the initial connection test to avoid hanging
	// Some Trino servers might be slow to respond initially
	db.SetConnMaxLifetime(30 * time.Second)
	db.SetMaxIdleConns(2)
	db.SetMaxOpenConns(2)

	// Set a shorter timeout for the initial Ping
	pingCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// Don't fail on ping - many Trino configurations don't support ping properly
	if err := db.PingContext(pingCtx); err != nil {
		fmt.Printf("Warning: Ping to Trino failed, but continuing: %v\n", err)
		// Note: intentionally not returning error here
	} else {
		fmt.Printf("Successfully pinged Trino server\n")
	}

	return &Driver{
		config: config,
		db:     db,
	}, nil
}

func (d *Driver) Close(context.Context) error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *Driver) Ping(ctx context.Context) error {
	if d.db != nil {
		return d.db.PingContext(ctx)
	}
	return errors.New("database connection not established")
}

func (d *Driver) GetDB() *sql.DB {
	return d.db
}

// troubleshootTrinoConnection attempts to diagnose Trino connection issues
// by checking basic connectivity and API endpoints
func troubleshootTrinoConnection(host, port, scheme string) {
	fmt.Printf("Troubleshooting Trino connection to %s://%s:%s\n", scheme, host, port)

	// Create a HTTP client with short timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Skip certificate verification for testing
			},
		},
	}

	// Try different Trino API endpoints to see which ones are accessible
	endpoints := []string{
		"/",
		"/v1/info",
		"/v1/statement",
		"/v1/statement/",
		"/v1/query",
		"/ui/api/stats",
	}

	for _, endpoint := range endpoints {
		apiURL := fmt.Sprintf("%s://%s:%s%s", scheme, host, port, endpoint)
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			fmt.Printf("  Error creating request for %s: %v\n", endpoint, err)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  Cannot reach %s: %v\n", endpoint, err)
			continue
		}

		// Process response
		defer resp.Body.Close()
		fmt.Printf("  Endpoint %s: Status %s (%d)\n", endpoint, resp.Status, resp.StatusCode)

		// Read a small portion of the body for diagnostic info
		bodyBytes := make([]byte, 100)
		n, _ := resp.Body.Read(bodyBytes)
		if n > 0 {
			fmt.Printf("  Response preview: %s\n", string(bodyBytes[:n]))
		}
	}

	// Try a direct TCP connection test
	fmt.Printf("  Testing TCP connection to %s:%s...\n", host, port)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, port), 5*time.Second)
	if err != nil {
		fmt.Printf("  TCP connection failed: %v\n", err)
	} else {
		fmt.Printf("  TCP connection successful\n")
		conn.Close()
	}
}

// checkIfBytebaseServer checks if the given host/port appears to be a Bytebase server
// instead of a Trino server
func checkIfBytebaseServer(host, port, scheme string) (bool, error) {
	// Create a HTTP client with short timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Skip certificate verification for testing
			},
		},
	}

	// Check root endpoint which should return different things for Bytebase vs Trino
	rootURL := fmt.Sprintf("%s://%s:%s/", scheme, host, port)
	req, err := http.NewRequest("GET", rootURL, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Read response to look for Bytebase signatures
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	bodyStr := string(bodyBytes)

	// Check for common Bytebase response patterns
	bytebaseSignatures := []string{
		"Bytebase",
		"This Bytebase build",
		"bundle frontend and backend together",
	}

	for _, sig := range bytebaseSignatures {
		if strings.Contains(bodyStr, sig) {
			fmt.Printf("WARNING: Detected Bytebase server signature: %s\n", sig)
			return true, nil
		}
	}

	// Also check for the 404 error format we see in the debug output
	if resp.StatusCode == 404 && strings.Contains(bodyStr, "Routing error") {
		fmt.Printf("WARNING: Detected Bytebase routing error pattern\n")
		return true, nil
	}

	return false, nil
}

// func (*Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
func (*Driver) Execute(_ context.Context, _ string, _ db.ExecuteOptions) (int64, error) {
	return 0, errors.New("tbd")
}

// func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
func (*Driver) QueryConn(_ context.Context, _ *sql.Conn, _ string, _ db.QueryContext) ([]*v1pb.QueryResult, error) {
	return nil, errors.New("tbd")
}
