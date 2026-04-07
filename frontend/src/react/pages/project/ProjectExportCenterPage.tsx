import { Download, Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AdvancedSearch,
  type ScopeOption,
  type SearchParams,
  type SearchScope,
} from "@/react/components/AdvancedSearch";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import {
  PagedTableFooter,
  usePagedData,
} from "@/react/pages/settings/shared/usePagedData";
import { router } from "@/router";
import { WORKSPACE_ROUTE_USER_PROFILE } from "@/router/dashboard/workspaceRoutes";
import { useIssueV1Store, useProjectV1Store, useUserStore } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import { RiskLevel } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  Issue_ApprovalStatus,
  Issue_Type,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  buildIssueFilterBySearchParams,
  displayRoleTitle,
  extractIssueUID,
  extractProjectResourceName,
  formatAbsoluteDateTime,
  getHighlightHTMLByRegExp,
  getIssueRoute,
  hasProjectPermissionV2,
  humanizeTs,
  PERMISSIONS_FOR_DATABASE_EXPORT_ISSUE,
  type SearchParams as VueSearchParams,
} from "@/utils";
import { DataExportPrepDrawer } from "./export-center/DataExportPrepDrawer";

export function ProjectExportCenterPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const issueStore = useIssueV1Store();
  const projectStore = useProjectV1Store();

  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const [showDrawer, setShowDrawer] = useState(false);

  // Read-only scopes (always present, not removable)
  const readonlyScopes: SearchScope[] = useMemo(
    () => [
      { id: "project", value: projectId, readonly: true },
      {
        id: "issue-type",
        value: Issue_Type[Issue_Type.DATABASE_EXPORT],
        readonly: true,
      },
    ],
    [projectId]
  );

  const defaultSearchParams = useCallback(
    (): SearchParams => ({
      query: "",
      scopes: [
        ...readonlyScopes,
        { id: "status", value: IssueStatus[IssueStatus.OPEN] },
      ],
    }),
    [readonlyScopes]
  );

  const [searchParams, setSearchParams] =
    useState<SearchParams>(defaultSearchParams);

  // Reset search params when projectId changes (e.g. navigating between projects)
  useEffect(() => {
    setSearchParams(defaultSearchParams());
  }, [projectId]);

  // Scope options for the search bar
  const scopeOptions: ScopeOption[] = useMemo(
    () => [
      {
        id: "status",
        title: t("common.status"),
        options: [
          {
            value: IssueStatus[IssueStatus.OPEN],
            keywords: ["open"],
            render: () => <span>{t("issue.table.open")}</span>,
          },
          {
            value: IssueStatus[IssueStatus.DONE],
            keywords: ["done", "approved"],
            render: () => <span>{t("common.approved")}</span>,
          },
          {
            value: IssueStatus[IssueStatus.CANCELED],
            keywords: ["canceled", "closed"],
            render: () => <span>{t("common.closed")}</span>,
          },
        ],
      },
      {
        id: "instance",
        title: t("common.instance"),
      },
      {
        id: "database",
        title: t("common.database"),
      },
      {
        id: "issue-label",
        title: t("issue.labels"),
      },
    ],
    [t]
  );

  // Permission check for creating export issues
  const canCreate = useMemo(() => {
    if (!project) return false;
    return PERMISSIONS_FOR_DATABASE_EXPORT_ISSUE.every((p) =>
      hasProjectPermissionV2(project, p)
    );
  }, [project]);

  // Build issue filter from search params
  const issueFilter = useMemo(() => {
    // Merge the readonly scopes into the search params for the filter builder
    const merged: VueSearchParams = {
      query: searchParams.query,
      scopes: searchParams.scopes.map((s) => ({
        id: s.id,
        value: s.value,
      })),
    };
    return buildIssueFilterBySearchParams(merged);
  }, [searchParams]);

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
    sessionKey: "export-center",
    fetchList: fetchIssueList,
  });

  return (
    <div className="py-4 w-full flex flex-col">
      <div className="px-4">
        <div className="w-full flex flex-col lg:flex-row items-start lg:items-center justify-between gap-2">
          <div className="flex flex-1 max-w-full items-center gap-x-2">
            <AdvancedSearch
              params={searchParams}
              onParamsChange={setSearchParams}
              scopeOptions={scopeOptions}
              placeholder={t("issue.advanced-search.filter")}
            />
            <Tooltip
              content={
                !canCreate
                  ? t("common.missing-required-permission", {
                      permissions:
                        PERMISSIONS_FOR_DATABASE_EXPORT_ISSUE.join(", "),
                    })
                  : undefined
              }
            >
              <Button disabled={!canCreate} onClick={() => setShowDrawer(true)}>
                <Download className="h-4 w-4 mr-1" />
                {t("quick-action.request-export-data")}
              </Button>
            </Tooltip>
          </div>
        </div>
      </div>

      {/* Issue list */}
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
          <>
            {paged.dataList.map((issue) => (
              <IssueListItem
                key={issue.name}
                issue={issue}
                highlightText={searchParams.query}
              />
            ))}
          </>
        )}

        {paged.dataList.length > 0 && (
          <div className="mx-4 mt-3">
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

      <DataExportPrepDrawer
        open={showDrawer}
        onClose={() => setShowDrawer(false)}
        projectName={project?.name ?? projectName}
      />
    </div>
  );
}

