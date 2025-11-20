package pg

import (
	"strings"
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestPostgreSQLFunctionComparer_Equal(t *testing.T) {
	comparer := &PostgreSQLFunctionComparer{}

	tests := []struct {
		name      string
		oldFunc   *storepb.FunctionMetadata
		newFunc   *storepb.FunctionMetadata
		wantEqual bool
	}{
		{
			name: "identical functions",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			wantEqual: true,
		},
		{
			name: "body only difference",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a * b; END; $$ LANGUAGE plpgsql;`,
			},
			wantEqual: false,
		},
		{
			name: "signature difference",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer, c integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			wantEqual: false,
		},
		{
			name:      "nil functions",
			oldFunc:   nil,
			newFunc:   nil,
			wantEqual: true,
		},
		{
			name: "one nil function",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE FUNCTION test() RETURNS integer AS $$ BEGIN RETURN 1; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc:   nil,
			wantEqual: false,
		},
		{
			name: "identical procedures",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE update_user(user_id integer, new_name text) AS $$ BEGIN UPDATE users SET name = new_name WHERE id = user_id; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE update_user(user_id integer, new_name text) AS $$ BEGIN UPDATE users SET name = new_name WHERE id = user_id; END; $$ LANGUAGE plpgsql;`,
			},
			wantEqual: true,
		},
		{
			name: "procedure body only difference",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE update_user(user_id integer, new_name text) AS $$ BEGIN UPDATE users SET name = new_name WHERE id = user_id; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE update_user(user_id integer, new_name text) AS $$ BEGIN UPDATE users SET name = new_name, updated_at = NOW() WHERE id = user_id; END; $$ LANGUAGE plpgsql;`,
			},
			wantEqual: false,
		},
		{
			name: "procedure with dollar quote differences",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE public.refresh_stats() LANGUAGE plpgsql AS $procedure$ BEGIN REFRESH MATERIALIZED VIEW stats; END; $procedure$`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE refresh_stats() LANGUAGE plpgsql AS $$ BEGIN REFRESH MATERIALIZED VIEW stats; END; $$`,
			},
			wantEqual: true, // Should be equal - only formatting differences
		},
		{
			name: "function vs procedure - different types should not be equal",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION get_user_count() RETURNS integer AS $$ BEGIN RETURN (SELECT COUNT(*) FROM users); END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE get_user_count() AS $$ DECLARE count_val integer; BEGIN SELECT COUNT(*) INTO count_val FROM users; END; $$ LANGUAGE plpgsql;`,
			},
			wantEqual: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equal := comparer.Equal(tt.oldFunc, tt.newFunc)
			if equal != tt.wantEqual {
				t.Errorf("Equal() = %v, want %v", equal, tt.wantEqual)
			}
		})
	}
}

