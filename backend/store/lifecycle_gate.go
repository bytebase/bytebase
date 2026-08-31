package store

import (
	"context"
	"database/sql"
	"slices"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// ErrLifecycleBusy reports fail-fast contention on a lifecycle gate.
var ErrLifecycleBusy = errors.New("resource is busy; retry")

type lifecycleRequirement int

const (
	lifecycleExisting lifecycleRequirement = iota
	lifecycleActive
)

type lifecycleScope struct {
	projects  map[string]lifecycleRequirement
	instances map[string]lifecycleRequirement
}

func (s *lifecycleScope) addProject(resourceID string, requirement lifecycleRequirement) {
	if resourceID == "" {
		return
	}
	if s.projects == nil {
		s.projects = make(map[string]lifecycleRequirement)
	}
	current, ok := s.projects[resourceID]
	if !ok || requirement > current {
		s.projects[resourceID] = requirement
	}
}

func (s *lifecycleScope) addInstance(resourceID string, requirement lifecycleRequirement) {
	if resourceID == "" {
		return
	}
	if s.instances == nil {
		s.instances = make(map[string]lifecycleRequirement)
	}
	current, ok := s.instances[resourceID]
	if !ok || requirement > current {
		s.instances[resourceID] = requirement
	}
}

func mergeLifecycleScopes(scopes ...lifecycleScope) lifecycleScope {
	merged := lifecycleScope{}
	for _, scope := range scopes {
		for resourceID, requirement := range scope.projects {
			merged.addProject(resourceID, requirement)
		}
		for resourceID, requirement := range scope.instances {
			merged.addInstance(resourceID, requirement)
		}
	}
	return merged
}

type lifecycleResourceKind int

const (
	lifecycleProject lifecycleResourceKind = iota
	lifecycleInstance
)

type lifecycleGate struct {
	kind        lifecycleResourceKind
	resourceID  string
	requirement lifecycleRequirement
}

func (s lifecycleScope) orderedGates() []lifecycleGate {
	projectIDs := make([]string, 0, len(s.projects))
	for resourceID := range s.projects {
		projectIDs = append(projectIDs, resourceID)
	}
	slices.Sort(projectIDs)

	instanceIDs := make([]string, 0, len(s.instances))
	for resourceID := range s.instances {
		instanceIDs = append(instanceIDs, resourceID)
	}
	slices.Sort(instanceIDs)

	gates := make([]lifecycleGate, 0, len(projectIDs)+len(instanceIDs))
	for _, resourceID := range projectIDs {
		gates = append(gates, lifecycleGate{kind: lifecycleProject, resourceID: resourceID, requirement: s.projects[resourceID]})
	}
	for _, resourceID := range instanceIDs {
		gates = append(gates, lifecycleGate{kind: lifecycleInstance, resourceID: resourceID, requirement: s.instances[resourceID]})
	}
	return gates
}

// runLifecycleWrite runs a purge-managed write under shared lifecycle gates.
func (s *Store) runLifecycleWrite(ctx context.Context, scope lifecycleScope, fn func(*sql.Tx) error) error {
	return s.runLifecycle(ctx, scope, lifecycleScope{}, fn)
}

// RunActiveProjectAndInstancesLifecycleWrite runs a caller-owned transaction
// that creates project data targeting active instances.
func (s *Store) RunActiveProjectAndInstancesLifecycleWrite(ctx context.Context, projectID string, instanceIDs []string, fn func(*sql.Tx) error) error {
	scope := lifecycleScope{}
	scope.addProject(projectID, lifecycleActive)
	for _, instanceID := range instanceIDs {
		scope.addInstance(instanceID, lifecycleActive)
	}
	return s.runLifecycleWrite(ctx, scope, fn)
}

// RunExistingProjectLifecycleWrite runs a caller-owned transaction that may
// intentionally continue work for an archived project before it is purged.
func (s *Store) RunExistingProjectLifecycleWrite(ctx context.Context, projectID string, fn func(*sql.Tx) error) error {
	return s.runProjectLifecycleWrite(ctx, projectID, lifecycleExisting, fn)
}

func (s *Store) runProjectLifecycleWrite(ctx context.Context, projectID string, requirement lifecycleRequirement, fn func(*sql.Tx) error) error {
	scope := lifecycleScope{}
	scope.addProject(projectID, requirement)
	return s.runLifecycleWrite(ctx, scope, fn)
}

// runLifecycleTransition runs an archive, restore, or purge under exclusive
// target gates and shared ancestor gates.
func (s *Store) runLifecycleTransition(ctx context.Context, sharedScope, exclusiveScope lifecycleScope, fn func(*sql.Tx) error) error {
	return s.runLifecycle(ctx, sharedScope, exclusiveScope, fn)
}

func (s *Store) runLifecycle(ctx context.Context, sharedScope, exclusiveScope lifecycleScope, fn func(*sql.Tx) error) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin lifecycle transaction")
	}
	defer tx.Rollback()

	merged := mergeLifecycleScopes(sharedScope, exclusiveScope)
	for _, gate := range merged.orderedGates() {
		exclusive := lifecycleScopeContains(exclusiveScope, gate)
		acquired, err := tryLifecycleGate(ctx, tx, gate, exclusive)
		if err != nil {
			return errors.Wrap(err, "failed to acquire lifecycle gate")
		}
		if !acquired {
			return common.Wrap(ErrLifecycleBusy, common.Conflict)
		}
	}

	if err := validateLifecycleScope(ctx, tx, merged); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return errors.Wrap(tx.Commit(), "failed to commit lifecycle transaction")
}

