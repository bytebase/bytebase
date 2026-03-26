package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniFunctionSDLDiffAndMigration(t *testing.T) {
	tests := []struct {
		name     string
		fromSDL  string
		toSDL    string
		contains []string
	}{
		{
			name:    "Create new function",
			fromSDL: ``,
			toSDL: `
				CREATE FUNCTION calculate_area(radius DECIMAL) RETURNS DECIMAL AS $$
				BEGIN
					RETURN 3.14159 * radius * radius;
				END;
				$$ LANGUAGE plpgsql;
			`,
			contains: []string{"CREATE FUNCTION", "calculate_area"},
		},
		{
			name: "Drop function",
			fromSDL: `
				CREATE FUNCTION calculate_area(radius DECIMAL) RETURNS DECIMAL AS $$
				BEGIN
					RETURN 3.14159 * radius * radius;
				END;
				$$ LANGUAGE plpgsql;
			`,
			toSDL:    ``,
			contains: []string{"DROP FUNCTION", "calculate_area"},
		},
		{
			name: "Modify function (CREATE OR REPLACE)",
			fromSDL: `
				CREATE FUNCTION calculate_area(radius DECIMAL) RETURNS DECIMAL AS $$
				BEGIN
					RETURN 3.14159 * radius * radius;
				END;
				$$ LANGUAGE plpgsql;
			`,
			toSDL: `
				CREATE FUNCTION calculate_area(radius DECIMAL) RETURNS DECIMAL AS $$
				BEGIN
					RETURN 3.14159265 * radius * radius;
				END;
				$$ LANGUAGE plpgsql;
			`,
			contains: []string{"CREATE OR REPLACE FUNCTION", "calculate_area"},
		},
		{
			name:    "Create new procedure",
			fromSDL: ``,
			toSDL: `
				CREATE PROCEDURE log_message(msg TEXT)
				LANGUAGE plpgsql
				AS $$
				BEGIN
					NULL;
				END;
				$$;
			`,
			contains: []string{"CREATE PROCEDURE", "log_message"},
		},
		{
			name: "Overloaded functions with different signatures",
			fromSDL: `
				CREATE FUNCTION add_numbers(a INTEGER, b INTEGER)
				RETURNS INTEGER AS $$
				BEGIN
					RETURN a + b;
				END;
				$$ LANGUAGE plpgsql;
			`,
			toSDL: `
				CREATE FUNCTION add_numbers(a INTEGER, b INTEGER)
				RETURNS INTEGER AS $$
				BEGIN
					RETURN a + b;
				END;
				$$ LANGUAGE plpgsql;

				CREATE FUNCTION add_numbers(a FLOAT, b FLOAT)
				RETURNS FLOAT AS $$
				BEGIN
					RETURN a + b;
				END;
				$$ LANGUAGE plpgsql;
			`,
			// Omni normalizes FLOAT to double precision
			contains: []string{"CREATE FUNCTION", "add_numbers", "double precision"},
		},
		{
			name: "Drop one overloaded function",
			fromSDL: `
				CREATE FUNCTION add_numbers(a INTEGER, b INTEGER)
				RETURNS INTEGER AS $$
				BEGIN
					RETURN a + b;
				END;
				$$ LANGUAGE plpgsql;

				CREATE FUNCTION add_numbers(a FLOAT, b FLOAT)
				RETURNS FLOAT AS $$
				BEGIN
					RETURN a + b;
				END;
				$$ LANGUAGE plpgsql;
			`,
			toSDL: `
				CREATE FUNCTION add_numbers(a INTEGER, b INTEGER)
				RETURNS INTEGER AS $$
				BEGIN
					RETURN a + b;
				END;
				$$ LANGUAGE plpgsql;
			`,
			contains: []string{"DROP FUNCTION", "add_numbers"},
		},
		{
			name: "Modify one overloaded function",
			fromSDL: `
				CREATE FUNCTION calculate(x INTEGER)
				RETURNS INTEGER AS $$
				BEGIN
					RETURN x * 2;
				END;
				$$ LANGUAGE plpgsql;

				CREATE FUNCTION calculate(x FLOAT)
				RETURNS FLOAT AS $$
				BEGIN
					RETURN x * 2.0;
				END;
				$$ LANGUAGE plpgsql;
			`,
			toSDL: `
				CREATE FUNCTION calculate(x INTEGER)
				RETURNS INTEGER AS $$
				BEGIN
					RETURN x * 3;
				END;
				$$ LANGUAGE plpgsql;

				CREATE FUNCTION calculate(x FLOAT)
				RETURNS FLOAT AS $$
				BEGIN
					RETURN x * 2.0;
				END;
				$$ LANGUAGE plpgsql;
			`,
			contains: []string{"CREATE OR REPLACE FUNCTION", "calculate"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			for _, s := range tt.contains {
				require.Contains(t, sql, s)
			}
		})
	}
}

