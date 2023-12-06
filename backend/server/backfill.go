package server

import (
	"context"
	"log/slog"

	v1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/store"
)

func backfillBranches(ctx context.Context, stores *store.Store) {
	ids, err := stores.ListBackfillBranches(ctx)
	if err != nil {
		slog.Error("failed to list backfill branches", log.BBError(err))
		return
	}
	count := 0
	for _, id := range ids {
		branch, err := stores.GetBranch(ctx, &store.FindBranchMessage{UID: &id})
		if err != nil {
			slog.Error("failed to get branch", slog.Int("branchID", id), log.BBError(err))
			continue
		}
		headMetadata, err := v1.TransformSchemaStringToDatabaseMetadata(branch.Engine, string(branch.Head.Schema))
		if err != nil {
			slog.Error("failed to transform head metadata", slog.Int("branchID", id), log.BBError(err))
			continue
		}
		branch.Head.Metadata = headMetadata
		baseMetadata, err := v1.TransformSchemaStringToDatabaseMetadata(branch.Engine, string(branch.Base.Schema))
		if err != nil {
			slog.Error("failed to transform head metadata", slog.Int("branchID", id), log.BBError(err))
			continue
		}
		branch.Base.Metadata = baseMetadata
		if err := stores.UpdateBranch(ctx, &store.UpdateBranchMessage{
			ProjectID:  branch.ProjectID,
			ResourceID: branch.ResourceID,
			Head:       branch.Head,
			Base:       branch.Base,
			UpdaterID:  branch.UpdaterID,
		}); err != nil {
			slog.Error("failed to update branch", slog.Int("branchID", id), log.BBError(err))
			continue
		}
		slog.Info("backfilled branch metadata", slog.Int("branchID", id))
		count++
	}
	if len(ids) > 0 && count > 0 {
		slog.Info("backfill branch done", slog.Int("total", len(ids)), slog.Int("done", count))
	}
}