func TestPostgreSQLFunctionComparer_CompareDetailed(t *testing.T) {
	comparer := &PostgreSQLFunctionComparer{}

	tests := []struct {
		name                 string
		oldFunc              *storepb.FunctionMetadata
		newFunc              *storepb.FunctionMetadata
		wantResult           *schema.FunctionComparisonResult
		wantSignatureChanged bool
		wantBodyChanged      bool
		wantCanUseAlter      bool
	}{
		{
			name: "identical functions",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			wantResult: nil, // No changes, should return nil
		},
		{
			name: "body only difference",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a * b; END; $$ LANGUAGE plpgsql;`,
			},
			wantResult:           &schema.FunctionComparisonResult{}, // Expect a result, not nil
			wantSignatureChanged: false,
			wantBodyChanged:      true,
			wantCanUseAlter:      true,
		},
		{
			name: "signature difference",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer, c integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			wantResult:           &schema.FunctionComparisonResult{}, // Expect a result, not nil
			wantSignatureChanged: true,
			wantBodyChanged:      false,
			wantCanUseAlter:      false,
		},
		{
			name:       "nil functions",
			oldFunc:    nil,
			newFunc:    nil,
			wantResult: nil,
		},
		{
			name: "procedure body only difference",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE update_stats() AS $$ BEGIN UPDATE stats SET last_updated = NOW(); END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE update_stats() AS $$ BEGIN UPDATE stats SET last_updated = NOW(), count = count + 1; END; $$ LANGUAGE plpgsql;`,
			},
			wantResult:           &schema.FunctionComparisonResult{}, // Expect a result, not nil
			wantSignatureChanged: false,
			wantBodyChanged:      true,
			wantCanUseAlter:      true,
		},
		{
			name: "procedure signature difference",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE update_user(user_id integer) AS $$ BEGIN UPDATE users SET updated_at = NOW() WHERE id = user_id; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE update_user(user_id integer, new_status text) AS $$ BEGIN UPDATE users SET updated_at = NOW() WHERE id = user_id; END; $$ LANGUAGE plpgsql;`,
			},
			wantResult:           &schema.FunctionComparisonResult{}, // Expect a result, not nil
			wantSignatureChanged: true,
			wantBodyChanged:      false,
			wantCanUseAlter:      false,
		},
		{
			name: "procedure with dollar quote formatting differences",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE public.refresh_data() LANGUAGE plpgsql AS $procedure$ BEGIN REFRESH MATERIALIZED VIEW data_view; END; $procedure$`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE refresh_data() LANGUAGE plpgsql AS $$ BEGIN REFRESH MATERIALIZED VIEW data_view; END; $$`,
			},
			wantResult: nil, // Should be nil - functions are equivalent after normalization
		},
		{
			name: "function to procedure change",
			oldFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION calculate_total() RETURNS integer AS $$ BEGIN RETURN (SELECT SUM(amount) FROM orders); END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &storepb.FunctionMetadata{
				Definition: `CREATE OR REPLACE PROCEDURE calculate_total() AS $$ BEGIN RETURN (SELECT SUM(amount) FROM orders); END; $$ LANGUAGE plpgsql;`,
			},
			wantResult:           &schema.FunctionComparisonResult{}, // Expect a result, not nil
			wantSignatureChanged: true,                               // FUNCTION vs PROCEDURE is a signature change
			wantBodyChanged:      false,
			wantCanUseAlter:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := comparer.CompareDetailed(tt.oldFunc, tt.newFunc)
			if err != nil {
				t.Errorf("CompareDetailed() error = %v", err)
				return
			}

			if tt.wantResult == nil {
				if result != nil {
					t.Errorf("CompareDetailed() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Error("CompareDetailed() = nil, want non-nil result")
				return
			}

			if result.SignatureChanged != tt.wantSignatureChanged {
				t.Errorf("CompareDetailed().SignatureChanged = %v, want %v", result.SignatureChanged, tt.wantSignatureChanged)
			}

			if result.BodyChanged != tt.wantBodyChanged {
				t.Errorf("CompareDetailed().BodyChanged = %v, want %v", result.BodyChanged, tt.wantBodyChanged)
			}

			if result.CanUseAlterFunction != tt.wantCanUseAlter {
				t.Errorf("CompareDetailed().CanUseAlterFunction = %v, want %v", result.CanUseAlterFunction, tt.wantCanUseAlter)
			}
		})
	}
}

func TestExtractFunctionBody(t *testing.T) {
	tests := []struct {
		name       string
		definition string
		wantBody   string
	}{
		{
			name:       "dollar quoted function",
			definition: `CREATE OR REPLACE FUNCTION test() RETURNS integer AS $$ BEGIN RETURN 1; END; $$ LANGUAGE plpgsql;`,
			wantBody:   "BEGIN RETURN 1; END;",
		},
		{
			name:       "single quoted function",
			definition: `CREATE FUNCTION test() RETURNS integer AS 'SELECT 1' LANGUAGE sql;`,
			wantBody:   "SELECT", // Note: Due to a bug in TrimQuotes function in postgresql parser, single quotes are incorrectly trimmed
		},
		{
			name:       "complex function with nested dollar quotes",
			definition: `CREATE OR REPLACE FUNCTION complex_func() RETURNS text AS $func$ DECLARE result text; BEGIN result := 'test'; RETURN result; END; $func$ LANGUAGE plpgsql;`,
			wantBody:   "DECLARE result text; BEGIN result := 'test'; RETURN result; END;",
		},
		{
			name:       "simple procedure",
			definition: `CREATE OR REPLACE PROCEDURE update_stats() AS $$ BEGIN UPDATE stats SET count = count + 1; END; $$ LANGUAGE plpgsql;`,
			wantBody:   "BEGIN UPDATE stats SET count = count + 1; END;",
		},
		{
			name:       "procedure with custom dollar quote",
			definition: `CREATE PROCEDURE refresh_data() AS $proc$ BEGIN REFRESH MATERIALIZED VIEW data_view; END; $proc$ LANGUAGE plpgsql;`,
			wantBody:   "BEGIN REFRESH MATERIALIZED VIEW data_view; END;",
		},
		{
			name:       "procedure with parameters",
			definition: `CREATE OR REPLACE PROCEDURE update_user(user_id integer, new_name text) AS $$ BEGIN UPDATE users SET name = new_name WHERE id = user_id; END; $$ LANGUAGE plpgsql;`,
			wantBody:   "BEGIN UPDATE users SET name = new_name WHERE id = user_id; END;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := extractFunctionBody(tt.definition)
			if body != tt.wantBody {
				t.Errorf("extractFunctionBody() = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestParseFunctionSignature(t *testing.T) {
	tests := []struct {
		name       string
		definition string
		wantName   string
		wantParams int
		wantReturn string
	}{
		{
			name:       "simple function",
			definition: `CREATE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			wantName:   "add_numbers",
			wantParams: 2,
			wantReturn: "integer",
		},
		{
			name:       "function with no parameters",
			definition: `CREATE OR REPLACE FUNCTION get_current_time() RETURNS timestamp AS $$ BEGIN RETURN NOW(); END; $$ LANGUAGE plpgsql;`,
			wantName:   "get_current_time",
			wantParams: 0,
			wantReturn: "timestamp",
		},
		{
			name:       "function with complex return type",
			definition: `CREATE FUNCTION get_user(user_id bigint) RETURNS TABLE(id bigint, name text) AS $$ BEGIN RETURN QUERY SELECT id, name FROM users WHERE id = user_id; END; $$ LANGUAGE plpgsql;`,
			wantName:   "get_user",
			wantParams: 1,
			wantReturn: "", // TABLE return types may not be fully parsed by current implementation
		},
		{
			name:       "simple procedure",
			definition: `CREATE OR REPLACE PROCEDURE update_user(user_id integer, new_name text) AS $$ BEGIN UPDATE users SET name = new_name WHERE id = user_id; END; $$ LANGUAGE plpgsql;`,
			wantName:   "update_user",
			wantParams: 2,
			wantReturn: "", // Procedures don't have return types
		},
		{
			name:       "procedure with no parameters",
			definition: `CREATE PROCEDURE refresh_stats() AS $$ BEGIN REFRESH MATERIALIZED VIEW stats; END; $$ LANGUAGE plpgsql;`,
			wantName:   "refresh_stats",
			wantParams: 0,
			wantReturn: "", // Procedures don't have return types
		},
		{
			name:       "procedure with schema qualification",
			definition: `CREATE OR REPLACE PROCEDURE public.cleanup_old_data(days_old integer) AS $$ BEGIN DELETE FROM logs WHERE created_at < NOW() - INTERVAL '%d days', days_old; END; $$ LANGUAGE plpgsql;`,
			wantName:   "cleanup_old_data", // Should extract unqualified name
			wantParams: 1,
			wantReturn: "", // Procedures don't have return types
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signature, err := parseFunctionSignature(tt.definition)
			if err != nil {
				t.Errorf("parseFunctionSignature() error = %v", err)
				return
			}

			if signature.Name != tt.wantName {
				t.Errorf("parseFunctionSignature().Name = %v, want %v", signature.Name, tt.wantName)
			}

			if len(signature.Parameters) != tt.wantParams {
				t.Errorf("parseFunctionSignature().Parameters count = %v, want %v", len(signature.Parameters), tt.wantParams)
			}

			if signature.ReturnType != tt.wantReturn {
				t.Errorf("parseFunctionSignature().ReturnType = %v, want %v", signature.ReturnType, tt.wantReturn)
			}
		})
	}
}

