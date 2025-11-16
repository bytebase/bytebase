package pg

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestCheckSDLIntegrity(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		wantCount int
		wantCodes []code.Code
	}{
		// ============================================================
		// FOREIGN KEY VALIDATION TESTS
		// ============================================================
		{
			name: "Valid FK - table and column exist with matching types",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	email TEXT NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (id)
);

CREATE TABLE public.orders (
	id BIGINT NOT NULL,
	user_id INTEGER NOT NULL,
	CONSTRAINT pk_orders PRIMARY KEY (id),
	CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES public.users(id)
);`,
			wantCount: 0,
		},
		{
			name: "Error - FK references non-existent table",
			statement: `
CREATE TABLE public.orders (
	id BIGINT NOT NULL,
	user_id INTEGER NOT NULL,
	CONSTRAINT pk_orders PRIMARY KEY (id),
	CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES public.users(id)
);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLForeignKeyTableNotFound},
		},
		{
			name: "Error - FK references non-existent column",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	email TEXT NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (id)
);

CREATE TABLE public.orders (
	id BIGINT NOT NULL,
	user_id INTEGER NOT NULL,
	CONSTRAINT pk_orders PRIMARY KEY (id),
	CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES public.users(user_id)
);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLForeignKeyColumnNotFound},
		},
		{
			name: "Valid FK - Composite FK with matching types",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	tenant_id INTEGER NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (id, tenant_id)
);

CREATE TABLE public.orders (
	id BIGINT NOT NULL,
	user_id INTEGER NOT NULL,
	tenant_id INTEGER NOT NULL,
	CONSTRAINT pk_orders PRIMARY KEY (id),
	CONSTRAINT fk_orders_user FOREIGN KEY (user_id, tenant_id) REFERENCES public.users(id, tenant_id)
);`,
			wantCount: 0,
		},
		{
			name: "Valid FK - VARCHAR types with different lengths are compatible",
			statement: `
CREATE TABLE public.users (
	email VARCHAR(255) NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (email)
);

CREATE TABLE public.sessions (
	id BIGINT NOT NULL,
	user_email VARCHAR(100) NOT NULL,
	CONSTRAINT pk_sessions PRIMARY KEY (id),
	CONSTRAINT fk_sessions_user FOREIGN KEY (user_email) REFERENCES public.users(email)
);`,
			wantCount: 0,
		},
		{
			name: "Valid FK - TEXT and VARCHAR are compatible",
			statement: `
CREATE TABLE public.users (
	email TEXT NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (email)
);

CREATE TABLE public.sessions (
	id BIGINT NOT NULL,
	user_email VARCHAR(255) NOT NULL,
	CONSTRAINT pk_sessions PRIMARY KEY (id),
	CONSTRAINT fk_sessions_user FOREIGN KEY (user_email) REFERENCES public.users(email)
);`,
			wantCount: 0,
		},

		// ============================================================
		// DUPLICATE NAME DETECTION TESTS
		// ============================================================
		{
			name: "Error - Duplicate table name in same schema",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (id)
);

CREATE TABLE public.users (
	user_id BIGINT NOT NULL,
	CONSTRAINT pk_users2 PRIMARY KEY (user_id)
);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLDuplicateTableName},
		},
		{
			name: "Valid - Same table name in different schemas",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (id)
);

CREATE TABLE analytics.users (
	id BIGINT NOT NULL,
	CONSTRAINT pk_analytics_users PRIMARY KEY (id)
);`,
			wantCount: 0,
		},
		{
			name: "Error - Duplicate index name in same schema",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	email TEXT NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (id)
);

CREATE INDEX idx_users_email ON public.users(email);
CREATE INDEX idx_users_email ON public.users(id);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLDuplicateIndexName},
		},
		{
			name: "Error - Duplicate constraint name in same schema",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	email TEXT NOT NULL,
	CONSTRAINT pk_common PRIMARY KEY (id),
	CONSTRAINT uk_users_email UNIQUE (email)
);

CREATE TABLE public.products (
	id INTEGER NOT NULL,
	code TEXT NOT NULL,
	CONSTRAINT pk_common PRIMARY KEY (id),
	CONSTRAINT uk_products_code UNIQUE (code)
);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLDuplicateConstraintName},
		},
		{
			name: "Error - Duplicate column name in same table",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	email TEXT NOT NULL,
	email TEXT NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (id)
);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLDuplicateColumnName},
		},
		{
			name: "Valid - Same column name in different tables",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	email TEXT NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (id)
);

