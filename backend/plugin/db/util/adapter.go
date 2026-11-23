//nolint:revive
package util

import (
	"context"
	"io"
	"sync"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// SessionQueryOptions customizes when OmniAdapter discards or keeps its held
// session after running a query.
type SessionQueryOptions struct {
	// WrapOpenError, when non-empty, wraps errors returned while lazily opening
	// the session in QueryHeldSession with this message prefix.
	WrapOpenError string

	// KeepOnError, when non-nil, is called if the query function returns an
	// error. Returning true keeps the held session (for recoverable SQL or
	// parse errors) unless ctx.Err() != nil. When nil, any error discards it.
	KeepOnError func(context.Context, error) bool

	// ShouldDiscardResults, when non-nil, is called after a query returns a nil
	// error with ctx.Err() == nil. Returning true discards the held session
	// while still returning the query results.
	ShouldDiscardResults func(context.Context, omnidriver.Session, []*omnidriver.Result) bool
}

// sessionHolder manages one lazily opened omni/driver.Session, serializing
// interactive QueryConn calls across a driver's lifetime.
type sessionHolder struct {
	mu      sync.Mutex
	session omnidriver.Session
}

// Session returns the currently held session, or nil if none is open.
func (h *sessionHolder) Session() omnidriver.Session {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.session
}

// SetSession replaces the held session. Intended for unit tests injecting a fake
// session; callers in production rely on lazy opening inside QueryHeldSession.
func (h *sessionHolder) SetSession(session omnidriver.Session) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.session = session
}

// close closes and clears the held session if one is open.
func (h *sessionHolder) close() error {
	h.mu.Lock()
	session := h.session
	h.session = nil
	h.mu.Unlock()
	if session == nil {
		return nil
	}
	return session.Close()
}

// query acquires (opening lazily if needed) the held session, runs fn under the
// session lock, and discards the session on cancellation, unrecoverable error,
// or ShouldDiscardResults.
func (h *sessionHolder) query(
	ctx context.Context,
	open func(context.Context) (omnidriver.Session, error),
	opts SessionQueryOptions,
	fn func(omnidriver.Session) ([]*omnidriver.Result, error),
) ([]*omnidriver.Result, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	session := h.session
	if session == nil {
		var err error
		session, err = open(ctx)
		if err != nil {
			return nil, err
		}
		h.session = session
	}

	results, err := fn(session)
	if err != nil {
		if ctx.Err() != nil || opts.KeepOnError == nil || !opts.KeepOnError(ctx, err) {
			h.discardLocked(session)
		}
		return results, err
	}
	if ctx.Err() != nil || (opts.ShouldDiscardResults != nil && opts.ShouldDiscardResults(ctx, session, results)) {
		h.discardLocked(session)
	}
	return results, nil
}

func (h *sessionHolder) discardLocked(session omnidriver.Session) {
	_ = session.Close()
	if h.session == session {
		h.session = nil
	}
}

// ExecuteConfig configures how OmniAdapter.ExecuteScript prepares options and
// logs for omni/driver.Executor.Execute.
type ExecuteConfig struct {
	// ParseTransactionDirective parses leading -- txn-mode and -- txn-isolation
	// directives from the script and strips them before execution.
	ParseTransactionDirective bool
	// SkipDirectiveOnCreateDatabase skips ParseTransactionDirective when
	// opts.CreateDatabase is true.
	SkipDirectiveOnCreateDatabase bool
	// StripDirectivesOnly removes directives via base.ParseTransactionConfig
	// without forwarding TransactionMode or IsolationLevel.
	StripDirectivesOnly bool
	// PassIsolation forwards the parsed IsolationLevel to Executor.Execute.
	PassIsolation bool
	// PassCreateDatabase forwards opts.CreateDatabase to Executor.Execute.
	PassCreateDatabase bool
	// PassMaxRetries forwards opts.MaximumRetries to Executor.Execute.
	PassMaxRetries bool
	// ShiftCommandRanges maps cleaned-script byte offsets back to the caller's
	// original script using DirectiveShift.
	ShiftCommandRanges bool
	// DefaultTransaction sets Transaction when directives are not parsed.
	DefaultTransaction omnidriver.TransactionMode
	// Preprocess transforms the script (after directive stripping) before
	// passing it to Executor.Execute.
	Preprocess func(string) string
	// LoggerOptions appends additional ExecuteLoggerOption values.
	LoggerOptions []ExecuteLoggerOption
}