// ---------------------------------------------------------------------------
// IssueStatusIcon
// ---------------------------------------------------------------------------

function IssueStatusIcon({ status }: { status: IssueStatus }) {
  switch (status) {
    case IssueStatus.OPEN:
      return (
        <span className="flex items-center justify-center rounded-full w-5 h-5 bg-white border-2 border-info text-info shrink-0">
          <span className="h-1.5 w-1.5 bg-info rounded-full" />
        </span>
      );
    case IssueStatus.DONE:
      return (
        <span className="flex items-center justify-center rounded-full w-5 h-5 bg-success text-white shrink-0">
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
              clipRule="evenodd"
            />
          </svg>
        </span>
      );
    case IssueStatus.CANCELED:
      return (
        <span className="flex items-center justify-center rounded-full w-5 h-5 bg-white border-2 text-gray-400 border-gray-400 shrink-0">
          <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M3 10a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
              clipRule="evenodd"
            />
          </svg>
        </span>
      );
    default:
      return null;
  }
}

// ---------------------------------------------------------------------------
// RiskLevelIcon
// ---------------------------------------------------------------------------

function RiskLevelIcon({ riskLevel }: { riskLevel: RiskLevel }) {
  const { t } = useTranslation();
  if (
    riskLevel === RiskLevel.RISK_LEVEL_UNSPECIFIED ||
    riskLevel === RiskLevel.LOW
  ) {
    return null;
  }
  const color =
    riskLevel === RiskLevel.MODERATE ? "text-warning" : "text-error";
  const label =
    riskLevel === RiskLevel.MODERATE
      ? t("issue.risk-level.moderate")
      : t("issue.risk-level.high");
  return (
    <Tooltip content={`${label} (${t("issue.risk-level.self")})`}>
      <svg
        className={`w-4 h-4 shrink-0 ${color}`}
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={2}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4.5c-.77-.833-2.694-.833-3.464 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z"
        />
      </svg>
    </Tooltip>
  );
}

// ---------------------------------------------------------------------------
// IssueApprovalStatusTag
// ---------------------------------------------------------------------------

function IssueApprovalStatusTag({ issue }: { issue: Issue }) {
  const { t } = useTranslation();
  const approvalSteps = issue.approvalTemplate?.flow?.roles ?? [];

  if (issue.approvalStatus === Issue_ApprovalStatus.CHECKING) {
    return (
      <span className="shrink-0 mt-1 inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600">
        {t("custom-approval.issue-review.generating-approval-flow")}
      </span>
    );
  }

  const progressText = t("issue.table.approval-progress", {
    approved: issue.approvers.length,
    total: approvalSteps.length,
  });

  if (approvalSteps.length > 0) {
    const status = issue.approvalStatus;
    if (status === Issue_ApprovalStatus.APPROVED) {
      return (
        <div className="shrink-0 flex flex-row sm:flex-col items-center sm:items-end gap-x-1.5 sm:gap-x-0 mt-1">
          <span className="inline-flex items-center rounded-full bg-success/10 text-success px-2 py-0.5 text-xs">
            {t("issue.table.approved")}
          </span>
          <span className="text-xs text-control-light whitespace-nowrap sm:mt-1">
            {progressText}
          </span>
        </div>
      );
    }
    if (status === Issue_ApprovalStatus.REJECTED) {
      return (
        <div className="shrink-0 flex flex-row sm:flex-col items-center sm:items-end gap-x-1.5 sm:gap-x-0 mt-1">
          <span className="inline-flex items-center rounded-full bg-warning/10 text-warning px-2 py-0.5 text-xs">
            {t("common.rejected")}
          </span>
          <span className="text-xs text-control-light whitespace-nowrap sm:mt-1">
            {progressText}
          </span>
        </div>
      );
    }
    if (status === Issue_ApprovalStatus.PENDING) {
      const currentRoleIndex = issue.approvers.length;
      const role = approvalSteps[currentRoleIndex];
      const roleName = role ? displayRoleTitle(role) : "";
      return (
        <div className="shrink-0 flex flex-row sm:flex-col items-center sm:items-end gap-x-1.5 sm:gap-x-0 mt-1">
          <span className="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600">
            {progressText}
          </span>
          {roleName && (
            <span className="text-xs text-control-light whitespace-nowrap sm:mt-1">
              {t("issue.table.waiting-role", { role: roleName })}
            </span>
          )}
        </div>
      );
    }
  }

  // No approval required
  return (
    <span className="shrink-0 mt-1 inline-flex items-center rounded-full bg-gray-50 px-2 py-0.5 text-xs text-gray-500">
      {t("custom-approval.approval-flow.skip")}
    </span>
  );
}

// ---------------------------------------------------------------------------
// IssueListItem
// ---------------------------------------------------------------------------

