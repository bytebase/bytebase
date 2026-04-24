import { create } from "@bufbuild/protobuf";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { revisionServiceClientConnect } from "@/connect";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { useRevisionStore } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import { ListRevisionsRequestSchema } from "@/types/proto-es/v1/revision_service_pb";
import { DatabaseRevisionTable } from "../revision/DatabaseRevisionTable";
import { ImportRevisionSheet } from "../revision/ImportRevisionSheet";

export function DatabaseRevisionPanel({ database }: { database: Database }) {
  const { t } = useTranslation();
  const revisionStore = useRevisionStore();
  const [showCreateRevisionDrawer, setShowCreateRevisionDrawer] =
    useState(false);
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set());
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
  const paged = usePagedData<Revision>({
    sessionKey: `bb.paged-revision-table.${database.name}`,
    fetchList: fetchRevisionList,
  });

  const refreshRef = useRef(paged.refresh);
  refreshRef.current = paged.refresh;

  const handleRevisionCreated = useCallback((_revisions: Revision[]) => {
    setShowCreateRevisionDrawer(false);
    refreshRef.current();
  }, []);

  const handleRevisionDeleted = useCallback(() => {
    refreshRef.current();
  }, []);

  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const handleDeleteSelected = useCallback(async () => {
    try {
      await Promise.allSettled(
        [...selectedNames].map((name) => revisionStore.deleteRevision(name))
      );
    } finally {
      setSelectedNames(new Set());
      setShowDeleteConfirm(false);
      handleRevisionDeleted();
    }
  }, [selectedNames, handleRevisionDeleted, revisionStore]);

  // Clear selection on page size change or database switch, but not on loadMore.
  useEffect(() => {
    setSelectedNames(new Set());
  }, [paged.pageSize, database.name]);

  return (
    <>
      <div className="flex flex-col gap-y-2">
        <div className="flex items-center justify-between">
          <div />
          <Button
            onClick={() => {
              setShowCreateRevisionDrawer(true);
            }}
          >
            {t("common.import")}
          </Button>
        </div>
        <DatabaseRevisionTable
          revisions={paged.dataList}
          loading={paged.isLoading}
          selectedNames={selectedNames}
          onSelectedNamesChange={setSelectedNames}
        />
        <div className="mt-2 flex items-center justify-between">
          {selectedNames.size > 0 ? (
            <Button
              variant="destructive"
              onClick={() => setShowDeleteConfirm(true)}
            >
              {t("common.delete")} ({selectedNames.size})
            </Button>
          ) : (
            <div />
          )}
          <PagedTableFooter
            pageSize={paged.pageSize}
            pageSizeOptions={paged.pageSizeOptions}
            onPageSizeChange={paged.onPageSizeChange}
            hasMore={paged.hasMore}
            isFetchingMore={paged.isFetchingMore}
            onLoadMore={paged.loadMore}
          />
        </div>
      </div>

      <AlertDialog
        open={showDeleteConfirm}
        onOpenChange={(open) => {
          if (!open) setShowDeleteConfirm(false);
        }}
      >
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("database.revision.delete-confirm-dialog")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("common.cannot-undo-this-action")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDeleteConfirm(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={handleDeleteSelected}>
              {t("common.delete")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <ImportRevisionSheet
        databaseName={database.name}
        projectName={database.project}
        open={showCreateRevisionDrawer}
        onCreated={handleRevisionCreated}
        onOpenChange={setShowCreateRevisionDrawer}
      />
    </>
  );
}
