import { create } from "@bufbuild/protobuf";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { createApp, h } from "vue";
import { RevisionDataTable } from "@/components/Revision";
import CreateRevisionDrawer from "@/components/Revision/CreateRevisionDrawer.vue";
import { revisionServiceClientConnect } from "@/connect";
import i18n from "@/plugins/i18n";
import NaiveUI from "@/plugins/naive-ui";
import { Button } from "@/react/components/ui/button";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { router } from "@/router";
import { pinia } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import {
  ListRevisionsRequestSchema,
  type Revision as RevisionMessage,
} from "@/types/proto-es/v1/revision_service_pb";

function VueRevisionTableMount({
  revisions,
  loading,
  onDelete,
}: {
  revisions: Revision[];
  loading: boolean;
  onDelete: () => void;
}) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    const app = createApp({
      render() {
        return h(RevisionDataTable as never, {
          key: `revision-table.${revisions
            .map((revision) => revision.name)
            .join(",")}`,
          revisions,
          loading,
          showSelection: true,
          onDelete,
        });
      },
    });
    app.use(router).use(pinia).use(i18n).use(NaiveUI);
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, [loading, onDelete, revisions]);

  return <div ref={containerRef} />;
}

function VueCreateRevisionDrawerMount({
  databaseName,
  open,
  onOpenChange,
  onCreated,
}: {
  databaseName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (revisions: RevisionMessage[]) => void;
}) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    const app = createApp({
      render() {
        return h(CreateRevisionDrawer as never, {
          database: databaseName,
          show: open,
          "onUpdate:show": onOpenChange,
          onCreated,
        });
      },
    });
    app.use(router).use(pinia).use(i18n).use(NaiveUI);
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, [databaseName, onCreated, onOpenChange, open]);

  return <div ref={containerRef} />;
}

export function DatabaseRevisionPanel({ database }: { database: Database }) {
  const { t } = useTranslation();
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
    (_revisions: RevisionMessage[]) => {
      setShowCreateRevisionDrawer(false);
      paged.refresh();
    },
    [paged.refresh]
  );

  const handleRevisionDeleted = useCallback(() => {
    paged.refresh();
  }, [paged.refresh]);

  return (
    <>
      <div className="flex flex-col gap-y-2">
        <div className="flex items-center justify-between">
          <div />
          <Button onClick={() => setShowCreateRevisionDrawer(true)}>
            {t("common.import")}
          </Button>
        </div>
        <VueRevisionTableMount
          revisions={paged.dataList}
          loading={paged.isLoading}
          onDelete={handleRevisionDeleted}
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
        onOpenChange={setShowCreateRevisionDrawer}
        onCreated={handleRevisionCreated}
      />
    </>
  );
}