// MySQLForkExecuteConfig is the ExecuteConfig shared by MySQL-protocol
// analytical forks (Doris and StarRocks), which ignore per-script isolation.
var MySQLForkExecuteConfig = ExecuteConfig{
	ParseTransactionDirective: true,
	PassCreateDatabase:        true,
	PassMaxRetries:            true,
}

// OmniAdapter provides shared db.Driver lifecycle, metadata sync, dump,
// held-session query, and script execution plumbing for omni/driver adapters.
type OmniAdapter struct {
	Driver omnidriver.Driver
	sessionHolder

	// WithInstanceMetadata controls whether SyncInstance populates Metadata
	// (LowerCaseTableNames and Roles).
	WithInstanceMetadata bool
	// NotFoundResource, when non-empty (e.g. "database" or "dataset"), maps
	// omnidriver.ErrNotFound from SyncDBSchema to common.NotFound.
	NotFoundResource string
	DatabaseName     string

	closersMu sync.Mutex
	closers   []func() error
}

// AdapterOptions configures an OmniAdapter's metadata sync and session behavior during OpenWith.
type AdapterOptions struct {
	WithInstanceMetadata bool
	NotFoundResource     string
	DatabaseName         string
	// EagerSession opens and retains a query session during OpenWith, closing
	// the driver if opening the session fails.
	EagerSession bool
}

// OpenWith configures the adapter from opts, runs open to acquire any auxiliary
// resources (such as SSH tunnels or Cloud SQL dialers) and open the underlying
// omni driver, and cleans up via Close if open (or eager session opening) fails.
func (a *OmniAdapter) OpenWith(
	ctx context.Context,
	opts AdapterOptions,
	open func(*OmniAdapter) (omnidriver.Driver, error),
) error {
	a.WithInstanceMetadata = opts.WithInstanceMetadata
	a.NotFoundResource = opts.NotFoundResource
	a.DatabaseName = opts.DatabaseName
	impl, err := open(a)
	if err != nil {
		_ = a.Close(ctx)
		return err
	}
	a.Driver = impl
	if opts.EagerSession {
		session, err := a.OpenSession(ctx)
		if err != nil {
			_ = a.Close(ctx)
			a.Driver = nil
			return err
		}
		a.SetSession(session)
	}
	return nil
}

// addCloser registers c to be closed when Close runs.
func (a *OmniAdapter) addCloser(c io.Closer) {
	if c == nil {
		return
	}
	a.AddCloseFunc(c.Close)
}

// AddCloseFunc registers fn to be called when Close runs.
func (a *OmniAdapter) AddCloseFunc(fn func() error) {
	if fn == nil {
		return
	}
	a.closersMu.Lock()
	a.closers = append(a.closers, fn)
	a.closersMu.Unlock()
}

// Close closes any held query session, the underlying omni driver, and all
// registered auxiliary closers (such as SSH tunnels or cloud IAM dialers).
func (a *OmniAdapter) Close(context.Context) error {
	var err error
	err = multierr.Append(err, a.close())
	if a.Driver != nil {
		err = multierr.Append(err, a.Driver.Close())
	}
	a.closersMu.Lock()
	closers := a.closers
	a.closers = nil
	a.closersMu.Unlock()
	for _, fn := range closers {
		err = multierr.Append(err, fn())
	}
	return err
}

// Ping reaches the server through the underlying omni driver.
func (a *OmniAdapter) Ping(ctx context.Context) error {
	return a.Driver.Ping(ctx)
}