CREATE TABLE public.products (
	id INTEGER NOT NULL,
	name TEXT NOT NULL,
	CONSTRAINT pk_products PRIMARY KEY (id)
);`,
			wantCount: 0,
		},

		// ============================================================
		// MULTIPLE PRIMARY KEY TESTS
		// ============================================================
		{
			name: "Error - Multiple PRIMARY KEY constraints on same table",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	email TEXT NOT NULL,
	CONSTRAINT pk_users_id PRIMARY KEY (id),
	CONSTRAINT pk_users_email PRIMARY KEY (email)
);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLMultiplePrimaryKey},
		},
		{
			name: "Error - Three PRIMARY KEY constraints on same table",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	email TEXT NOT NULL,
	username TEXT NOT NULL,
	CONSTRAINT pk1 PRIMARY KEY (id),
	CONSTRAINT pk2 PRIMARY KEY (email),
	CONSTRAINT pk3 PRIMARY KEY (username)
);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLMultiplePrimaryKey},
		},
		{
			name: "Valid - One PRIMARY KEY with multiple columns (composite PK)",
			statement: `
CREATE TABLE public.user_roles (
	user_id INTEGER NOT NULL,
	role_id INTEGER NOT NULL,
	CONSTRAINT pk_user_roles PRIMARY KEY (user_id, role_id)
);`,
			wantCount: 0,
		},

		// ============================================================
		// VIEW DEPENDENCY TESTS
		// ============================================================
		{
			name: "Valid VIEW - references existing table",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	email TEXT NOT NULL,
	active BOOLEAN NOT NULL DEFAULT true,
	CONSTRAINT pk_users PRIMARY KEY (id)
);

CREATE VIEW public.active_users AS
	SELECT id, email FROM public.users WHERE active = true;`,
			wantCount: 0,
		},
		{
			name: "Error - VIEW references non-existent table",
			statement: `
CREATE VIEW public.active_users AS
	SELECT id, email FROM public.users WHERE active = true;`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLViewDependencyNotFound},
		},
		{
			name: "Error - VIEW references multiple tables, one doesn't exist",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	email TEXT NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (id)
);

CREATE VIEW public.user_orders AS
	SELECT u.id, u.email, o.id as order_id
	FROM public.users u
	JOIN public.orders o ON u.id = o.user_id;`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLViewDependencyNotFound},
		},
		{
			name: "Valid VIEW - references all existing tables",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	email TEXT NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (id)
);

CREATE TABLE public.orders (
	id BIGINT NOT NULL,
	user_id INTEGER NOT NULL,
	CONSTRAINT pk_orders PRIMARY KEY (id),
	CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES public.users(id)
);

CREATE VIEW public.user_orders AS
	SELECT u.id, u.email, o.id as order_id
	FROM public.users u
	JOIN public.orders o ON u.id = o.user_id;`,
			wantCount: 0,
		},

		// ============================================================
		// COMPLEX SCENARIOS - MULTIPLE ERRORS
		// ============================================================
		{
			name: "Multiple errors - FK table not found + type mismatch",
			statement: `
CREATE TABLE public.orders (
	id BIGINT NOT NULL,
	user_id SMALLINT NOT NULL,
	price NUMERIC NOT NULL,
	CONSTRAINT pk_orders PRIMARY KEY (id),
	CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES public.users(id)
);`,
			wantCount: 1,
			wantCodes: []code.Code{
				code.SDLForeignKeyTableNotFound,
			},
		},
		{
			name: "Multiple errors - Duplicate names + multiple PKs",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	email TEXT NOT NULL,
	CONSTRAINT pk_common PRIMARY KEY (id),
	CONSTRAINT pk_common2 PRIMARY KEY (email)
);

CREATE TABLE public.orders (
	id BIGINT NOT NULL,
	user_id INTEGER NOT NULL,
	CONSTRAINT pk_common PRIMARY KEY (id)
);`,
			wantCount: 2,
			wantCodes: []code.Code{
				code.SDLDuplicateConstraintName, // Detected first during Pass 1
				code.SDLMultiplePrimaryKey,      // Detected during Pass 2
			},
		},

		// ============================================================
		// CONSTRAINT NAME SCOPING TESTS
		// ============================================================
		{
			name: "Valid - Same FK name on different tables (table-level scoping)",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (id)
);

CREATE TABLE public.orders (
	id INTEGER NOT NULL,
	user_id INTEGER NOT NULL,
	CONSTRAINT pk_orders PRIMARY KEY (id),
	CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id)
);

CREATE TABLE public.invoices (
	id INTEGER NOT NULL,
	user_id INTEGER NOT NULL,
	CONSTRAINT pk_invoices PRIMARY KEY (id),
	CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id)
);`,
			wantCount: 0, // FK constraints are table-scoped, same name allowed on different tables
		},
		{
			name: "Valid - Same CHECK name on different tables (table-level scoping)",
			statement: `
CREATE TABLE public.products (
	id INTEGER NOT NULL,
	price NUMERIC NOT NULL,
	CONSTRAINT pk_products PRIMARY KEY (id),
	CONSTRAINT chk_positive CHECK (price > 0)
);

CREATE TABLE public.services (
	id INTEGER NOT NULL,
	price NUMERIC NOT NULL,
	CONSTRAINT pk_services PRIMARY KEY (id),
	CONSTRAINT chk_positive CHECK (price > 0)
);`,
			wantCount: 0, // CHECK constraints are table-scoped, same name allowed on different tables
		},
		{
			name: "Error - Duplicate FK name in same table",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (id)
);

