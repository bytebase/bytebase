package plancheck

import storepb "github.com/bytebase/bytebase/backend/generated-go/store"

// CheckTarget represents a derived check target from a plan.
// This is computed at runtime from the plan's specs, not stored.
type CheckTarget struct {
	// Target is the database resource name: instances/{instance}/databases/{database}
	Target string
	// SheetSha256 is the content hash of the SQL sheet
	SheetSha256 string
	// EnablePriorBackup indicates if backup before migration is enabled
	EnablePriorBackup bool
	// EnableGhost indicates if gh-ost online migration is enabled
	EnableGhost bool
	// GhostFlags are configuration flags for gh-ost
	GhostFlags map[string]string
	// Types are the plan check types to run for this target
	Types []storepb.PlanCheckType
}
