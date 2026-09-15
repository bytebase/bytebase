package saas

import (
	"bytes"
	"context"
	"io"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sample"
)

func TestRandomPasswordMeetsCloudSQLComplexity(t *testing.T) {
	for _, value := range []byte{0, 0xff, 'x'} {
		password, err := randomPassword(bytes.NewReader(bytes.Repeat([]byte{value}, 32)))
		require.NoError(t, err)
		var lower, upper, digit, symbol bool
		for _, char := range password {
			lower = lower || unicode.IsLower(char)
			upper = upper || unicode.IsUpper(char)
			digit = digit || unicode.IsDigit(char)
			symbol = symbol || (!unicode.IsLetter(char) && !unicode.IsDigit(char))
		}
		require.True(t, lower && upper && digit && symbol, "password must satisfy Cloud SQL complexity for input byte %d", value)
		require.GreaterOrEqual(t, len(password), 43)
	}
}

func TestRandomPasswordRequiresFullEntropy(t *testing.T) {
	password, err := randomPassword(bytes.NewReader(make([]byte, 31)))
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	require.Empty(t, password)
}

func TestCheckAvailableRejectsUnconfiguredManager(t *testing.T) {
	var manager *Manager
	err := manager.CheckAvailable(context.Background())
	require.Error(t, err)
	require.Equal(t, sample.FailureUnavailable, sample.FailureKindOf(err))
}

func TestSampleNamesAreStableAndShort(t *testing.T) {
	database, role := sampleNames("sample-0123456789abcdef")
	require.Equal(t, "bb_sample_d3bc52190e66d2ca", database)
	require.Equal(t, "bb_sample_role_d3bc52190e66d2ca", role)
}
