import { Loader2, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { SearchParams } from "@/react/components/AdvancedSearch";
import {
  BatchActionBar,
  BatchIssueStatusActionDrawer,
  IssueListItem,
  IssueSearchBar,
  PresetButtons,
  useIssueSearchScopeOptions,
} from "@/react/components/IssueTable";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  refreshIssueList,
  useCurrentUserV1,
  useIssueV1Store,
  useUIStateStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  buildIssueFilterBySearchParams,
  buildSearchParamsBySearchText,
  buildSearchTextBySearchParams,
  mergeSearchParams,
  type SearchParams as VueSearchParams,
  type SearchScope as VueSearchScope,
} from "@/utils";

export function ProjectIssueDashboardPage({
  projectId,
}: {
  projectId: string;
}) {
  const { t } = useTranslation();
  const issueStore = useIssueV1Store();
  const uiStateStore = useUIStateStore();
  const currentUser = useCurrentUserV1();
  const me = useVueState(() => currentUser.value);

  const projectName = `${projectNamePrefix}${projectId}`;

  // Hint
  const HINT_KEY = "issue.hint-dismissed";
  const hideHint = useVueState(() => uiStateStore.getIntroStateByKey(HINT_KEY));
  const dismissHint = useCallback(() => {
    uiStateStore.saveIntroStateByKey({ key: HINT_KEY, newState: true });
  }, [uiStateStore]);

  // Read-only scopes
  const readonlyScopes: VueSearchScope[] = useMemo(
    () => [{ id: "project", value: projectId, readonly: true }],
    [projectId]
  );

  const defaultSearchParams = useCallback((): SearchParams => {
    const myEmail = me?.email ?? "";
    return {
      query: "",
      scopes: [
        ...readonlyScopes.map((s) => ({
          id: s.id,
          value: s.value,
          readonly: s.readonly,
        })),
        { id: "status", value: IssueStatus[IssueStatus.OPEN] },
        {
          id: "approval",
          value: ApprovalStatus[ApprovalStatus.PENDING],
        },
        { id: "current-approver", value: myEmail },
      ],
    };
  }, [readonlyScopes, me]);

  // Initialize from URL or defaults
  const initialQueryRef = useRef<string | null>(null);
  const [searchParams, setSearchParams] = useState<SearchParams>(() => {
    const urlQ = new URLSearchParams(window.location.search).get("q") ?? null;
    initialQueryRef.current = urlQ;
    if (urlQ) {
      const urlParams = buildSearchParamsBySearchText(urlQ);
      const base: VueSearchParams = {
        query: "",
        scopes: readonlyScopes as VueSearchScope[],
      };
      const merged = mergeSearchParams(base, urlParams as VueSearchParams);
      return {
        query: merged.query,
        scopes: merged.scopes.map((s) => ({
          id: s.id,
          value: s.value,
          readonly: (s as VueSearchScope & { readonly?: boolean }).readonly,
        })),
      };
    }
    return defaultSearchParams();
  });

  const [orderBy, setOrderBy] = useState("");

  // Reset on project change
  useEffect(() => {
    setSearchParams(defaultSearchParams());
  }, [projectId]);

  // URL sync
  const isUpdatingUrl = useRef(false);
  useEffect(() => {
    if (isUpdatingUrl.current) return;
    const queryString = buildSearchTextBySearchParams(
      searchParams as VueSearchParams
    );
    const currentQ = new URLSearchParams(window.location.search).get("q");
    if (queryString === currentQ) return;

    const isDefault =
      queryString ===
      buildSearchTextBySearchParams(defaultSearchParams() as VueSearchParams);
    if (isDefault && !initialQueryRef.current) {
      if (currentQ) {
        isUpdatingUrl.current = true;
        router
          .replace({
            query: { ...router.currentRoute.value.query, q: undefined },
          })
          .finally(() => {
            isUpdatingUrl.current = false;
          });
      }
    } else {
      isUpdatingUrl.current = true;
      router
        .replace({
          query: {
            ...router.currentRoute.value.query,
            q: queryString || undefined,
          },
        })
        .finally(() => {
          isUpdatingUrl.current = false;
        });
    }
  }, [searchParams]);

  // Issue filter
  const issueFilter = useMemo(() => {
    const filter = buildIssueFilterBySearchParams(
      searchParams as VueSearchParams
    );
    filter.orderBy = orderBy;
    return filter;
  }, [searchParams, orderBy]);

  const fetchIssueList = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const { nextPageToken, issues } = await issueStore.listIssues({
        find: issueFilter,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
      });
      return { list: issues, nextPageToken };
    },
    [issueStore, issueFilter]
  );

  const paged = usePagedData<Issue>({
    sessionKey: "bb.issue-table.project-issues",
    fetchList: fetchIssueList,
  });

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
    <div className="py-4 flex flex-col">
      <div className="px-4 flex flex-col gap-y-2">
        {!hideHint && (
          <DismissibleAlert onClose={dismissHint}>
            {t("issue.subtitle")}
          </DismissibleAlert>
        )}
        <IssueSearchBar
          params={searchParams}
          onParamsChange={setSearchParams}
          orderBy={orderBy}
          onOrderByChange={setOrderBy}
          scopeOptions={scopeOptions}
        />
        <PresetButtons params={searchParams} onParamsChange={setSearchParams} />
      </div>

      <div className="mt-2">
        {paged.isLoading ? (
          <div className="flex justify-center py-8 text-control-light">
            <Loader2 className="w-5 h-5 animate-spin" />
          </div>
        ) : paged.dataList.length === 0 ? (
          <div className="flex justify-center py-8 text-control-light">
            {t("common.no-data")}
          </div>
        ) : (
          paged.dataList.map((issue) => (
            <IssueListItem
              key={issue.name}
              issue={issue}
              selected={selectedNames.has(issue.name)}
              onToggleSelection={() => toggleSelection(issue.name)}
              highlightText={searchParams.query}
            />
          ))
        )}
        {paged.dataList.length > 0 && (
          <div className="mt-4 mx-2">
            <PagedTableFooter
              pageSize={paged.pageSize}
              pageSizeOptions={paged.pageSizeOptions}
              onPageSizeChange={paged.onPageSizeChange}
              hasMore={paged.hasMore}
              isFetchingMore={paged.isFetchingMore}
              onLoadMore={paged.loadMore}
            />
          </div>
        )}
      </div>

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
    </div>
  );
}

// ---------------------------------------------------------------------------
// DismissibleAlert
// ---------------------------------------------------------------------------

function DismissibleAlert({
  children,
  onClose,
}: {
  children: React.ReactNode;
  onClose: () => void;
}) {
  return (
    <div className="relative w-full rounded-xs border border-accent/30 bg-accent/5 text-accent px-4 py-3 text-sm flex gap-x-3 items-start">
      <svg
        className="h-5 w-5 shrink-0 mt-0.5"
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="10" />
        <path d="M12 16v-4" />
        <path d="M12 8h.01" />
      </svg>
      <div className="flex-1">{children}</div>
      <button
        className="p-0.5 hover:bg-accent/10 rounded-xs shrink-0"
        onClick={onClose}
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}
