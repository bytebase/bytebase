package schema

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ViewChangeType represents the type of change detected in a view.
type ViewChangeType int

const (
	// ViewChangeNone indicates no change.
	ViewChangeNone ViewChangeType = iota
	// ViewChangeDefinition indicates the view definition has changed.
	ViewChangeDefinition
	// ViewChangeComment indicates only the comment has changed.
	ViewChangeComment
	// ViewChangeColumn indicates column metadata has changed.
	ViewChangeColumn
	// ViewChangeTrigger indicates triggers have changed.
	ViewChangeTrigger
	// ViewChangeOther indicates other metadata changes.
	ViewChangeOther
)

// ViewChange represents a detected change in a view.
type ViewChange struct {
	Type        ViewChangeType
	Description string
	// RequiresRecreation indicates if the change requires dropping and recreating the view.
	RequiresRecreation bool
}

// MaterializedViewChangeType represents the type of change detected in a materialized view.
type MaterializedViewChangeType int

const (
	// MaterializedViewChangeNone indicates no change.
	MaterializedViewChangeNone MaterializedViewChangeType = iota
	// MaterializedViewChangeDefinition indicates the materialized view definition has changed.
	MaterializedViewChangeDefinition
	// MaterializedViewChangeComment indicates only the comment has changed.
	MaterializedViewChangeComment
	// MaterializedViewChangeIndex indicates indexes have changed.
	MaterializedViewChangeIndex
	// MaterializedViewChangeTrigger indicates triggers have changed.
	MaterializedViewChangeTrigger
	// MaterializedViewChangeOther indicates other metadata changes.
	MaterializedViewChangeOther
)

// MaterializedViewChange represents a detected change in a materialized view.
type MaterializedViewChange struct {
	Type        MaterializedViewChangeType
	Description string
	// RequiresRecreation indicates if the change requires dropping and recreating the materialized view.
	RequiresRecreation bool
}

// ViewComparer provides engine-specific view comparison logic.
type ViewComparer interface {
	// CompareView compares two views and returns the detected changes.
	CompareView(oldView, newView *storepb.ViewMetadata) ([]ViewChange, error)

	// CompareMaterializedView compares two materialized views and returns the detected changes.
	CompareMaterializedView(oldMV, newMV *storepb.MaterializedViewMetadata) ([]MaterializedViewChange, error)
}

// DefaultViewComparer provides default view comparison logic that can be used by most engines.
type DefaultViewComparer struct{}

// CompareView compares two views using default logic.
func (*DefaultViewComparer) CompareView(oldView, newView *storepb.ViewMetadata) ([]ViewChange, error) {
	if oldView == nil || newView == nil {
		return nil, nil
	}

	var changes []ViewChange

	// Compare definition
	if oldView.Definition != newView.Definition {
		changes = append(changes, ViewChange{
			Type:               ViewChangeDefinition,
			Description:        "View definition changed",
			RequiresRecreation: true,
		})
	}

	// Compare comment
	if oldView.Comment != newView.Comment {
		changes = append(changes, ViewChange{
			Type:               ViewChangeComment,
			Description:        "View comment changed",
			RequiresRecreation: false,
		})
	}

	return changes, nil
}

// CompareMaterializedView compares two materialized views using default logic.
func (*DefaultViewComparer) CompareMaterializedView(oldMV, newMV *storepb.MaterializedViewMetadata) ([]MaterializedViewChange, error) {
	if oldMV == nil || newMV == nil {
		return nil, nil
	}

	var changes []MaterializedViewChange

	// Compare definition
	if oldMV.Definition != newMV.Definition {
		changes = append(changes, MaterializedViewChange{
			Type:               MaterializedViewChangeDefinition,
			Description:        "Materialized view definition changed",
			RequiresRecreation: true,
		})
	}

	// Compare comment
	if oldMV.Comment != newMV.Comment {
		changes = append(changes, MaterializedViewChange{
			Type:               MaterializedViewChangeComment,
			Description:        "Materialized view comment changed",
			RequiresRecreation: false,
		})
	}

	return changes, nil
}

// viewComparerRegistry holds engine-specific view comparers.
var viewComparerRegistry = make(map[storepb.Engine]ViewComparer)

// RegisterViewComparer registers a view comparer for a specific engine.
func RegisterViewComparer(engine storepb.Engine, comparer ViewComparer) {
	viewComparerRegistry[engine] = comparer
}

// GetViewComparer returns the view comparer for a specific engine.
// If no engine-specific comparer is registered, it returns the default comparer.
func GetViewComparer(engine storepb.Engine) ViewComparer {
	if comparer, exists := viewComparerRegistry[engine]; exists {
		return comparer
	}
	return &DefaultViewComparer{}
}

// init registers the default comparer for common engines.
func init() {
	defaultComparer := &DefaultViewComparer{}

	// Register default comparer for engines that don't need special handling
	RegisterViewComparer(storepb.Engine_MYSQL, defaultComparer)
	RegisterViewComparer(storepb.Engine_TIDB, defaultComparer)
	RegisterViewComparer(storepb.Engine_ORACLE, defaultComparer)
	RegisterViewComparer(storepb.Engine_MSSQL, defaultComparer)

	// Note: PostgreSQL might need a custom comparer to be registered separately
	// due to its unique handling of materialized views and indexes.
}
