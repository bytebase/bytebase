package schema

import (
	"strings"
	"sync"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// IndexComparer provides database-specific index comparison logic.
type IndexComparer interface {
	// CompareIndexWhereConditions compares WHERE conditions from two index definitions
	CompareIndexWhereConditions(def1, def2 string) bool

	// ExtractWhereClauseFromIndexDef extracts the WHERE clause from an index definition
	ExtractWhereClauseFromIndexDef(definition string) string
}

// DefaultIndexComparer provides default index comparison logic using simple string matching.
type DefaultIndexComparer struct{}

// CompareIndexWhereConditions provides a default implementation using simple string extraction.
func (c *DefaultIndexComparer) CompareIndexWhereConditions(def1, def2 string) bool {
	whereClause1 := c.ExtractWhereClauseFromIndexDef(def1)
	whereClause2 := c.ExtractWhereClauseFromIndexDef(def2)

	// Simple string comparison - not ideal but works for basic cases
	return whereClause1 == whereClause2
}

// ExtractWhereClauseFromIndexDef provides a default implementation using simple string matching.
func (*DefaultIndexComparer) ExtractWhereClauseFromIndexDef(definition string) string {
	// This is the original hack method - kept for compatibility
	if definition == "" {
		return ""
	}

	// Find WHERE keyword (case insensitive)
	defUpper := strings.ToUpper(definition)
	whereIdx := strings.Index(defUpper, " WHERE ")

	if whereIdx == -1 {
		return ""
	}

	// Extract everything after WHERE, removing trailing semicolon if present
	whereClause := strings.TrimSpace(definition[whereIdx+7:]) // 7 = len(" WHERE ")
	whereClause = strings.TrimSuffix(whereClause, ";")
	return strings.TrimSpace(whereClause)
}

// Global registry for index comparers
var (
	indexComparers     = make(map[storepb.Engine]IndexComparer)
	indexComparerMutex sync.RWMutex
)

// RegisterIndexComparer registers an index comparer for a specific engine.
func RegisterIndexComparer(engine storepb.Engine, comparer IndexComparer) {
	indexComparerMutex.Lock()
	defer indexComparerMutex.Unlock()
	indexComparers[engine] = comparer
}

// GetIndexComparer returns the registered index comparer for the given engine.
func GetIndexComparer(engine storepb.Engine) IndexComparer {
	indexComparerMutex.RLock()
	defer indexComparerMutex.RUnlock()

	if comparer, exists := indexComparers[engine]; exists {
		return comparer
	}

	// Return default comparer if no specific comparer is registered
	return &DefaultIndexComparer{}
}

func init() {
	// Register default comparers for engines that don't have specific implementations
	defaultComparer := &DefaultIndexComparer{}

	RegisterIndexComparer(storepb.Engine_MYSQL, defaultComparer)
	RegisterIndexComparer(storepb.Engine_TIDB, defaultComparer)
	RegisterIndexComparer(storepb.Engine_ORACLE, defaultComparer)
	RegisterIndexComparer(storepb.Engine_MSSQL, defaultComparer)
	RegisterIndexComparer(storepb.Engine_SQLITE, defaultComparer)
	RegisterIndexComparer(storepb.Engine_MONGODB, defaultComparer)
	RegisterIndexComparer(storepb.Engine_REDIS, defaultComparer)
	RegisterIndexComparer(storepb.Engine_SNOWFLAKE, defaultComparer)
	RegisterIndexComparer(storepb.Engine_CLICKHOUSE, defaultComparer)
}