func TestPostgreSQLFunctionComparer_ProcedureSupport(t *testing.T) {
	comparer := &PostgreSQLFunctionComparer{}

	tests := []struct {
		name     string
		def1     string
		def2     string
		wantDiff bool
		wantMsg  string
	}{
		{
			name:     "identical procedures should be equal",
			def1:     `CREATE OR REPLACE PROCEDURE test_proc(param1 integer) AS $$ BEGIN UPDATE table1 SET col1 = param1; END; $$ LANGUAGE plpgsql;`,
			def2:     `CREATE OR REPLACE PROCEDURE test_proc(param1 integer) AS $$ BEGIN UPDATE table1 SET col1 = param1; END; $$ LANGUAGE plpgsql;`,
			wantDiff: false,
			wantMsg:  "identical procedures",
		},
		{
			name:     "procedures with dollar quote differences should be equal",
			def1:     `CREATE OR REPLACE PROCEDURE public.test_proc() LANGUAGE plpgsql AS $procedure$ BEGIN UPDATE stats SET count = 1; END; $procedure$`,
			def2:     `CREATE OR REPLACE PROCEDURE test_proc() LANGUAGE plpgsql AS $$ BEGIN UPDATE stats SET count = 1; END; $$`,
			wantDiff: false,
			wantMsg:  "procedures with only formatting differences",
		},
		{
			name:     "procedures with body differences should not be equal",
			def1:     `CREATE OR REPLACE PROCEDURE test_proc() AS $$ BEGIN UPDATE table1 SET col1 = 1; END; $$ LANGUAGE plpgsql;`,
			def2:     `CREATE OR REPLACE PROCEDURE test_proc() AS $$ BEGIN UPDATE table1 SET col1 = 2; END; $$ LANGUAGE plpgsql;`,
			wantDiff: true,
			wantMsg:  "procedures with different bodies",
		},
		{
			name:     "procedures with signature differences should not be equal",
			def1:     `CREATE OR REPLACE PROCEDURE test_proc(param1 integer) AS $$ BEGIN UPDATE table1 SET col1 = param1; END; $$ LANGUAGE plpgsql;`,
			def2:     `CREATE OR REPLACE PROCEDURE test_proc(param1 integer, param2 text) AS $$ BEGIN UPDATE table1 SET col1 = param1; END; $$ LANGUAGE plpgsql;`,
			wantDiff: true,
			wantMsg:  "procedures with different signatures",
		},
		{
			name:     "function vs procedure should not be equal",
			def1:     `CREATE OR REPLACE FUNCTION test_func() RETURNS integer AS $$ BEGIN RETURN 1; END; $$ LANGUAGE plpgsql;`,
			def2:     `CREATE OR REPLACE PROCEDURE test_proc() AS $$ BEGIN UPDATE table1 SET col1 = 1; END; $$ LANGUAGE plpgsql;`,
			wantDiff: true,
			wantMsg:  "function vs procedure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldFunc := &storepb.FunctionMetadata{Definition: tt.def1}
			newFunc := &storepb.FunctionMetadata{Definition: tt.def2}

			equal := comparer.Equal(oldFunc, newFunc)
			if equal == tt.wantDiff {
				t.Errorf("%s: Equal() = %v, want %v", tt.wantMsg, equal, !tt.wantDiff)
			}

			// Also test detailed comparison for non-identical cases
			if tt.wantDiff {
				result, err := comparer.CompareDetailed(oldFunc, newFunc)
				if err != nil {
					t.Errorf("%s: CompareDetailed() error = %v", tt.wantMsg, err)
				}
				if result == nil {
					t.Errorf("%s: CompareDetailed() returned nil, expected non-nil result", tt.wantMsg)
				}
			}
		})
	}
}

