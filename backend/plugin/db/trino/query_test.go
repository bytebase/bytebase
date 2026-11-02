package trino

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trinodb/trino-go-client/trino"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestMakeValueByTypeNameHoldsAVariant(t *testing.T) {
	destination := makeValueByTypeName("VARIANT", nil)
	variant, ok := destination.(*trino.Variant)
	require.True(t, ok, "a VARIANT column needs a *trino.Variant destination, got %T", destination)

	// rows.Scan hands a VARIANT destination the trino.Variant the driver built,
	// which is why the default destination below cannot serve one.
	require.NoError(t, variant.Scan(trino.Variant{}))
	require.Error(t, new(sql.NullString).Scan(trino.Variant{}))
}

func TestConvertValueReadsAVariant(t *testing.T) {
	// A SQL NULL reaches Scan as nil and leaves the Variant invalid.
	//
	// This pins the NULL path only. convertValue's *trino.Variant branch is not
	// covered: a populated Variant cannot be built outside the driver, since its
	// fields are unexported, the only constructor decodes a server response, and
	// a hand-made Variant{Valid: true} panics in String. Removing that branch
	// still passes this test, so a VARIANT value rendering correctly needs a
	// Trino server to prove.
	variant := new(trino.Variant)
	require.NoError(t, variant.Scan(nil))
	require.False(t, variant.Valid)
	require.Equal(t, util.NullRowValue, convertValue("VARIANT", nil, variant))
}