CREATE TABLE public.posts (
	id INTEGER NOT NULL,
	author_id INTEGER NOT NULL,
	editor_id INTEGER NOT NULL,
	CONSTRAINT pk_posts PRIMARY KEY (id),
	CONSTRAINT fk_user FOREIGN KEY (author_id) REFERENCES public.users(id),
	CONSTRAINT fk_user FOREIGN KEY (editor_id) REFERENCES public.users(id)
);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLDuplicateConstraintName},
		},
		{
			name: "Error - Duplicate PRIMARY KEY name across tables (schema-level scoping)",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	CONSTRAINT pk_common PRIMARY KEY (id)
);

CREATE TABLE public.orders (
	id INTEGER NOT NULL,
	CONSTRAINT pk_common PRIMARY KEY (id)
);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLDuplicateConstraintName},
		},
		{
			name: "Error - Duplicate UNIQUE name across tables (schema-level scoping)",
			statement: `
CREATE TABLE public.users (
	email TEXT NOT NULL,
	CONSTRAINT uk_email UNIQUE (email)
);

CREATE TABLE public.customers (
	email TEXT NOT NULL,
	CONSTRAINT uk_email UNIQUE (email)
);`,
			wantCount: 1,
			wantCodes: []code.Code{code.SDLDuplicateConstraintName},
		},

		// ============================================================
		// EDGE CASES
		// ============================================================
		{
			name: "Valid - Self-referencing FK (same table)",
			statement: `
CREATE TABLE public.employees (
	id INTEGER NOT NULL,
	manager_id INTEGER,
	CONSTRAINT pk_employees PRIMARY KEY (id),
	CONSTRAINT fk_employees_manager FOREIGN KEY (manager_id) REFERENCES public.employees(id)
);`,
			wantCount: 0,
		},
		{
			name:      "Valid - Empty SDL (no tables)",
			statement: ``,
			wantCount: 0,
		},
		{
			name: "Valid - Comments only",
			statement: `-- This is a comment
-- Another comment`,
			wantCount: 0,
		},
		{
			name: "Valid - Table with no constraints",
			statement: `
CREATE TABLE public.logs (
	id BIGINT,
	message TEXT,
	created_at TIMESTAMP
);`,
			wantCount: 0,
		},
		{
			name: "Error - Unnamed FK constraint (detected by style check, not integrity)",
			statement: `
CREATE TABLE public.users (
	id INTEGER NOT NULL,
	CONSTRAINT pk_users PRIMARY KEY (id)
);

CREATE TABLE public.orders (
	id BIGINT NOT NULL,
	user_id INTEGER NOT NULL,
	CONSTRAINT pk_orders PRIMARY KEY (id),
	FOREIGN KEY (user_id) REFERENCES public.users(id)
);`,
			wantCount: 0, // Integrity check doesn't care about unnamed constraints
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			advices, err := checkSingleStatement(tc.statement)
			require.NoError(t, err, "checkSingleStatement should not return error for valid SQL syntax")
			require.Equal(t, tc.wantCount, len(advices), "Unexpected number of advices. Advices: %+v", advices)

			if tc.wantCount > 0 {
				require.NotNil(t, tc.wantCodes, "wantCodes should be specified when wantCount > 0")
				require.Equal(t, len(tc.wantCodes), len(advices), "Number of advices should match wantCodes")

				// Verify each advice has the expected error code and ERROR status
				for i, advice := range advices {
					require.Equal(t, storepb.Advice_ERROR, advice.Status, "All SDL integrity check advices should have ERROR status")
					require.Equal(t, tc.wantCodes[i].Int32(), advice.Code, "Advice %d should have expected error code. Got: %s", i, advice.Title)
					require.NotEmpty(t, advice.Title, "Advice should have a title")
					require.NotEmpty(t, advice.Content, "Advice should have content")
					require.NotNil(t, advice.StartPosition, "Advice should have start position")
					require.Greater(t, advice.StartPosition.Line, int32(0), "Line number should be positive")
				}
			}
		})
	}
}

func TestCheckSDLIntegrity_InvalidSQL(t *testing.T) {
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
			advices, err := checkSingleStatement(tc.statement)
			// Syntax errors are returned as advices, not as errors
			require.NoError(t, err)
			require.Greater(t, len(advices), 0, "Should have syntax error advice")
			require.Equal(t, code.StatementSyntaxError.Int32(), advices[0].Code)
		})
	}
}
