package pg

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestCheckSDLStyle(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		wantCount int
		wantCodes []code.Code
	}{
		{
			name: "Valid SDL - table with schema name, table-level named constraints",
			statement: `CREATE TABLE public.users (
				id INTEGER,
				email TEXT,
				CONSTRAINT pk_users PRIMARY KEY (id),
				CONSTRAINT uk_users_email UNIQUE (email)
			);`,
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name: "Valid SDL - table with allowed column-level constraints (NOT NULL, DEFAULT)",
			statement: `CREATE TABLE public.users (
				id INTEGER NOT NULL DEFAULT 0,
				email TEXT NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				CONSTRAINT pk_users PRIMARY KEY (id)
			);`,
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name: "Valid SDL - table with GENERATED column constraint",
			statement: `CREATE TABLE public.users (
				id INTEGER NOT NULL,
				email TEXT NOT NULL,
				email_lower TEXT GENERATED ALWAYS AS (LOWER(email)) STORED,
				CONSTRAINT pk_users PRIMARY KEY (id)
			);`,
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name: "Error - Missing schema name for table",
			statement: `CREATE TABLE users (
				id INTEGER,
				CONSTRAINT pk_users PRIMARY KEY (id)
			);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLRequireSchemaName},
		},
		{
			name: "Error - Column-level PRIMARY KEY constraint",
			statement: `CREATE TABLE public.users (
				id INTEGER PRIMARY KEY,
				email TEXT
			);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLDisallowColumnConstraint},
		},
		{
			name: "Error - Column-level UNIQUE constraint",
			statement: `CREATE TABLE public.users (
				id INTEGER,
				email TEXT UNIQUE
			);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLDisallowColumnConstraint},
		},
		{
			name: "Error - Column-level CHECK constraint",
			statement: `CREATE TABLE public.users (
				id INTEGER CHECK (id > 0),
				email TEXT
			);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLDisallowColumnConstraint},
		},
		{
			name: "Error - Column-level FOREIGN KEY constraint",
			statement: `CREATE TABLE public.orders (
				id INTEGER,
				user_id INTEGER REFERENCES public.users(id)
			);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLDisallowColumnConstraint},
		},
		{
			name: "Error - Table-level constraint without name",
			statement: `CREATE TABLE public.users (
				id INTEGER,
				PRIMARY KEY (id)
			);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLRequireConstraintName},
		},
		{
			name: "Error - Table-level UNIQUE without name",
			statement: `CREATE TABLE public.users (
				id INTEGER,
				email TEXT,
				UNIQUE (email)
			);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLRequireConstraintName},
		},
		{
			name: "Error - Table-level CHECK without name",
			statement: `CREATE TABLE public.users (
				id INTEGER,
				CHECK (id > 0)
			);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLRequireConstraintName},
		},
		{
			name: "Error - Table-level FOREIGN KEY without name",
			statement: `CREATE TABLE public.users (
				id INTEGER,
				CONSTRAINT pk_users PRIMARY KEY (id)
			);
			CREATE TABLE public.orders (
				id INTEGER,
				user_id INTEGER,
				FOREIGN KEY (user_id) REFERENCES public.users(id)
			);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLRequireConstraintName},
		},
		{
			name: "Error - Multiple disallowed column-level constraints on one column",
			statement: `CREATE TABLE public.users (
				id INTEGER PRIMARY KEY UNIQUE CHECK (id > 0),
				email TEXT
			);`,
			wantCount: 1, // One column with multiple disallowed constraints
			wantCodes: []code.Code{
				code.SDLDisallowColumnConstraint,
			},
		},
		{
			name: "Error - Multiple columns with disallowed constraints",
			statement: `CREATE TABLE public.users (
				id INTEGER PRIMARY KEY,
				email TEXT UNIQUE
			);`,
			wantCount: 2, // Two columns with disallowed constraints
			wantCodes: []code.Code{
				code.SDLDisallowColumnConstraint,
				code.SDLDisallowColumnConstraint,
			},
		},
		{
			name: "Valid SDL - Mix of allowed column constraints with table-level constraints",
			statement: `CREATE TABLE public.users (
				id INTEGER NOT NULL DEFAULT 0,
				email TEXT NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				CONSTRAINT pk_users PRIMARY KEY (id),
				CONSTRAINT uk_users_email UNIQUE (email)
			);`,
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name: "Error - Missing schema name and unnamed constraint",
			statement: `CREATE TABLE users (
				id INTEGER,
				PRIMARY KEY (id)
			);`,
			wantCount: 2,
			wantCodes: []code.Code{
				code.SDLRequireSchemaName,
				code.SDLRequireConstraintName,
			},
		},
		{
			name: "Error - All three types of errors",
			statement: `CREATE TABLE users (
				id INTEGER PRIMARY KEY,
				UNIQUE (id)
			);`,
			wantCount: 3,
			wantCodes: []code.Code{
				code.SDLRequireSchemaName,
				code.SDLDisallowColumnConstraint,
				code.SDLRequireConstraintName,
			},
		},
		{
			name:      "Valid SDL - CREATE INDEX with schema name",
			statement: `CREATE INDEX idx_users_email ON public.users (email);`,
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name:      "Error - CREATE INDEX without schema name",
			statement: `CREATE INDEX idx_users_email ON users (email);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLRequireSchemaName},
		},
		{
			name:      "Error - CREATE INDEX without schema name (quoted table)",
			statement: `CREATE INDEX "idx_departments_budget" ON ONLY "departments" (budget);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLRequireSchemaName},
		},
		{
			name:      "Valid SDL - CREATE UNIQUE INDEX with schema name",
			statement: `CREATE UNIQUE INDEX idx_users_email ON public.users (email);`,
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name:      "Error - CREATE UNIQUE INDEX without schema name",
			statement: `CREATE UNIQUE INDEX idx_users_email ON users (email);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLRequireSchemaName},
		},
		{
			name:      "Error - CREATE INDEX (unnamed) without schema name",
			statement: `CREATE INDEX ON users (email);`,
			wantCount: 2, // Both: missing index name AND missing schema name
			wantCodes: []code.Code{
				code.SDLRequireIndexName,
				code.SDLRequireSchemaName,
			},
		},
		{
			name:      "Error - CREATE INDEX (unnamed) with schema name",
			statement: `CREATE INDEX ON public.users (email);`,
			wantCount: 1, // Only missing index name
			wantCodes: []code.Code{code.SDLRequireIndexName},
		},
		{
			name: "Valid SDL - CREATE VIEW with schema name",
			statement: `CREATE TABLE public.users (id INTEGER, active BOOLEAN, CONSTRAINT pk_users PRIMARY KEY (id));
CREATE VIEW public.active_users AS SELECT * FROM public.users WHERE active = true;`,
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name: "Error - CREATE VIEW without schema name",
			statement: `CREATE TABLE public.users (id INTEGER, active BOOLEAN, CONSTRAINT pk_users PRIMARY KEY (id));
CREATE VIEW active_users AS SELECT * FROM public.users WHERE active = true;`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLRequireSchemaName},
		},
		{
			name:      "Valid SDL - CREATE SEQUENCE with schema name",
			statement: `CREATE SEQUENCE public.user_id_seq START 1;`,
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name:      "Error - CREATE SEQUENCE without schema name",
			statement: `CREATE SEQUENCE user_id_seq START 1;`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLRequireSchemaName},
		},
		{
			name: "Valid SDL - CREATE FUNCTION with schema name",
			statement: `CREATE FUNCTION public.get_user_count() RETURNS INTEGER AS $$
				SELECT COUNT(*) FROM public.users;
			$$ LANGUAGE SQL;`,
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name: "Error - CREATE FUNCTION without schema name",
			statement: `CREATE FUNCTION get_user_count() RETURNS INTEGER AS $$
				SELECT COUNT(*) FROM public.users;
			$$ LANGUAGE SQL;`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLRequireSchemaName},
		},
		{
			name:      "Valid SDL - ALTER SEQUENCE with schema name",
			statement: `ALTER SEQUENCE public.user_id_seq OWNED BY public.users.id;`,
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name:      "Error - ALTER SEQUENCE without schema name",
			statement: `ALTER SEQUENCE user_id_seq OWNED BY public.users.id;`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLRequireSchemaName},
		},
		{
			name:      "Valid SDL - ALTER SEQUENCE with other options",
			statement: `ALTER SEQUENCE public.user_id_seq INCREMENT BY 5 MINVALUE 1 MAXVALUE 1000;`,
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name: "Valid SDL - Multiple statements all valid",
			statement: `CREATE TABLE public.users (
				id INTEGER,
				email TEXT,
				CONSTRAINT pk_users PRIMARY KEY (id)
			);

			CREATE INDEX idx_users_email ON public.users (email);

			CREATE SEQUENCE public.user_id_seq START 1;`,
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name: "Error - Multiple statements with multiple errors",
			statement: `CREATE TABLE users (
				id INTEGER PRIMARY KEY
			);

			CREATE INDEX idx_users_email ON users (email);

			CREATE VIEW active_users AS SELECT * FROM users;`,
			wantCount: 4, // schema name for table + column constraint + schema name for index + schema name for view
			wantCodes: []code.Code{
				code.SDLRequireSchemaName,        // table
				code.SDLDisallowColumnConstraint, // column constraint
				code.SDLRequireSchemaName,        // index
				code.SDLRequireSchemaName,        // view
			},
		},
		{
			name: "Valid SDL - Complex table with multiple named constraints",
			statement: `CREATE TABLE public.users (
				id INTEGER,
				CONSTRAINT pk_users PRIMARY KEY (id)
			);
			CREATE TABLE public.products (
				id INTEGER,
				CONSTRAINT pk_products PRIMARY KEY (id)
			);
			CREATE TABLE public.orders (
				id INTEGER,
				user_id INTEGER,
				product_id INTEGER,
				quantity INTEGER,
				total_price NUMERIC,
				CONSTRAINT pk_orders PRIMARY KEY (id),
				CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES public.users(id),
				CONSTRAINT fk_orders_product FOREIGN KEY (product_id) REFERENCES public.products(id),
				CONSTRAINT uk_orders_composite UNIQUE (user_id, product_id),
				CONSTRAINT chk_orders_quantity CHECK (quantity > 0),
				CONSTRAINT chk_orders_price CHECK (total_price >= 0)
			);`,
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name: "Error - FK reference without schema name (table-level)",
			statement: `CREATE TABLE public.users (
				id INTEGER,
				CONSTRAINT pk_users PRIMARY KEY (id)
			);
			CREATE TABLE public.orders (
				id INTEGER,
				user_id INTEGER,
				CONSTRAINT pk_orders PRIMARY KEY (id),
				CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users(id)
			);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLRequireSchemaName},
		},
		{
			name: "Error - FK reference without schema name (column-level)",
			statement: `CREATE TABLE public.orders (
				id INTEGER,
				user_id INTEGER REFERENCES users(id)
			);`,
			wantCount: 2, // Both: disallowed column constraint AND missing schema
			wantCodes: []code.Code{
				code.SDLRequireSchemaName,
				code.SDLDisallowColumnConstraint,
			},
		},
		{
			name: "Error - Multiple FK references without schema",
			statement: `CREATE TABLE public.users (
				id INTEGER,
				CONSTRAINT pk_users PRIMARY KEY (id)
			);
			CREATE TABLE public.products (
				id INTEGER,
				CONSTRAINT pk_products PRIMARY KEY (id)
			);
			CREATE TABLE public.orders (
				id INTEGER,
				user_id INTEGER,
				product_id INTEGER,
				CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users(id),
				CONSTRAINT fk_orders_product FOREIGN KEY (product_id) REFERENCES products(id)
			);`,
			wantCount: 2, // Two FK references without schema
			wantCodes: []code.Code{
				code.SDLRequireSchemaName,
				code.SDLRequireSchemaName,
			},
		},
		{
			name: "Valid SDL - Mix of FK with and without errors",
			statement: `CREATE TABLE public.users (
				id INTEGER,
				CONSTRAINT pk_users PRIMARY KEY (id)
			);
			CREATE TABLE public.products (
				id INTEGER,
				CONSTRAINT pk_products PRIMARY KEY (id)
			);
			CREATE TABLE public.orders (
				id INTEGER,
				user_id INTEGER,
				product_id INTEGER,
				CONSTRAINT pk_orders PRIMARY KEY (id),
				CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES public.users(id),
				CONSTRAINT fk_orders_product FOREIGN KEY (product_id) REFERENCES public.products(id)
			);`,
			wantCount: 0,
			wantCodes: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			advices, err := CheckSDLStyle(tc.statement)
			require.NoError(t, err, "CheckSDLStyle should not return error for valid SQL")
			require.Equal(t, tc.wantCount, len(advices), "Unexpected number of advices")

			if tc.wantCount > 0 {
				require.NotNil(t, tc.wantCodes, "wantCodes should be specified when wantCount > 0")
				require.Equal(t, len(tc.wantCodes), len(advices), "Number of advices should match wantCodes")

				// Verify each advice has the expected error code and ERROR status
				for i, advice := range advices {
					require.Equal(t, storepb.Advice_ERROR, advice.Status, "All SDL check advices should have ERROR status")
					require.Equal(t, tc.wantCodes[i].Int32(), advice.Code, "Advice %d should have expected error code", i)
					require.NotEmpty(t, advice.Title, "Advice should have a title")
					require.NotEmpty(t, advice.Content, "Advice should have content")
					require.NotNil(t, advice.StartPosition, "Advice should have start position")
					require.Greater(t, advice.StartPosition.Line, int32(0), "Line number should be positive")
				}
			}
		})
	}
}

func TestCheckSDLStyle_InvalidSQL(t *testing.T) {
	tests := []struct {
		name      string
		statement string
	}{
		{
			name:      "Syntax error - missing parenthesis",
			statement: `CREATE TABLE public.users (id INTEGER;`,
		},
		{
			name:      "Syntax error - invalid keyword",
			statement: `CREATE TABLEX public.users (id INTEGER);`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CheckSDLStyle(tc.statement)
			require.Error(t, err, "CheckSDLStyle should return error for invalid SQL")
		})
	}
}

func TestCheckSDLStyle_EmptyStatement(t *testing.T) {
	advices, err := CheckSDLStyle("")
	require.NoError(t, err)
	require.Empty(t, advices, "Empty statement should return no advices")
}

func TestCheckSDLStyle_CommentOnly(t *testing.T) {
	advices, err := CheckSDLStyle("-- This is just a comment")
	require.NoError(t, err)
	require.Empty(t, advices, "Comment-only statement should return no advices")
}
