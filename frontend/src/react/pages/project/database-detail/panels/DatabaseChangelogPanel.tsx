import { useCallback } from "react";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { useChangelogStore } from "@/store";
import type {
  Changelog,
  Database,
} from "@/types/proto-es/v1/database_service_pb";
import { DatabaseChangelogTable } from "../changelog/DatabaseChangelogTable";

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
      <DatabaseChangelogTable
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