func TestOmniFunctionASTOnlyModeValidation(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE FUNCTION get_total(amount DECIMAL) RETURNS DECIMAL AS $$
		BEGIN
			RETURN amount * 1.1;
		END;
		$$ LANGUAGE plpgsql;
	`)
	require.Contains(t, sql, "CREATE FUNCTION")
	require.Contains(t, sql, "get_total")
}

func TestOmniOverloadedFunctionSignatureHandling(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE FUNCTION format_value(val INTEGER)
		RETURNS TEXT AS $$
		BEGIN
			RETURN val::TEXT;
		END;
		$$ LANGUAGE plpgsql;

		CREATE FUNCTION format_value(val DECIMAL, precision_digits INTEGER)
		RETURNS TEXT AS $$
		BEGIN
			RETURN ROUND(val, precision_digits)::TEXT;
		END;
		$$ LANGUAGE plpgsql;
	`)
	require.Contains(t, sql, "format_value")
	functionCount := strings.Count(sql, "format_value")
	require.Equal(t, 2, functionCount, "Should reference format_value exactly twice (one per overload)")
}

func TestOmniProcedureDropMigration(t *testing.T) {
	tests := []struct {
		name     string
		fromSDL  string
		toSDL    string
		contains []string
	}{
		{
			name: "Drop procedure should use DROP PROCEDURE",
			fromSDL: `
				CREATE PROCEDURE update_user_name(IN user_id integer, IN new_name character varying)
				LANGUAGE plpgsql
				AS $$
				BEGIN
					UPDATE users SET name = new_name WHERE id = user_id;
				END;
				$$;
			`,
			toSDL:    ``,
			contains: []string{"DROP PROCEDURE", "update_user_name"},
		},
		{
			name: "Drop function should use DROP FUNCTION",
			fromSDL: `
				CREATE FUNCTION calculate_tax(amount numeric) RETURNS numeric
				LANGUAGE plpgsql
				AS $$
				BEGIN
					RETURN amount * 0.1;
				END;
				$$;
			`,
			toSDL:    ``,
			contains: []string{"DROP FUNCTION", "calculate_tax"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			for _, s := range tt.contains {
				require.Contains(t, sql, s)
			}
		})
	}
}

func TestOmniDropProcedureShouldNotAffectFunctionComment(t *testing.T) {
	tests := []struct {
		name    string
		fromSDL string
		toSDL   string
	}{
		{
			name: "Drop procedure without comment should not affect function comment",
			fromSDL: `
				CREATE OR REPLACE FUNCTION test_function()
				RETURNS void
				LANGUAGE plpgsql
				AS $function$
				BEGIN
					RAISE NOTICE 'Test function executed';
				END
				$function$;

				COMMENT ON FUNCTION "public"."test_function"() IS 'A test function that raises a notice';

				CREATE PROCEDURE new_procedure()
				LANGUAGE plpgsql
				AS $$
				BEGIN
					NULL;
				END;
				$$;
			`,
			toSDL: `
				CREATE OR REPLACE FUNCTION test_function()
				RETURNS void
				LANGUAGE plpgsql
				AS $function$
				BEGIN
					RAISE NOTICE 'Test function executed';
				END
				$function$;

				COMMENT ON FUNCTION "public"."test_function"() IS 'A test function that raises a notice';
			`,
		},
		{
			name: "Drop procedure with comment should not affect function comment",
			fromSDL: `
				CREATE OR REPLACE FUNCTION test_function()
				RETURNS void
				LANGUAGE plpgsql
				AS $function$
				BEGIN
					RAISE NOTICE 'Test function executed';
				END
				$function$;

				COMMENT ON FUNCTION "public"."test_function"() IS 'A test function that raises a notice';

				CREATE PROCEDURE new_procedure()
				LANGUAGE plpgsql
				AS $$
				BEGIN
					NULL;
				END;
				$$;

				COMMENT ON PROCEDURE "public"."new_procedure"() IS 'A test procedure';
			`,
			toSDL: `
				CREATE OR REPLACE FUNCTION test_function()
				RETURNS void
				LANGUAGE plpgsql
				AS $function$
				BEGIN
					RAISE NOTICE 'Test function executed';
				END
				$function$;

				COMMENT ON FUNCTION "public"."test_function"() IS 'A test function that raises a notice';
			`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)

			// Should drop the procedure
			require.Contains(t, sql, "DROP PROCEDURE")
			require.Contains(t, sql, "new_procedure")

			// Should NOT contain any COMMENT statement about test_function
			if strings.Contains(sql, "test_function") && strings.Contains(sql, "COMMENT") {
				require.Fail(t, "Found unexpected COMMENT statement about test_function in migration: "+sql)
			}
		})
	}
}