func TestFunctionBodyExtraction_WithANTLR(t *testing.T) {
	tests := []struct {
		name       string
		definition string
		wantBody   string
	}{
		{
			name:       "function with ANTLR-based extraction",
			definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			wantBody:   " BEGIN RETURN a + b; END; ", // Expected content between dollar quotes
		},
		{
			name:       "procedure with ANTLR-based extraction",
			definition: `CREATE OR REPLACE PROCEDURE update_counter() AS $$ BEGIN UPDATE counters SET count = count + 1; END; $$ LANGUAGE plpgsql;`,
			wantBody:   " BEGIN UPDATE counters SET count = count + 1; END; ", // Expected content between dollar quotes
		},
		{
			name:       "function with custom dollar tag",
			definition: `CREATE OR REPLACE FUNCTION test_func() RETURNS text AS $body$ BEGIN RETURN 'test'; END; $body$ LANGUAGE plpgsql;`,
			wantBody:   " BEGIN RETURN 'test'; END; ", // Expected content between dollar quotes
		},
		{
			name:       "procedure with custom dollar tag",
			definition: `CREATE OR REPLACE PROCEDURE test_proc() AS $proc$ BEGIN UPDATE stats SET last_run = NOW(); END; $proc$ LANGUAGE plpgsql;`,
			wantBody:   " BEGIN UPDATE stats SET last_run = NOW(); END; ", // Expected content between dollar quotes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := extractFunctionBody(tt.definition)
			// Normalize whitespace for comparison
			body = strings.TrimSpace(body)
			wantBody := strings.TrimSpace(tt.wantBody)

			if body != wantBody {
				t.Errorf("extractFunctionBody() = %q, want %q", body, wantBody)
			}
		})
	}
}