func lifecycleScopeContains(scope lifecycleScope, gate lifecycleGate) bool {
	switch gate.kind {
	case lifecycleProject:
		_, ok := scope.projects[gate.resourceID]
		return ok
	case lifecycleInstance:
		_, ok := scope.instances[gate.resourceID]
		return ok
	default:
		return false
	}
}

func tryLifecycleGate(ctx context.Context, tx *sql.Tx, gate lifecycleGate, exclusive bool) (bool, error) {
	var namespace AdvisoryLockKey
	switch gate.kind {
	case lifecycleProject:
		namespace = AdvisoryLockKeyProjectLifecycle
	case lifecycleInstance:
		namespace = AdvisoryLockKeyInstanceLifecycle
	default:
		return false, errors.Errorf("unknown lifecycle resource kind %d", gate.kind)
	}
	query := "SELECT pg_try_advisory_xact_lock_shared($1, hashtext($2))"
	if exclusive {
		query = "SELECT pg_try_advisory_xact_lock($1, hashtext($2))"
	}
	var acquired bool
	if err := tx.QueryRowContext(ctx, query, int32(namespace), gate.resourceID).Scan(&acquired); err != nil {
		return false, err
	}
	return acquired, nil
}

func validateLifecycleScope(ctx context.Context, tx *sql.Tx, scope lifecycleScope) error {
	for _, gate := range scope.orderedGates() {
		var deleted bool
		var query string
		switch gate.kind {
		case lifecycleProject:
			query = "SELECT deleted FROM project WHERE resource_id = $1"
		case lifecycleInstance:
			query = "SELECT deleted FROM instance WHERE resource_id = $1"
		default:
			return errors.Errorf("unknown lifecycle resource kind %d", gate.kind)
		}
		if err := tx.QueryRowContext(ctx, query, gate.resourceID).Scan(&deleted); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return common.Errorf(common.NotFound, "%s %s not found", lifecycleResourceName(gate.kind), gate.resourceID)
			}
			return errors.Wrapf(err, "failed to validate %s %s", lifecycleResourceName(gate.kind), gate.resourceID)
		}
		if deleted && gate.requirement == lifecycleActive {
			return common.Errorf(common.NotFound, "%s %s is archived", lifecycleResourceName(gate.kind), gate.resourceID)
		}
	}
	return nil
}

func lifecycleResourceName(kind lifecycleResourceKind) string {
	if kind == lifecycleProject {
		return "project"
	}
	return "instance"
}
