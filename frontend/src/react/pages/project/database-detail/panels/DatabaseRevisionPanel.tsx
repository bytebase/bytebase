import { create } from "@bufbuild/protobuf";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { createApp, h, reactive } from "vue";
import CreateRevisionDrawer from "@/components/Revision/CreateRevisionDrawer.vue";
import { revisionServiceClientConnect } from "@/connect";
import i18n from "@/plugins/i18n";
import NaiveUI from "@/plugins/naive-ui";
import { Button } from "@/react/components/ui/button";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { router } from "@/router";
import { pinia, useRevisionStore } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import { ListRevisionsRequestSchema } from "@/types/proto-es/v1/revision_service_pb";
import { DatabaseRevisionTable } from "../revision/DatabaseRevisionTable";

function VueCreateRevisionDrawerMount({
  databaseName,
  open,
  onCreated,
  onOpenChange,
}: {
  databaseName: string;
  open: boolean;
  onCreated: (revisions: Revision[]) => void;
  onOpenChange: (open: boolean) => void;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const bridgeStateRef = useRef(
    reactive({
      databaseName,
      onCreated,
      onOpenChange,
      open,
    })
  );

  useEffect(() => {
    const bridgeState = bridgeStateRef.current;
    bridgeState.databaseName = databaseName;
    bridgeState.onCreated = onCreated;
    bridgeState.onOpenChange = onOpenChange;
    bridgeState.open = open;
  }, [databaseName, onCreated, onOpenChange, open]);

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    const bridgeState = bridgeStateRef.current;

    const app = createApp({
      render() {
        return h(CreateRevisionDrawer as never, {
          database: bridgeState.databaseName,
          show: bridgeState.open,
          "onUpdate:show": bridgeState.onOpenChange,
          onCreated: bridgeState.onCreated,
        });
      },
    });
    app.use(router).use(pinia).use(i18n).use(NaiveUI);
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, []);

  return <div ref={containerRef} />;
}

export function DatabaseRevisionPanel({ database }: { database: Database }) {
  const { t } = useTranslation();
  const revisionStore = useRevisionStore();
  const [showCreateRevisionDrawer, setShowCreateRevisionDrawer] =
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
  const paged = usePagedData<Revision>({
    sessionKey: `bb.paged-revision-table.${database.name}`,
    fetchList: fetchRevisionList,
  });

  const handleRevisionCreated = useCallback(
    (_revisions: Revision[]) => {
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
      <VueCreateRevisionDrawerMount
        databaseName={database.name}
        open={showCreateRevisionDrawer}
        onCreated={handleRevisionCreated}
        onOpenChange={setShowCreateRevisionDrawer}
      />
    </>
  );
}
