import { create } from "@bufbuild/protobuf";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { revisionServiceClientConnect } from "@/connect";
import { Button } from "@/react/components/ui/button";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { useRevisionStore } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import {
  ListRevisionsRequestSchema,
  type Revision as RevisionMessage,
} from "@/types/proto-es/v1/revision_service_pb";
import { CreateRevisionDialog } from "../revision/CreateRevisionDialog";
import { DatabaseRevisionTable } from "../revision/DatabaseRevisionTable";

export function DatabaseRevisionPanel({ database }: { database: Database }) {
  const { t } = useTranslation();
  const revisionStore = useRevisionStore();
  const [showCreateRevisionDrawer, setShowCreateRevisionDrawer] =
    useState(false);
  const [existingVersions, setExistingVersions] = useState<string[]>([]);
  const [isLoadingExistingVersions, setIsLoadingExistingVersions] =
    useState(false);
  const fetchRevisionList = useCallback(
    async ({
      pageToken,
      pageSize,
    }: {
      pageToken: string;
      pageSize: number;
    }) => {
      const request = create(ListRevisionsRequestSchema, {
        parent: database.name,
        pageSize,
        pageToken,
      });
      const { nextPageToken, revisions } =
        await revisionServiceClientConnect.listRevisions(request);
      return {
        nextPageToken,
        list: revisions,
      };
    },
    [database.name]
  );
  const fetchAllRevisionVersions = useCallback(async () => {
    const versions = new Set<string>();
    let pageToken = "";

    do {
      const request = create(ListRevisionsRequestSchema, {
        parent: database.name,
        pageSize: 1000,
        pageToken,
      });
      const { nextPageToken, revisions } =
        await revisionServiceClientConnect.listRevisions(request);

      for (const revision of revisions) {
        if (revision.version) {
          versions.add(revision.version);
        }
      }

      pageToken = nextPageToken;
    } while (pageToken);

    return Array.from(versions);
  }, [database.name]);
  const paged = usePagedData<Revision>({
    sessionKey: `bb.paged-revision-table.${database.name}`,
    fetchList: fetchRevisionList,
  });

  useEffect(() => {
    if (!showCreateRevisionDrawer) {
      return;
    }

    let cancelled = false;
    setIsLoadingExistingVersions(true);
    void fetchAllRevisionVersions()
      .then((versions) => {
        if (!cancelled) {
          setExistingVersions(versions);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoadingExistingVersions(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [fetchAllRevisionVersions, showCreateRevisionDrawer]);

  const handleRevisionCreated = useCallback(
    (_revisions: RevisionMessage[]) => {
      setShowCreateRevisionDrawer(false);
      paged.refresh();
    },
    [paged.refresh]
  );

  const handleRevisionDeleted = useCallback(() => {
    paged.refresh();
  }, [paged.refresh]);

  const handleDelete = useCallback(
    async (name: string) => {
      if (!window.confirm(t("database.revision.delete-confirm-dialog"))) {
        return;
      }
      await revisionStore.deleteRevision(name);
      handleRevisionDeleted();
    },
    [handleRevisionDeleted, revisionStore, t]
  );

  return (
    <>
      <div className="flex flex-col gap-y-2">
        <div className="flex items-center justify-between">
          <div />
          <Button onClick={() => setShowCreateRevisionDrawer(true)}>
            {t("common.import")}
          </Button>
        </div>
        <DatabaseRevisionTable
          revisions={paged.dataList}
          loading={paged.isLoading}
          onDelete={handleDelete}
        />
        <PagedTableFooter
          pageSize={paged.pageSize}
          pageSizeOptions={paged.pageSizeOptions}
          onPageSizeChange={paged.onPageSizeChange}
          hasMore={paged.hasMore}
          isFetchingMore={paged.isFetchingMore}
          onLoadMore={paged.loadMore}
        />
      </div>
      <CreateRevisionDialog
        databaseName={database.name}
        existingVersions={existingVersions}
        isCheckingExistingVersions={isLoadingExistingVersions}
        open={showCreateRevisionDrawer}
        projectName={database.project}
        onOpenChange={setShowCreateRevisionDrawer}
        onCreated={handleRevisionCreated}
      />
    </>
  );
}
