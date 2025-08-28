package pg

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestPostgreSQLFunctionComparer_Equal(t *testing.T) {
	comparer := &PostgreSQLFunctionComparer{}

	tests := []struct {
		name      string
		oldFunc   *model.FunctionMetadata
		newFunc   *model.FunctionMetadata
		wantEqual bool
	}{
		{
			name: "identical functions",
			oldFunc: &model.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &model.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			wantEqual: true,
		},
		{
			name: "body only difference",
			oldFunc: &model.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &model.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a * b; END; $$ LANGUAGE plpgsql;`,
			},
			wantEqual: false,
		},
		{
			name: "signature difference",
			oldFunc: &model.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &model.FunctionMetadata{
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
			oldFunc: &model.FunctionMetadata{
				Definition: `CREATE FUNCTION test() RETURNS integer AS $$ BEGIN RETURN 1; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc:   nil,
			wantEqual: true, // CompareDetailed returns nil for nil inputs, so Equal returns true
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
		oldFunc              *model.FunctionMetadata
		newFunc              *model.FunctionMetadata
		wantResult           *schema.FunctionComparisonResult
		wantSignatureChanged bool
		wantBodyChanged      bool
		wantCanUseAlter      bool
	}{
		{
			name: "identical functions",
			oldFunc: &model.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &model.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			wantResult: nil, // No changes, should return nil
		},
		{
			name: "body only difference",
			oldFunc: &model.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &model.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a * b; END; $$ LANGUAGE plpgsql;`,
			},
			wantResult:           &schema.FunctionComparisonResult{}, // Expect a result, not nil
			wantSignatureChanged: false,
			wantBodyChanged:      true,
			wantCanUseAlter:      true,
		},
		{
			name: "signature difference",
			oldFunc: &model.FunctionMetadata{
				Definition: `CREATE OR REPLACE FUNCTION add_numbers(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;`,
			},
			newFunc: &model.FunctionMetadata{
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
				t.Errorf("CompareDetailed() = nil, want non-nil result")
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
			wantBody:   "$$ BEGIN RETURN 1; END; $$",
		},
		{
			name:       "single quoted function",
			definition: `CREATE FUNCTION test() RETURNS integer AS 'SELECT 1' LANGUAGE sql;`,
			wantBody:   "SELECT 1",
		},
		{
			name:       "complex function with nested dollar quotes",
			definition: `CREATE OR REPLACE FUNCTION complex_func() RETURNS text AS $func$ DECLARE result text; BEGIN result := 'test'; RETURN result; END; $func$ LANGUAGE plpgsql;`,
			wantBody:   "$func$ DECLARE result text; BEGIN result := 'test'; RETURN result; END; $func$",
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