// SyncInstance reads instance metadata from the underlying omni Syncer and
// converts it to db.InstanceMetadata.
func (a *OmniAdapter) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	syncer, ok := a.Driver.(omnidriver.Syncer)
	if !ok {
		return nil, errors.Errorf("driver %T does not implement omni/driver.Syncer", a.Driver)
	}
	instance, err := syncer.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return omniInstanceToMetadata(instance, a.WithInstanceMetadata), nil
}

// SyncDBSchema reads the connected database schema from the underlying omni
// Syncer, translating omnidriver.ErrNotFound when NotFoundResource is set.
func (a *OmniAdapter) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	syncer, ok := a.Driver.(omnidriver.Syncer)
	if !ok {
		return nil, errors.Errorf("driver %T does not implement omni/driver.Syncer", a.Driver)
	}
	metadata, err := syncer.SyncDatabase(ctx)
	if err != nil {
		if a.NotFoundResource != "" && errors.Is(err, omnidriver.ErrNotFound) {
			return nil, common.Errorf(common.NotFound, "%s %q not found", a.NotFoundResource, a.DatabaseName)
		}
		return nil, err
	}
	return metadata, nil
}

// Dump writes the database schema DDL through the underlying omni Dumper.
func (a *OmniAdapter) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	dumper, ok := a.Driver.(omnidriver.Dumper)
	if !ok {
		return errors.Errorf("driver %T does not implement omni/driver.Dumper", a.Driver)
	}
	return dumper.Dump(ctx, out, dbMetadata)
}

// OpenSession opens a new session from the underlying omni Querier.
func (a *OmniAdapter) OpenSession(ctx context.Context) (omnidriver.Session, error) {
	querier, ok := a.Driver.(omnidriver.Querier)
	if !ok {
		return nil, errors.Errorf("driver %T does not implement omni/driver.Querier", a.Driver)
	}
	return querier.Session(ctx)
}

// ProbeOmni validates pCtx against opts, opens a fresh session via OpenSession,
// runs the probe, and closes the session before returning.
func (a *OmniAdapter) ProbeOmni(ctx context.Context, pCtx db.ProbeContext, statement string, opts OmniProbeOptions) (db.ProbeResult, error) {
	if err := validateOmniProbeContext(pCtx, opts); err != nil {
		return db.ProbeResult{}, err
	}
	session, err := a.OpenSession(ctx)
	if err != nil {
		return db.ProbeResult{}, err
	}
	defer func() { _ = session.Close() }()
	return ProbeOmniSession(ctx, session, pCtx, statement, opts)
}

// QueryHeldSession runs statement on the adapter's held session, opening it
// lazily on first call and serializing concurrent callers.
func (a *OmniAdapter) QueryHeldSession(
	ctx context.Context,
	statement string,
	queryOpts omnidriver.QueryOptions,
	sessionOpts SessionQueryOptions,
) ([]*omnidriver.Result, error) {
	open := a.OpenSession
	if sessionOpts.WrapOpenError != "" {
		prefix := sessionOpts.WrapOpenError
		open = func(ctx context.Context) (omnidriver.Session, error) {
			session, err := a.OpenSession(ctx)
			if err != nil {
				return nil, errors.Wrap(err, prefix)
			}
			return session, nil
		}
	}
	return a.query(ctx, open, sessionOpts, func(session omnidriver.Session) ([]*omnidriver.Result, error) {
		return session.Query(ctx, statement, queryOpts)
	})
}

