package db

import (
	"context"
	"fmt"
	"regexp"
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

type DBUser struct {
	Name  string
	Grant string
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
	UserList     []DBUser
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
	Migrate  MigrationType = "MIGRATE"
	Branch   MigrationType = "BRANCH"
)

func (e MigrationType) String() string {
	switch e {
	case Baseline:
		return "BASELINE"
	case Migrate:
		return "MIGRATE"
	case Branch:
		return "BRANCH"
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

// ParseMigrationInfo matches filePath against filePathTemplate
// If filePath matches, then it will derive MigrationInfo from the filePath.
// Both filePath and filePathTemplate are the full file path (including the base directory) of the repository.
func ParseMigrationInfo(filePath string, filePathTemplate string) (*MigrationInfo, error) {
	placeholderList := []string{
		"ENV_NAME",
		"VERSION",
		"DB_NAME",
		"TYPE",
		"DESCRIPTION",
	}
	filePathRegex := filePathTemplate
	for _, placeholder := range placeholderList {
		filePathRegex = strings.ReplaceAll(filePathRegex, fmt.Sprintf("{{%s}}", placeholder), fmt.Sprintf("(?P<%s>[a-zA-Z0-9+-=/_#?!$. ]+)", placeholder))
	}
	myRegex, err := regexp.Compile(filePathRegex)
	if err != nil {
		return nil, fmt.Errorf("invalid file path template: %q", filePathTemplate)
	}
	if !myRegex.MatchString(filePath) {
		return nil, fmt.Errorf("file path %q does not match file path template %q", filePath, filePathTemplate)
	}

	mi := &MigrationInfo{
		Engine: VCS,
		Type:   Migrate,
	}
	matchList := myRegex.FindStringSubmatch(filePath)
	for _, placeholder := range placeholderList {
		index := myRegex.SubexpIndex(placeholder)
		if index >= 0 {
			if placeholder == "ENV_NAME" {
				mi.Environment = matchList[index]
			} else if placeholder == "VERSION" {
				mi.Version = matchList[index]
			} else if placeholder == "DB_NAME" {
				mi.Namespace = matchList[index]
				mi.Database = matchList[index]
			} else if placeholder == "TYPE" {
				if matchList[index] == "baseline" {
					mi.Type = Baseline
				} else if matchList[index] == "migrate" {
					mi.Type = Migrate
				} else {
					return nil, fmt.Errorf("file path %q contains invalid migration type %q, must be 'baseline' or 'migrate'", filePath, matchList[index])
				}
			} else if placeholder == "DESCRIPTION" {
				mi.Description = matchList[index]
			}
		}
	}

	if mi.Version == "" {
		return nil, fmt.Errorf("file path %q does not contain {{VERSION}}, configured file path template %q", filePath, filePathTemplate)
	}
	if mi.Namespace == "" {
		return nil, fmt.Errorf("file path %q does not contain {{DB_NAME}}, configured file path template %q", filePath, filePathTemplate)
	}

	if mi.Description == "" {
		if mi.Type == Baseline {
			mi.Description = fmt.Sprintf("Create %s baseline", mi.Database)
		} else {
			mi.Description = fmt.Sprintf("Create %s migration", mi.Database)
		}
	} else {
		// Replace _ with space
		mi.Description = strings.ReplaceAll(mi.Description, "_", " ")
		// Capitalize first letter
		mi.Description = strings.ToUpper(mi.Description[:1]) + mi.Description[1:]
	}

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

type ConnectionConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
}

// Context not used for establishing the db connection, but is useful for logging.
type ConnectionContext struct {
	EnvironmentName string
	InstanceName    string
}

type Driver interface {
	open(config ConnectionConfig, ctx ConnectionContext) (Driver, error)
	// Remember to call Close to avoid connection leak
	Close(ctx context.Context) error
	Ping(ctx context.Context) error
	SyncSchema(ctx context.Context) ([]*DBUser, []*DBSchema, error)
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
func Open(dbType Type, driverConfig DriverConfig, connectionConfig ConnectionConfig, ctx ConnectionContext) (Driver, error) {
	driversMu.RLock()
	f, ok := drivers[dbType]
	driversMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("db: unknown driver %v", dbType)
	}

	driver, err := f(driverConfig).open(connectionConfig, ctx)
	if err != nil {
		return nil, err
	}

	if err := driver.Ping(context.Background()); err != nil {
		return nil, err
	}

	return driver, nil
}
