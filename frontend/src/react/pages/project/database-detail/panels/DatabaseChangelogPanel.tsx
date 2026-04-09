import { useCallback, useEffect, useRef } from "react";
import { h } from "vue";
import { ChangelogDataTable } from "@/components/Changelog";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { createLegacyVueApp } from "@/react/legacy/mountLegacyVueApp";
import { useChangelogStore } from "@/store";
import type {
  Changelog,
  Database,
} from "@/types/proto-es/v1/database_service_pb";

function VueChangelogTableMount({
  changelogs,
  loading,
}: {
  changelogs: Changelog[];
  loading: boolean;
}) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    const app = createLegacyVueApp({
      render() {
        return h(ChangelogDataTable as never, {
          key: `changelog-table.${changelogs
            .map((changelog) => changelog.name)
            .join(",")}`,
          changelogs,
          loading,
          showSelection: false,
        });
      },
    });
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, [changelogs, loading]);

  return <div ref={containerRef} />;
}

export function DatabaseChangelogPanel({ database }: { database: Database }) {
  const changelogStore = useChangelogStore();
  const fetchChangelogList = useCallback(
    async ({
      pageToken,
      pageSize,
    }: {
      pageToken: string;
      pageSize: number;
    }) => {
      const { nextPageToken, changelogs } =
        await changelogStore.fetchChangelogList({
          parent: database.name,
          pageSize,
          pageToken,
        });
      return {
        nextPageToken,
        list: changelogs,
      };
    },
    [changelogStore, database.name]
  );
  const paged = usePagedData<Changelog>({
    sessionKey: `bb.paged-changelog-table.${database.name}`,
    fetchList: fetchChangelogList,
  });

  return (
    <div className="flex flex-col gap-y-4">
      <VueChangelogTableMount
        changelogs={paged.dataList}
        loading={paged.isLoading}
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
  );
}
