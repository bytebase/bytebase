package oceanbase

// ExplainJSON represents the JSON structure returned by EXPLAIN format=json
// We only care about the EST.ROWS field for row count estimation
type ExplainJSON struct {
	EstRows int64 `json:"EST.ROWS"`
}
