import { useCallback, useEffect, useMemo, useState } from "react";
import {
  markListScrollRestorationEntry,
  useListScrollRestorationKey,
  useListScrollRestorationLoadMore,
} from "@/app/router/NavigationScrollRestoration";
import type { SearchParams } from "@/components/AdvancedSearch";
import {
  BatchActionBar,
  BatchIssueStatusActionDrawer,
  IssueListPanel,
  IssueSearchBar,
  useIssueSearchScopeOptions,
} from "@/components/IssueTable";
import {
  WorkspacePageContent,
  WorkspacePageFooter,
  WorkspacePageLayout,
} from "@/components/WorkspacePageLayout";
import { useCurrentUser } from "@/hooks/useAppState";
import { PagedTableFooter, usePagedData } from "@/hooks/usePagedData";
import { useURLSearchParam } from "@/hooks/useURLSearchParam";
import { refreshIssueList } from "@/lib/issue/issueListRefresh";
import { useAppStore } from "@/stores/app";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  buildIssueFilterBySearchParams,
  buildSearchParamsBySearchText,
  buildSearchTextBySearchParams,
} from "@/utils";

const parseSearchParams = (query: string): SearchParams =>
  buildSearchParamsBySearchText(query);
const serializeSearchParams = (params: SearchParams): string =>
  buildSearchTextBySearchParams(params);

export function MyIssuesPage() {
  const listScrollRestorationKey = useListScrollRestorationKey();
  const batchGetOrFetchUsers = useAppStore(
    (state) => state.batchGetOrFetchUsers
  );
  const me = useCurrentUser();

  const defaultSearchParams = useMemo((): SearchParams => {
    const myEmail = me?.email ?? "";
    return {
      query: "",
      scopes: [
        { id: "status", value: IssueStatus[IssueStatus.OPEN] },
        {
          id: "approval",
          value: ApprovalStatus[ApprovalStatus.PENDING],
        },
        { id: "current-approver", value: myEmail },
      ],
    };
  }, [me]);

  const [searchParams, setSearchParams] = useURLSearchParam({
    param: "q",
    parse: parseSearchParams,
    serialize: serializeSearchParams,
    defaultValue: defaultSearchParams,
  });
  const [orderBy, setOrderBy] = useURLSearchParam({
    param: "order",
    defaultValue: "",
  });
  const viewCacheKey = useMemo(
    () =>
      JSON.stringify([
        "my-issues",
        serializeSearchParams(searchParams),
        orderBy,
      ]),
    [orderBy, searchParams]
  );

  // Issue filter
  const issueFilter = useMemo(() => {
    const filter = buildIssueFilterBySearchParams(searchParams);
    filter.orderBy = orderBy;
    return filter;
  }, [searchParams, orderBy]);

  const fetchIssueList = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const { nextPageToken, issues } = await useAppStore
        .getState()
        .listIssues({
          find: issueFilter,
          pageSize: params.pageSize,
          pageToken: params.pageToken,
        });
      return { list: issues, nextPageToken };
    },
    [issueFilter]
  );

  const paged = usePagedData<Issue>({
    sessionKey: "bb.issue-table.my-issues",
    cacheKey: viewCacheKey,
    cacheRestoreToken: listScrollRestorationKey,
    fetchList: fetchIssueList,
  });
  useListScrollRestorationLoadMore(paged);

  useEffect(() => {
    if (paged.dataList.length === 0) {
      return;
    }
    void batchGetOrFetchUsers(paged.dataList.map((issue) => issue.creator));
  }, [batchGetOrFetchUsers, paged.dataList]);

  // Scope options (no project scope — cross-project)
  const scopeOptions = useIssueSearchScopeOptions();

  // Selection
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set());

  useEffect(() => {
    setSelectedNames((prev) => {
      const current = new Set(paged.dataList.map((i) => i.name));
      const next = new Set<string>();
      for (const name of prev) {
        if (current.has(name)) next.add(name);
      }
      return next.size === prev.size ? prev : next;
    });
  }, [paged.dataList]);

  const selectedIssues = useMemo(
    () => paged.dataList.filter((i) => selectedNames.has(i.name)),
    [paged.dataList, selectedNames]
  );

  const toggleSelection = useCallback((name: string) => {
    setSelectedNames((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }, []);
  const toggleSelectAll = useCallback(() => {
    setSelectedNames((prev) => {
      const allSelected =
        paged.dataList.length > 0 &&
        paged.dataList.every((i) => prev.has(i.name));
      if (allSelected) return new Set();
      return new Set(paged.dataList.map((i) => i.name));
    });
  }, [paged.dataList]);

  // Batch actions
  const [batchAction, setBatchAction] = useState<
    "CLOSE" | "REOPEN" | undefined
  >();

  const handleBatchUpdated = useCallback(() => {
    setBatchAction(undefined);
    setSelectedNames(new Set());
    refreshIssueList();
    paged.refresh();
  }, [paged]);

  return (
    <WorkspacePageLayout>
      <IssueSearchBar
        params={searchParams}
        onParamsChange={setSearchParams}
        orderBy={orderBy}
        onOrderByChange={setOrderBy}
        scopeOptions={scopeOptions}
      />

      <WorkspacePageContent>
        <IssueListPanel
          params={searchParams}
          onParamsChange={setSearchParams}
          isLoading={paged.isLoading}
          issues={paged.dataList}
          selectedNames={selectedNames}
          onToggleSelection={toggleSelection}
          onOpenIssue={markListScrollRestorationEntry}
          showProject
        />
        {paged.dataList.length > 0 && (
          <WorkspacePageFooter className="px-2">
            <PagedTableFooter
              pageSize={paged.pageSize}
              pageSizeOptions={paged.pageSizeOptions}
              onPageSizeChange={paged.onPageSizeChange}
              hasMore={paged.hasMore}
              isFetchingMore={paged.isFetchingMore}
              onLoadMore={paged.loadMore}
            />
          </WorkspacePageFooter>
        )}
      </WorkspacePageContent>

      {selectedIssues.length > 0 && (
        <BatchActionBar
          issues={selectedIssues}
          allSelected={
            paged.dataList.length > 0 &&
            paged.dataList.every((i) => selectedNames.has(i.name))
          }
          onToggleSelectAll={toggleSelectAll}
          onStartAction={setBatchAction}
        />
      )}

      {/* Modals (portaled, position-independent) */}
      <BatchIssueStatusActionDrawer
        issues={selectedIssues}
        action={batchAction}
        onClose={() => setBatchAction(undefined)}
        onUpdated={handleBatchUpdated}
      />
    </WorkspacePageLayout>
  );
}
