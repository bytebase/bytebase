package mysql

// Unit tests for the multi-file SDL file-name sanitization + de-duplication. These need no
// live database — they pin the safe-file-name behavior for MySQL identifiers that contain
// path-unsafe or ambiguous characters.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeFileStem(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "users", want: "users"},
		{name: "with_underscore_preserved", in: "user_roles", want: "user_roles"},
		{name: "double_underscore_preserved", in: "a__b", want: "a__b"},
		{name: "slash_replaced", in: "a/b", want: "a_b"},
		{name: "backslash_replaced", in: `a\b`, want: "a_b"},
		{name: "backtick_replaced", in: "a`b", want: "a_b"},
		{name: "space_replaced", in: "my table", want: "my_table"},
		{name: "runs_collapsed", in: "a  b", want: "a_b"},
		{name: "dot_replaced", in: "a.b", want: "a_b"},
		{name: "leading_dot_trimmed", in: ".hidden", want: "hidden"},
		{name: "trailing_space_trimmed", in: "trail ", want: "trail"},
		{name: "unicode_replaced", in: "café", want: "caf"},
		{name: "all_unsafe_falls_back", in: "///", want: "_"},
		{name: "empty_falls_back", in: "", want: "_"},
		{name: "digits_and_dash", in: "v1-2", want: "v1-2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, sanitizeFileStem(tc.in))
		})
	}
}

func TestFileNameAllocatorDedup(t *testing.T) {
	a := newFileNameAllocator()

	// Distinct sanitized-equal names must get distinct paths within a dir.
	require.Equal(t, "tables/a_b.sql", a.alloc("tables", "a/b"))
	require.Equal(t, "tables/a_b_1.sql", a.alloc("tables", "a.b"))
	require.Equal(t, "tables/a_b_2.sql", a.alloc("tables", "a b"))

	// Case-only collisions collide case-insensitively (file systems may be case-insensitive).
	require.Equal(t, "tables/Foo.sql", a.alloc("tables", "Foo"))
	require.Equal(t, "tables/foo_1.sql", a.alloc("tables", "foo"))

	// Same stem in a DIFFERENT directory does not collide.
	require.Equal(t, "views/a_b.sql", a.alloc("views", "a/b"))
}

// TestFileNameAllocatorReservesGeneratedSuffix pins the fix for a real object whose
// sanitized name equals a suffix the allocator would generate. With `Foo`, `foo`, `foo_1`
// (lower_case_table_names=0, so all three are distinct MySQL objects), the naive allocator
// hands `foo` the generated stem `foo_1` AND later hands the real `foo_1` the same `foo_1`
// — two schema.File entries with identical paths, so on zip extraction one silently
// overwrites the other and an object's DDL vanishes from the exported SDL. Every returned
// stem must be reserved so all three paths stay DISTINCT.
func TestFileNameAllocatorReservesGeneratedSuffix(t *testing.T) {
	a := newFileNameAllocator()

	got := []string{
		a.alloc("tables", "Foo"),
		a.alloc("tables", "foo"),
		a.alloc("tables", "foo_1"),
	}
	require.Equal(t, []string{"tables/Foo.sql", "tables/foo_1.sql", "tables/foo_1_1.sql"}, got)

	// No path is handed out twice — the property that prevents silent overwrite.
	seen := map[string]bool{}
	for _, p := range got {
		require.False(t, seen[p], "duplicate path %q", p)
		seen[p] = true
	}

	// A real object literally named like an already-generated suffix also stays distinct.
	b := newFileNameAllocator()
	require.Equal(t, "tables/t.sql", b.alloc("tables", "t"))
	require.Equal(t, "tables/t_1.sql", b.alloc("tables", "t/"))    // sanitizes to "t", bumped to t_1
	require.Equal(t, "tables/t_1_1.sql", b.alloc("tables", "t_1")) // real "t_1" cannot reuse the generated t_1
}
