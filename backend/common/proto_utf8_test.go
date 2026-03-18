//nolint:revive
package common

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestSanitizeProtoStringFields_SimpleString(t *testing.T) {
	msg := &storepb.DatabaseMetadata{
		SyncError: "valid utf8 string",
	}
	SanitizeProtoStringFields(msg)
	assert.Equal(t, "valid utf8 string", msg.GetSyncError())
}

func TestSanitizeProtoStringFields_InvalidUTF8String(t *testing.T) {
	// 0xff and 0xfe are invalid UTF-8 bytes.
	bad := "hello\xff\xfeworld"
	require.False(t, utf8.ValidString(bad))

	msg := &storepb.DatabaseMetadata{
		SyncError: bad,
	}
	SanitizeProtoStringFields(msg)

	want := strings.ToValidUTF8(bad, "")
	assert.Equal(t, want, msg.GetSyncError())
	assert.True(t, utf8.ValidString(msg.GetSyncError()))
}

func TestSanitizeProtoStringFields_MapValues(t *testing.T) {
	bad := "value\xff\xfe"
	msg := &storepb.DatabaseMetadata{
		Labels: map[string]string{
			"clean_key": bad,
		},
	}
	SanitizeProtoStringFields(msg)

	want := strings.ToValidUTF8(bad, "")
	assert.Equal(t, want, msg.GetLabels()["clean_key"])
	assert.True(t, utf8.ValidString(msg.GetLabels()["clean_key"]))
}

func TestSanitizeProtoStringFields_MapKeys(t *testing.T) {
	badKey := "key\xff"
	msg := &storepb.DatabaseMetadata{
		Labels: map[string]string{
			badKey:      "good_value",
			"other_key": "other_value",
		},
	}
	SanitizeProtoStringFields(msg)

	cleanKey := strings.ToValidUTF8(badKey, "")

	// Bad key should be removed, clean key inserted.
	_, hasBad := msg.GetLabels()[badKey]
	assert.False(t, hasBad, "bad key should be removed")

	val, hasClean := msg.GetLabels()[cleanKey]
	assert.True(t, hasClean, "cleaned key should exist")
	assert.Equal(t, "good_value", val)

	// Other key should be untouched.
	assert.Equal(t, "other_value", msg.GetLabels()["other_key"])
}

func TestSanitizeProtoStringFields_NestedMessage(t *testing.T) {
	badComment := "table comment\xc0\xaf"
	require.False(t, utf8.ValidString(badComment))

	msg := &storepb.DatabaseSchemaMetadata{
		Name: "test_db",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name:    "users",
						Comment: badComment,
					},
				},
			},
		},
	}
	SanitizeProtoStringFields(msg)

	want := strings.ToValidUTF8(badComment, "")
	got := msg.GetSchemas()[0].GetTables()[0].GetComment()
	assert.Equal(t, want, got)
	assert.True(t, utf8.ValidString(got))

	// Unaffected fields stay intact.
	assert.Equal(t, "test_db", msg.GetName())
	assert.Equal(t, "public", msg.GetSchemas()[0].GetName())
	assert.Equal(t, "users", msg.GetSchemas()[0].GetTables()[0].GetName())
}

func TestSanitizeProtoStringFields_NilMessage(t *testing.T) {
	// Must not panic.
	SanitizeProtoStringFields(nil)

	var msg *storepb.DatabaseMetadata
	SanitizeProtoStringFields(msg)
}

func TestSanitizeProtoStringFields_ValidDataUnchanged(t *testing.T) {
	msg := &storepb.DatabaseMetadata{
		Labels: map[string]string{
			"env":    "production",
			"region": "us-east-1",
		},
		SyncError: "all good",
		Release:   "projects/p1/releases/r1",
	}

	SanitizeProtoStringFields(msg)

	assert.Equal(t, "all good", msg.GetSyncError())
	assert.Equal(t, "projects/p1/releases/r1", msg.GetRelease())
	assert.Equal(t, "production", msg.GetLabels()["env"])
	assert.Equal(t, "us-east-1", msg.GetLabels()["region"])
}