function IssueListItem({
  issue,
  highlightText = "",
}: {
  issue: Issue;
  highlightText?: string;
}) {
  const { t } = useTranslation();
  const userStore = useUserStore();
  const projectStore = useProjectV1Store();

  const creator =
    userStore.getUserByIdentifier(issue.creator) || unknownUser(issue.creator);

  const issueProject = useVueState(() =>
    projectStore.getProjectByName(
      `${projectNamePrefix}${extractProjectResourceName(issue.name)}`
    )
  );

  const createTimeTs = Math.floor(
    getTimeForPbTimestampProtoEs(issue.createTime, 0) / 1000
  );

  const issueUrl = useMemo(() => {
    const issueRoute = getIssueRoute(issue);
    return router.resolve({
      name: issueRoute.name,
      params: issueRoute.params,
    }).fullPath;
  }, [issue.name]);

  const onRowClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.ctrlKey || e.metaKey) {
        window.open(issueUrl, "_blank");
      } else {
        router.push(issueUrl);
      }
    },
    [issueUrl]
  );

  // Labels
  const validLabels = useMemo(() => {
    if (!issueProject?.issueLabels) return [];
    const pool = new Set(
      issueProject.issueLabels.map((l: { value: string }) => l.value)
    );
    const validValues = issue.labels.filter((l) => pool.has(l));
    return issueProject.issueLabels.filter((l: { value: string }) =>
      validValues.includes(l.value)
    );
  }, [issue.labels, issueProject]);

  // Highlight
  const highlightWords = useMemo(
    () => (highlightText ? highlightText.toLowerCase().split(" ") : []),
    [highlightText]
  );

  const highlightedTitle = useMemo(
    () =>
      getHighlightHTMLByRegExp(
        issue.title,
        highlightWords,
        false,
        "bg-yellow-100"
      ),
    [issue.title, highlightWords]
  );

  const highlightedDescription = useMemo(
    () =>
      getHighlightHTMLByRegExp(
        issue.description,
        highlightWords,
        false,
        "bg-yellow-100"
      ),
    [issue.description, highlightWords]
  );

  const expanded =
    highlightText &&
    issue.description &&
    highlightWords.some((word) =>
      issue.description.toLowerCase().includes(word)
    );

  return (
    <div
      className="flex items-start gap-x-2 px-3 sm:px-4 py-3 cursor-pointer border-b border-gray-100 hover:bg-gray-50"
      onClick={onRowClick}
    >
      {/* Left: issue content */}
      <div className="flex-1 min-w-0 flex flex-col sm:flex-row sm:items-start sm:gap-x-2">
        <div className="flex-1 min-w-0">
          {/* Line 1: status icon + title + labels */}
          <div className="flex items-center gap-x-1.5">
            <div className="h-6 flex justify-center items-center">
              <IssueStatusIcon status={issue.status} />
            </div>
            {issue.title ? (
              <a
                href={issueUrl}
                className="font-medium text-main text-base truncate hover:underline"
                onClick={(e) => e.stopPropagation()}
                dangerouslySetInnerHTML={{ __html: highlightedTitle }}
              />
            ) : (
              <a
                href={issueUrl}
                className="font-medium text-base truncate hover:underline italic text-gray-400"
                onClick={(e) => e.stopPropagation()}
              >
                {t("common.untitled")}
              </a>
            )}
            <RiskLevelIcon riskLevel={issue.riskLevel} />
            {validLabels.map((label: { value: string; color: string }) => (
              <span
                key={label.value}
                className="inline-flex items-center gap-x-1 px-1.5 py-0.5 rounded-xs text-xs whitespace-nowrap border shrink-0"
              >
                <span
                  className="w-2.5 h-2.5 rounded-sm shrink-0"
                  style={{ backgroundColor: label.color }}
                />
                {label.value}
              </span>
            ))}
          </div>
          {/* Line 2: metadata */}
          <div className="flex items-center flex-wrap gap-x-1 text-xs text-control-light mt-1">
            <span className="opacity-80">#{extractIssueUID(issue.name)}</span>
            <span>&middot;</span>
            {t("common.created")}
            <Tooltip content={formatAbsoluteDateTime(createTimeTs * 1000)}>
              <span>{humanizeTs(createTimeTs)}</span>
            </Tooltip>
            <span>&middot;</span>
            <a
              className="hover:underline"
              href="#"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                router.push({
                  name: WORKSPACE_ROUTE_USER_PROFILE,
                  params: { principalEmail: creator.email },
                });
              }}
            >
              {creator.title}
            </a>
          </div>
          {/* Expanded description for search highlights */}
          {expanded && (
            <div
              className="mt-2 max-h-80 overflow-auto whitespace-pre-wrap break-all text-sm text-control-light"
              dangerouslySetInnerHTML={{
                __html: highlightedDescription,
              }}
            />
          )}
        </div>

        {/* Right: approval status tag */}
        <IssueApprovalStatusTag issue={issue} />
      </div>
    </div>
  );
}
