import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "react-router";
import {
  markScrollRestorationEntry,
  useScrollRestorationKey,
  useScrollRestorationLoadMore,
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
  ProjectPageContent,
  ProjectPageFooter,
  ProjectPageInfo,
  ProjectPageLayout,
} from "@/components/ProjectPageLayout";
import { useCurrentUser } from "@/hooks/useAppState";
import { PagedTableFooter, usePagedData } from "@/hooks/usePagedData";
import { useURLSearchParam } from "@/hooks/useURLSearchParam";
import { refreshIssueList } from "@/lib/issue/issueListRefresh";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  buildIssueFilterBySearchParams,
  buildSearchParamsBySearchText,
  buildSearchTextBySearchParams,
  mergeSearchParams,
  type SearchScope as VueSearchScope,
} from "@/utils";
import { projectIssuesPagedDataCacheScope } from "./pagedDataCacheScope";

const serializeSearchParams = (params: SearchParams): string =>
  buildSearchTextBySearchParams({
    ...params,
    scopes: params.scopes.filter((scope) => !scope.readonly),
  });

export function ProjectIssueDashboardPage({
  projectId,
}: {
  projectId: string;
}) {
  const { t } = useTranslation();
  const location = useLocation();
  const scrollRestorationKey = useScrollRestorationKey();
  const batchGetOrFetchUsers = useAppStore(
    (state) => state.batchGetOrFetchUsers
  );
  const me = useCurrentUser();

  const projectName = `${projectNamePrefix}${projectId}`;

  // Read-only scopes
  const readonlyScopes: VueSearchScope[] = useMemo(
    () => [{ id: "project", value: projectId, readonly: true }],
    [projectId]
  );

  const defaultSearchParams = useMemo((): SearchParams => {
    const myEmail = me?.email ?? "";
    return {
      query: "",
      scopes: [
        ...readonlyScopes,
        { id: "status", value: IssueStatus[IssueStatus.OPEN] },
        {
          id: "approval",
          value: ApprovalStatus[ApprovalStatus.PENDING],
        },
        { id: "current-approver", value: myEmail },
      ],
    };
  }, [readonlyScopes, me]);

  const parseSearchParams = useCallback(
    (query: string): SearchParams => {
      const urlParams = buildSearchParamsBySearchText(query);
      return mergeSearchParams(
        { query: "", scopes: [...readonlyScopes] },
        {
          ...urlParams,
          // The project is owned by the route path, not the query string.
          scopes: urlParams.scopes.filter((scope) => scope.id !== "project"),
        }
      );
    },
    [readonlyScopes]
  );
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
        "project-issues",
        projectName,
        serializeSearchParams(searchParams),
        orderBy,
      ]),
    [orderBy, projectName, searchParams]
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
    sessionKey: "bb.issue-table.project-issues",
    cacheKey: viewCacheKey,
    cacheScope: projectIssuesPagedDataCacheScope(projectId),
    cacheRestoreToken: scrollRestorationKey,
    fetchList: fetchIssueList,
  });
  useScrollRestorationLoadMore(paged);

  useEffect(() => {
    if (paged.dataList.length === 0) {
      return;
    }
    void batchGetOrFetchUsers(paged.dataList.map((issue) => issue.creator));
  }, [batchGetOrFetchUsers, paged.dataList]);

  // Scope options
  const scopeOptions = useIssueSearchScopeOptions(projectName);

  // Selection state
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
  const handleOpenIssue = useCallback(
    () => markScrollRestorationEntry(location),
    [location]
  );

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
    <ProjectPageLayout>
      <div className="flex flex-col gap-y-2">
        <ProjectPageInfo description={t("issue.subtitle")} />
        <IssueSearchBar
          params={searchParams}
          onParamsChange={setSearchParams}
          orderBy={orderBy}
          onOrderByChange={setOrderBy}
          scopeOptions={scopeOptions}
        />
      </div>

      <ProjectPageContent>
        <IssueListPanel
          params={searchParams}
          onParamsChange={setSearchParams}
          isLoading={paged.isLoading}
          issues={paged.dataList}
          selectedNames={selectedNames}
          onToggleSelection={toggleSelection}
          onOpenIssue={handleOpenIssue}
        />
        {paged.dataList.length > 0 && (
          <ProjectPageFooter className="px-2">
            <PagedTableFooter
              pageSize={paged.pageSize}
              pageSizeOptions={paged.pageSizeOptions}
              onPageSizeChange={paged.onPageSizeChange}
              hasMore={paged.hasMore}
              isFetchingMore={paged.isFetchingMore}
              onLoadMore={paged.loadMore}
            />
          </ProjectPageFooter>
        )}
      </ProjectPageContent>

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
    </ProjectPageLayout>
  );
}