// QueryEphemeral opens a fresh session, runs fn, and closes the session before returning.
func (a *OmniAdapter) QueryEphemeral(ctx context.Context, fn func(omnidriver.Session) error) error {
	session, err := a.OpenSession(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()
	return fn(session)
}

// QueryEphemeralSession opens a fresh session, runs statement with queryOpts,
// and closes the session before returning.
func (a *OmniAdapter) QueryEphemeralSession(
	ctx context.Context,
	statement string,
	queryOpts omnidriver.QueryOptions,
) ([]*omnidriver.Result, error) {
	var results []*omnidriver.Result
	err := a.QueryEphemeral(ctx, func(session omnidriver.Session) error {
		var queryErr error
		results, queryErr = session.Query(ctx, statement, queryOpts)
		return queryErr
	})
	return results, err
}

// QueryHeldConn runs statement on the held session and converts the results
// into API QueryResults, discarding partial results on error.
func (a *OmniAdapter) QueryHeldConn(
	ctx context.Context,
	statement string,
	queryOpts omnidriver.QueryOptions,
	sessionOpts SessionQueryOptions,
	resultOpts QueryResultOptions,
) ([]*v1pb.QueryResult, error) {
	results, err := a.QueryHeldSession(ctx, statement, queryOpts, sessionOpts)
	if err != nil {
		return nil, err
	}
	return OmniResultsToQueryResultsWith(results, queryOpts.MaxResultBytes, resultOpts), nil
}

// QueryEphemeralConn runs statement on a fresh session, closes the session,
// and converts the results into API QueryResults.
func (a *OmniAdapter) QueryEphemeralConn(
	ctx context.Context,
	statement string,
	queryOpts omnidriver.QueryOptions,
	resultOpts QueryResultOptions,
) ([]*v1pb.QueryResult, error) {
	results, err := a.QueryEphemeralSession(ctx, statement, queryOpts)
	if err != nil {
		return nil, err
	}
	return OmniResultsToQueryResultsWith(results, queryOpts.MaxResultBytes, resultOpts), nil
}

// BuildExecuteOptions strips configured transaction directives from statement
// and constructs the matching omni/driver.ExecuteOptions according to cfg.
func BuildExecuteOptions(statement string, opts db.ExecuteOptions, cfg ExecuteConfig) (string, omnidriver.ExecuteOptions) {
	cleaned := statement
	var txMode omnidriver.TransactionMode
	var isolation omnidriver.IsolationLevel

	switch {
	case cfg.StripDirectivesOnly:
		_, cleaned = base.ParseTransactionConfig(statement)
	case cfg.ParseTransactionDirective && (!cfg.SkipDirectiveOnCreateDatabase || !opts.CreateDatabase):
		txMode, isolation, cleaned = parseTransactionMode(statement)
	default:
		txMode = cfg.DefaultTransaction
	}

	loggerOpts := make([]ExecuteLoggerOption, 0, len(cfg.LoggerOptions)+1)
	if cfg.ShiftCommandRanges {
		shift := DirectiveShift(statement, cleaned)
		loggerOpts = append(loggerOpts, WithCommandRange(func(command omnidriver.Command) *storepb.Range {
			start, end := shift(command.ByteStart, command.ByteEnd)
			return &storepb.Range{Start: int32(start), End: int32(end)}
		}))
	}
	loggerOpts = append(loggerOpts, cfg.LoggerOptions...)

	if cfg.Preprocess != nil {
		cleaned = cfg.Preprocess(cleaned)
	}

	execOpts := omnidriver.ExecuteOptions{
		Transaction: txMode,
		Log:         NewExecuteLogger(opts, loggerOpts...),
	}
	if cfg.PassIsolation {
		execOpts.Isolation = isolation
	}
	if cfg.PassCreateDatabase {
		execOpts.CreateDatabase = opts.CreateDatabase
	}
	if cfg.PassMaxRetries {
		execOpts.MaxRetries = opts.MaximumRetries
	}
	return cleaned, execOpts
}

// ExecuteScript runs statement through the underlying omni Executor according to cfg.
func (a *OmniAdapter) ExecuteScript(ctx context.Context, statement string, opts db.ExecuteOptions, cfg ExecuteConfig) (int64, error) {
	executor, ok := a.Driver.(omnidriver.Executor)
	if !ok {
		return 0, errors.Errorf("driver %T does not implement omni/driver.Executor", a.Driver)
	}
	cleaned, execOpts := BuildExecuteOptions(statement, opts, cfg)
	return executor.Execute(ctx, cleaned, execOpts)
}
