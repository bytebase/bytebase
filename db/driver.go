package db

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

type Type string

const (
	Mysql Type = "MYSQL"
)

func (e Type) String() string {
	switch e {
	case Mysql:
		return "MYSQL"
	}
	return "UNKNOWN"
}

type DBIndex struct {
	Name string
	// This could refer to a column or an expression
	Expression string
	Position   int
	Type       string
	Unique     bool
	Visible    bool
	Comment    string
}

type DBColumn struct {
	Name         string
	Position     int
	Default      *string
	Nullable     bool
	Type         string
	CharacterSet string
	Collation    string
	Comment      string
}

type DBTable struct {
	Name          string
	CreatedTs     int64
	UpdatedTs     int64
	Type          string
	Engine        string
	Collation     string
	RowCount      int64
	DataSize      int64
	IndexSize     int64
	DataFree      int64
	CreateOptions string
	Comment       string
	ColumnList    []DBColumn
	IndexList     []DBIndex
}

type DBSchema struct {
	Name         string
	CharacterSet string
	Collation    string
	TableList    []DBTable
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[Type]DriverFunc)
)

type DriverConfig struct {
	Logger *zap.Logger
}

type DriverFunc func(DriverConfig) Driver

type MigrationEngine string

const (
	UI  MigrationEngine = "UI"
	VCS MigrationEngine = "VCS"
)

func (e MigrationEngine) String() string {
	switch e {
	case UI:
		return "UI"
	case VCS:
		return "VCS"
	}
	return "UNKNOWN"
}

type MigrationType string

const (
	Baseline MigrationType = "BASELINE"
	Sql      MigrationType = "SQL"
)

func (e MigrationType) String() string {
	switch e {
	case Baseline:
		return "BASELINE"
	case Sql:
		return "SQL"
	}
	return "UNKNOWN"
}

type MigrationInfoPayload struct {
	VCSPushEvent *common.VCSPushEvent `json:"pushEvent,omitempty"`
}

type MigrationInfo struct {
	Version     string
	Namespace   string
	Database    string
	Environment string
	Engine      MigrationEngine
	Type        MigrationType
	Description string
	Creator     string
	IssueId     string
	Payload     string
}

// ParseMigrationInfo derives MigrationInfo from fullPath and baseDir
// filepath is the full file path in the repository. The format is {{baseDir}}/[{{subdir}}/]/{{filename}}
// Expected filename example, {{version}} can be arbitrary string without "__"
// - {{version}}__db1 (a normal migration without description)
// - {{version}}__db1__create_t1 (a normal migration with "create t1" as description)
// - {{version}}__db1__baseline  (a baseline migration without description)
// - {{version}}__db1__baseline__create_t1  (a baseline migration with "create t1" as description)
func ParseMigrationInfo(fullPath string, baseDir string) (*MigrationInfo, error) {
	filename := filepath.Base(fullPath)
	parentDir := filepath.Base(filepath.Clean(filepath.Dir(strings.TrimPrefix(fullPath, baseDir))))
	if parentDir == "." || parentDir == "/" {
		parentDir = ""
	}
	parts := strings.Split(strings.TrimSuffix(filename, ".sql"), "__")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid filename format, got %v, want {{version}}__{{dbname}}[__{{type}}][__{{description}}].sql", filename)
	}

	mi := &MigrationInfo{
		Engine:      VCS,
		Version:     parts[0],
		Namespace:   parts[1],
		Database:    parts[1],
		Environment: parentDir,
	}

	migrationType := Sql
	description := ""
	if len(parts) > 2 {
		if parts[2] == "baseline" {
			migrationType = Baseline
			if len(parts) > 3 {
				description = strings.Join(parts[3:], " ")
			}
		} else {
			description = strings.Join(parts[2:], " ")
		}
	}
	if description == "" {
		if migrationType == Baseline {
			description = fmt.Sprintf("Create %s baseline", mi.Database)
		} else {
			description = fmt.Sprintf("Create %s migration", mi.Database)
		}
	}
	mi.Type = migrationType
	// Replace _ with space
	description = strings.ReplaceAll(description, "_", " ")
	// Capitalize first letter
	mi.Description = strings.ToUpper(description[:1]) + description[1:]

	return mi, nil
}

type MigrationHistory struct {
	ID int

	Creator   string
	CreatedTs int64
	Updater   string
	UpdatedTs int64

	Namespace         string
	Sequence          int
	Engine            MigrationEngine
	Type              MigrationType
	Version           string
	Description       string
	Statement         string
	ExecutionDuration int
	IssueId           string
	Payload           string
}

type MigrationHistoryFind struct {
	Database *string
	// If specified, then it will only fetch "Limit" most recent migration histories
	Limit *int
}

type Driver interface {
	open(config ConnectionConfig) (Driver, error)
	// Remember to call Close to avoid connection leak
	Close(ctx context.Context) error
	Ping(ctx context.Context) error
	SyncSchema(ctx context.Context) ([]*DBSchema, error)
	Execute(ctx context.Context, statement string) error

	// Migration related
	// Check whether we need to setup migration (e.g. creating/upgrading the migration related tables)
	NeedsSetupMigration(ctx context.Context) (bool, error)
	// Create or upgrade migration related tables
	SetupMigrationIfNeeded(ctx context.Context) error
	// Execute migration will apply the statement and record the migration history on success.
	ExecuteMigration(ctx context.Context, m *MigrationInfo, statement string) error
	// Find the migration history list and return most recent item first.
	FindMigrationHistoryList(ctx context.Context, find *MigrationHistoryFind) ([]*MigrationHistory, error)
}

type ConnectionConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
}

// Register makes a database driver available by the provided type.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func register(dbType Type, f DriverFunc) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if f == nil {
		panic("db: Register driver is nil")
	}
	if _, dup := drivers[dbType]; dup {
		panic("db: Register called twice for driver " + dbType)
	}
	drivers[dbType] = f
}

// Open opens a database specified by its database driver type and connection config
func Open(dbType Type, driverConfig DriverConfig, connectionConfig ConnectionConfig) (Driver, error) {
	driversMu.RLock()
	f, ok := drivers[dbType]
	driversMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("db: unknown driver %v", dbType)
	}

	driver, err := f(driverConfig).open(connectionConfig)
	if err != nil {
		return nil, err
	}

	if err := driver.Ping(context.Background()); err != nil {
		return nil, err
	}

	return driver, nil
}
