package base

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestRestoreHelpers(t *testing.T) {
	t.Run("ValidateBackupItem", func(t *testing.T) {
		require.EqualError(t, ValidateBackupItem(nil), "backup item is nil")
		require.EqualError(t, ValidateBackupItem(&storepb.PriorBackupDetail_Item{}), "backup item source table is nil")
		require.EqualError(t, ValidateBackupItem(&storepb.PriorBackupDetail_Item{
			SourceTable: &storepb.PriorBackupDetail_Item_Table{},
		}), "backup item target table is nil")
		require.NoError(t, ValidateBackupItem(&storepb.PriorBackupDetail_Item{
			SourceTable: &storepb.PriorBackupDetail_Item_Table{},
			TargetTable: &storepb.PriorBackupDetail_Item_Table{},
		}))
	})

	t.Run("ParseBackupItemDatabases", func(t *testing.T) {
		item := &storepb.PriorBackupDetail_Item{
			SourceTable: &storepb.PriorBackupDetail_Item_Table{Database: "instances/i1/databases/src"},
			TargetTable: &storepb.PriorBackupDetail_Item_Table{Database: "projects/p1/instances/i1/databases/bak"},
		}
		src, dst, err := ParseBackupItemDatabases(item)
		require.NoError(t, err)
		require.Equal(t, "src", src)
		require.Equal(t, "bak", dst)

		badSrc := &storepb.PriorBackupDetail_Item{
			SourceTable: &storepb.PriorBackupDetail_Item_Table{Database: "bad"},
			TargetTable: item.TargetTable,
		}
		_, _, err = ParseBackupItemDatabases(badSrc)
		require.ErrorContains(t, err, "failed to get source database ID for bad")

		badDst := &storepb.PriorBackupDetail_Item{
			SourceTable: item.SourceTable,
			TargetTable: &storepb.PriorBackupDetail_Item_Table{Database: "bad"},
		}
		_, _, err = ParseBackupItemDatabases(badDst)
		require.ErrorContains(t, err, "failed to get target database ID for bad")
	})
}
