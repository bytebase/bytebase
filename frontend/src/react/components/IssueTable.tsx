import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { ArrowUpDown, Calendar, Check, Loader2, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/connect";
import {
  AdvancedSearch,
  type ScopeOption,
  type SearchParams,
  type ValueOption,
} from "@/react/components/AdvancedSearch";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_USER_PROFILE } from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useCurrentUserV1,
  useProjectV1Store,
  useUserStore,
} from "@/store";
import {
  getTimeForPbTimestampProtoEs,
  isValidProjectName,
  unknownUser,
} from "@/types";
import { RiskLevel } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  BatchUpdateIssuesStatusRequestSchema,
  Issue_ApprovalStatus,
  Issue_Type,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Label } from "@/types/proto-es/v1/project_service_pb";
import {
  displayRoleTitle,
  extractIssueUID,
  extractProjectResourceName,
  getHighlightHTMLByRegExp,
  getIssueRoute,
  getTsRangeFromSearchParams,
  getValueFromSearchParams,
  getValuesFromSearchParams,
  projectOfIssue,
  upsertScope,
  type SearchParams as VueSearchParams,
  type SearchScope as VueSearchScope,
} from "@/utils";

// ===========================================================================
// IssueSearchBar (AdvancedSearch + TimeRange + Sort)
// ===========================================================================

export function IssueSearchBar({
  params,
  onParamsChange,
  orderBy,
  onOrderByChange,
  scopeOptions,
}: {
  params: SearchParams;
  onParamsChange: (p: SearchParams) => void;
  orderBy: string;
  onOrderByChange: (v: string) => void;
  scopeOptions: ScopeOption[];
}) {
  return (
    <div className="flex items-center md:gap-2">
      <div className="flex-1 min-w-0">
        <AdvancedSearch
          params={params}
          onParamsChange={onParamsChange}
          scopeOptions={scopeOptions}
        />
      </div>
      <TimeRangePicker params={params} onParamsChange={onParamsChange} />
      <IssueSortDropdown orderBy={orderBy} onOrderByChange={onOrderByChange} />
    </div>
  );
}

// ===========================================================================
// TimeRangePicker
// ===========================================================================

function TimeRangePicker({
  params,
  onParamsChange,
}: {
  params: SearchParams;
  onParamsChange: (p: SearchParams) => void;
}) {
  const { t } = useTranslation();
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  useClickOutside(containerRef, open, () => setOpen(false));

  const timeRange = useMemo(
    () => getTsRangeFromSearchParams(params as VueSearchParams, "created"),
    [params]
  );

  const hasRange = !!timeRange;

  useEffect(() => {
    if (timeRange) {
      setFrom(dayjs(timeRange[0]).format("YYYY-MM-DD"));
      setTo(dayjs(timeRange[1]).format("YYYY-MM-DD"));
    } else {
      setFrom("");
      setTo("");
    }
  }, [timeRange]);

  const applyRange = useCallback(() => {
    const fromTs = from ? dayjs(from).startOf("day").valueOf() : null;
    const toTs = to ? dayjs(to).endOf("day").valueOf() : null;
    const updated = upsertScope({
      params: params as VueSearchParams,
      scopes: {
        id: "created",
        value: fromTs && toTs ? `${fromTs},${toTs}` : "",
      },
    });
    onParamsChange({
      query: updated.query,
      scopes: updated.scopes.map((s) => ({
        id: s.id,
        value: s.value,
        readonly: (s as VueSearchScope & { readonly?: boolean }).readonly,
      })),
    });
    setOpen(false);
  }, [from, to, params, onParamsChange]);

  const clearRange = useCallback(() => {
    const updated = upsertScope({
      params: params as VueSearchParams,
      scopes: { id: "created", value: "" },
    });
    onParamsChange({
      query: updated.query,
      scopes: updated.scopes.map((s) => ({
        id: s.id,
        value: s.value,
        readonly: (s as VueSearchScope & { readonly?: boolean }).readonly,
      })),
    });
    setOpen(false);
  }, [params, onParamsChange]);

  return (
    <div ref={containerRef} className="relative">
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          hasRange
            ? "text-accent! hover:text-accent"
            : "text-control-placeholder hover:text-control"
        )}
        onClick={() => setOpen(!open)}
      >
        <Calendar className="w-4 h-4" />
      </Button>
      {open && (
        <div className="absolute right-0 top-full mt-1 z-50 bg-white border border-gray-200 rounded-sm shadow-lg p-3 flex flex-col gap-2 min-w-[240px]">
          <label className="text-xs text-control-light">
            {t("common.from")}
          </label>
          <Input
            type="date"
            value={from}
            max={dayjs().format("YYYY-MM-DD")}
            onChange={(e) => setFrom(e.target.value)}
          />
          <label className="text-xs text-control-light">{t("common.to")}</label>
          <Input
            type="date"
            value={to}
            max={dayjs().format("YYYY-MM-DD")}
            onChange={(e) => setTo(e.target.value)}
          />
          <div className="flex items-center gap-x-2 mt-1">
            <Button size="sm" onClick={applyRange} disabled={!from || !to}>
              {t("common.confirm")}
            </Button>
            {hasRange && (
              <Button variant="ghost" size="sm" onClick={clearRange}>
                {t("common.clear")}
              </Button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// ===========================================================================
// IssueSortDropdown
// ===========================================================================

function IssueSortDropdown({
  orderBy,
  onOrderByChange,
}: {
  orderBy: string;
  onOrderByChange: (v: string) => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  useClickOutside(containerRef, open, () => setOpen(false));

  const sortOptions = useMemo(
    () => [
      {
        label: t("issue.sort.created"),
        children: [
          { key: "create_time desc", label: t("issue.sort.descending") },
          { key: "create_time asc", label: t("issue.sort.ascending") },
        ],
      },
      {
        label: t("issue.sort.updated"),
        children: [
          { key: "update_time desc", label: t("issue.sort.descending") },
          { key: "update_time asc", label: t("issue.sort.ascending") },
        ],
      },
    ],
    [t]
  );

  const handleSelect = (key: string) => {
    onOrderByChange(key === orderBy ? "" : key);
    setOpen(false);
  };

  return (
    <div ref={containerRef} className="relative">
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          orderBy
            ? "text-accent! hover:text-accent"
            : "text-control-placeholder hover:text-control"
        )}
        onClick={() => setOpen(!open)}
      >
        <ArrowUpDown className="w-4 h-4" />
        <span className="hidden md:inline ml-1">{t("issue.sort.sort")}</span>
      </Button>
      {open && (
        <div className="absolute right-0 top-full mt-1 z-50 bg-white border border-gray-200 rounded-sm shadow-lg min-w-[180px] py-1">
          {sortOptions.map((group) => (
            <div key={group.label}>
              <div className="px-3 py-1.5 text-xs text-control-light font-medium">
                {group.label}
              </div>
              {group.children.map((opt) => (
                <button
                  key={opt.key}
                  type="button"
                  className="w-full text-left px-3 py-1.5 text-sm hover:bg-gray-50 flex items-center gap-x-2"
                  onClick={() => handleSelect(opt.key)}
                >
                  <Check
                    className={cn(
                      "w-3 h-3",
                      orderBy === opt.key ? "text-accent" : "text-transparent"
                    )}
                  />
                  {opt.label}
                </button>
              ))}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ===========================================================================
// PresetButtons
// ===========================================================================

export function PresetButtons({
  params,
  onParamsChange,
}: {
  params: SearchParams;
  onParamsChange: (p: SearchParams) => void;
}) {
  const { t } = useTranslation();
  const currentUser = useCurrentUserV1();
  const me = useVueState(() => currentUser.value);

  type PresetValue = "WAITING_APPROVAL" | "OPEN" | "CLOSED" | "ALL";

  const presets: { value: PresetValue; label: string }[] = useMemo(
    () => [
      { value: "WAITING_APPROVAL", label: t("issue.waiting-approval") },
      { value: "OPEN", label: t("issue.table.open") },
      { value: "CLOSED", label: t("issue.table.closed") },
      { value: "ALL", label: t("common.all") },
    ],
    [t]
  );

  const isActive = useCallback(
    (preset: PresetValue): boolean => {
      const vp = params as VueSearchParams;
      if (preset === "WAITING_APPROVAL") {
        return (
          getValueFromSearchParams(vp, "approval") ===
          Issue_ApprovalStatus[Issue_ApprovalStatus.PENDING]
        );
      }
      if (preset === "OPEN") {
        const status = getValueFromSearchParams(vp, "status");
        const approval = getValueFromSearchParams(vp, "approval");
        return status === IssueStatus[IssueStatus.OPEN] && !approval;
      }
      const statuses = getValuesFromSearchParams(vp, "status");
      if (preset === "CLOSED") {
        return (
          statuses.includes(IssueStatus[IssueStatus.DONE]) &&
          !statuses.includes(IssueStatus[IssueStatus.OPEN])
        );
      }
      if (preset === "ALL") {
        return statuses.length === 0;
      }
      return false;
    },
    [params]
  );

  const activePreset = presets.find((p) => isActive(p.value))?.value ?? "";

  const selectPreset = useCallback(
    (preset: PresetValue) => {
      const myEmail = me?.email ?? "";
      const readonlyScopes = params.scopes.filter((s) => s.readonly);
      let newParams: VueSearchParams = {
        query: "",
        scopes: [...readonlyScopes],
      };

      if (preset === "WAITING_APPROVAL") {
        newParams = upsertScope({
          params: newParams,
          scopes: [
            { id: "status", value: IssueStatus[IssueStatus.OPEN] },
            {
              id: "approval",
              value: Issue_ApprovalStatus[Issue_ApprovalStatus.PENDING],
            },
            { id: "current-approver", value: myEmail },
          ],
        });
      } else if (preset === "OPEN") {
        newParams = upsertScope({
          params: newParams,
          scopes: [{ id: "status", value: IssueStatus[IssueStatus.OPEN] }],
        });
      } else if (preset === "CLOSED") {
        newParams = upsertScope({
          params: newParams,
          scopes: [
            { id: "status", value: IssueStatus[IssueStatus.DONE] },
            { id: "status", value: IssueStatus[IssueStatus.CANCELED] },
          ],
          allowMultiple: true,
        });
      }

      onParamsChange({
        query: newParams.query,
        scopes: newParams.scopes.map((s) => ({
          id: s.id,
          value: s.value,
          readonly: (s as VueSearchScope & { readonly?: boolean }).readonly,
        })),
      });
    },
    [params, me, onParamsChange]
  );

  return (
    <div className="shrink-0 flex border-b border-control-border">
      {presets.map((preset) => (
        <button
          key={preset.value}
          type="button"
          className={cn(
            "px-3 py-1.5 text-sm font-medium border-b-2 -mb-px transition-colors",
            activePreset === preset.value
              ? "border-accent text-accent"
              : "border-transparent text-control-light hover:text-control"
          )}
          onClick={() => selectPreset(preset.value)}
        >
          {preset.label}
        </button>
      ))}
    </div>
  );
}

// ===========================================================================
// useIssueSearchScopeOptions
// ===========================================================================

export function useIssueSearchScopeOptions(
  projectName?: string
): ScopeOption[] {
  const { t } = useTranslation();
  const userStore = useUserStore();
  const projectStore = useProjectV1Store();
  const currentUser = useCurrentUserV1();
  const me = useVueState(() => currentUser.value);

  const [projectLabels, setProjectLabels] = useState<Label[]>([]);

  useEffect(() => {
    if (!projectName || !isValidProjectName(projectName)) return;
    projectStore.getOrFetchProjectByName(projectName).then((project) => {
      const labels = new Map<string, Label>();
      for (const label of project.issueLabels) {
        labels.set(`${label.value}-${label.color}`, label);
      }
      setProjectLabels([...labels.values()]);
    });
  }, [projectName, projectStore]);

  const searchPrincipal = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      const resp = await userStore.fetchUserList({
        pageSize: 50,
        filter: { query: keyword },
      });
      return resp.users.map<ValueOption>((user) => ({
        value: user.email,
        keywords: [user.email, user.title],
        render: () => (
          <div className="flex items-center gap-x-1">
            <span>{user.title}</span>
            {user.name === me?.name && (
              <span className="text-xs text-control-light">
                ({t("common.you")})
              </span>
            )}
          </div>
        ),
      }));
    },
    [userStore, me, t]
  );

  return useMemo(
    (): ScopeOption[] => [
      {
        id: "status",
        title: t("common.status"),
        description: t("issue.advanced-search.scope.status.description"),
        allowMultiple: true,
        options: [
          {
            value: IssueStatus[IssueStatus.OPEN],
            keywords: ["open"],
            render: () => <span>{t("issue.table.open")}</span>,
          },
          {
            value: IssueStatus[IssueStatus.DONE],
            keywords: ["closed", "done"],
            render: () => <span>{t("common.approved")}</span>,
          },
          {
            value: IssueStatus[IssueStatus.CANCELED],
            keywords: ["closed", "canceled"],
            render: () => <span>{t("common.closed")}</span>,
          },
        ],
      },
      {
        id: "approval",
        title: t("issue.advanced-search.scope.approval.title"),
        description: t("issue.advanced-search.scope.approval.description"),
        options: [
          {
            value: Issue_ApprovalStatus[Issue_ApprovalStatus.CHECKING],
            keywords: ["checking"],
            render: () => (
              <span>
                {t("issue.advanced-search.scope.approval.value.checking")}
              </span>
            ),
          },
          {
            value: Issue_ApprovalStatus[Issue_ApprovalStatus.PENDING],
            keywords: ["pending"],
            render: () => (
              <span>
                {t("issue.advanced-search.scope.approval.value.pending")}
              </span>
            ),
          },
          {
            value: Issue_ApprovalStatus[Issue_ApprovalStatus.APPROVED],
            keywords: ["approved", "done"],
            render: () => (
              <span>
                {t("issue.advanced-search.scope.approval.value.approved")}
              </span>
            ),
          },
          {
            value: Issue_ApprovalStatus[Issue_ApprovalStatus.REJECTED],
            keywords: ["rejected"],
            render: () => (
              <span>
                {t("issue.advanced-search.scope.approval.value.rejected")}
              </span>
            ),
          },
          {
            value: Issue_ApprovalStatus[Issue_ApprovalStatus.SKIPPED],
            keywords: ["skipped"],
            render: () => (
              <span>
                {t("issue.advanced-search.scope.approval.value.skipped")}
              </span>
            ),
          },
        ],
      },
      {
        id: "issue-type",
        title: t("issue.advanced-search.scope.issue-type.title"),
        description: t("issue.advanced-search.scope.issue-type.description"),
        allowMultiple: true,
        options: [
          {
            value: Issue_Type[Issue_Type.DATABASE_CHANGE],
            keywords: ["database", "change"],
            render: () => (
              <span>
                {t(
                  "issue.advanced-search.scope.issue-type.value.database-change"
                )}
              </span>
            ),
          },
          {
            value: Issue_Type[Issue_Type.ROLE_GRANT],
            keywords: ["grant", "request"],
            render: () => (
              <span>
                {t("issue.advanced-search.scope.issue-type.value.role-grant")}
              </span>
            ),
          },
          {
            value: Issue_Type[Issue_Type.DATABASE_EXPORT],
            keywords: ["database", "export"],
            render: () => (
              <span>
                {t(
                  "issue.advanced-search.scope.issue-type.value.database-export"
                )}
              </span>
            ),
          },
          {
            value: Issue_Type[Issue_Type.ACCESS_GRANT],
            keywords: ["access", "grant"],
            render: () => (
              <span>
                {t("issue.advanced-search.scope.issue-type.value.access-grant")}
              </span>
            ),
          },
        ],
      },
      {
        id: "creator",
        title: t("issue.advanced-search.scope.creator.title"),
        description: t("issue.advanced-search.scope.creator.description"),
        onSearch: searchPrincipal,
      },
      {
        id: "current-approver",
        title: t("issue.advanced-search.scope.current-approver.title"),
        description: t(
          "issue.advanced-search.scope.current-approver.description"
        ),
        onSearch: searchPrincipal,
      },
      {
        id: "issue-label",
        title: t("issue.advanced-search.scope.issue-label.title"),
        description: t("issue.advanced-search.scope.issue-label.description"),
        allowMultiple: true,
        options: projectLabels.map((label) => ({
          value: label.value,
          keywords: [label.value],
          render: () => (
            <div className="flex items-center gap-x-2">
              <div
                className="w-4 h-4 rounded-sm"
                style={{ backgroundColor: label.color }}
              />
              {label.value}
            </div>
          ),
        })),
      },
      {
        id: "risk-level",
        title: t("issue.risk-level.self"),
        description: t("issue.risk-level.filter"),
        allowMultiple: true,
        options: [
          {
            value: RiskLevel[RiskLevel.HIGH],
            keywords: ["high"],
            render: () => <span>{t("issue.risk-level.high")}</span>,
          },
          {
            value: RiskLevel[RiskLevel.MODERATE],
            keywords: ["moderate"],
            render: () => <span>{t("issue.risk-level.moderate")}</span>,
          },
          {
            value: RiskLevel[RiskLevel.LOW],
            keywords: ["low"],
            render: () => <span>{t("issue.risk-level.low")}</span>,
          },
        ],
      },
    ],
    [t, searchPrincipal, projectLabels]
  );
}

// ===========================================================================
// IssueListItem
// ===========================================================================

export function IssueListItem({
  issue,
  selected,
  onToggleSelection,
  highlightText = "",
  showProject = false,
}: {
  issue: Issue;
  selected: boolean;
  onToggleSelection: () => void;
  highlightText?: string;
  showProject?: boolean;
}) {
  const { t } = useTranslation();
  const userStore = useUserStore();

  const creator =
    userStore.getUserByIdentifier(issue.creator) || unknownUser(issue.creator);

  const issueProject = useMemo(() => projectOfIssue(issue), [issue]);

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
  const labels = useMemo(() => {
    if (!issueProject?.issueLabels) return [];
    const pool = new Set(issueProject.issueLabels.map((l: Label) => l.value));
    const validValues = issue.labels.filter((l) => pool.has(l));
    return issueProject.issueLabels.filter((l: Label) =>
      validValues.includes(l.value)
    );
  }, [issue.labels, issueProject]);

  // Search highlighting
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
      <input
        type="checkbox"
        className="shrink-0 mt-1 accent-accent"
        checked={selected}
        onChange={(e) => {
          e.stopPropagation();
          onToggleSelection();
        }}
        onClick={(e) => e.stopPropagation()}
      />

      <div className="flex-1 min-w-0 flex flex-col sm:flex-row sm:items-start sm:gap-x-2">
        <div className="flex-1 min-w-0">
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
            {labels.map((label: Label) => (
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
          <div className="flex items-center flex-wrap gap-x-1 text-xs text-control-light mt-1">
            <span className="opacity-80">#{extractIssueUID(issue.name)}</span>
            <span>&middot;</span>
            {t("common.created")}
            <HumanizeTs ts={createTimeTs} />
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
            {showProject && issueProject && (
              <>
                <span>&middot;</span>
                <a
                  className="hover:underline"
                  href="#"
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    router.push({
                      name: PROJECT_V1_ROUTE_DETAIL,
                      params: {
                        projectId: extractProjectResourceName(
                          issueProject.name
                        ),
                      },
                    });
                  }}
                >
                  {issueProject.title}
                </a>
              </>
            )}
          </div>
          {expanded && (
            <div
              className="mt-2 max-h-80 overflow-auto whitespace-pre-wrap break-all text-sm text-control-light"
              dangerouslySetInnerHTML={{ __html: highlightedDescription }}
            />
          )}
        </div>

        <IssueApprovalStatusTag issue={issue} />
      </div>
    </div>
  );
}

// ===========================================================================
// IssueStatusIcon
// ===========================================================================

export function IssueStatusIcon({ status }: { status: IssueStatus }) {
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

// ===========================================================================
// RiskLevelIcon
// ===========================================================================

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

// ===========================================================================
// IssueApprovalStatusTag
// ===========================================================================

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

  return (
    <span className="shrink-0 mt-1 inline-flex items-center rounded-full bg-gray-50 px-2 py-0.5 text-xs text-gray-500">
      {t("custom-approval.approval-flow.skip")}
    </span>
  );
}

// ===========================================================================
// BatchActionBar
// ===========================================================================

export function BatchActionBar({
  issues,
  allSelected,
  onToggleSelectAll,
  onStartAction,
}: {
  issues: Issue[];
  allSelected: boolean;
  onToggleSelectAll: () => void;
  onStartAction: (action: "CLOSE" | "REOPEN") => void;
}) {
  const { t } = useTranslation();

  const statuses = useMemo(() => {
    const s = new Set<IssueStatus>();
    for (const issue of issues) s.add(issue.status);
    return s;
  }, [issues]);

  const canClose =
    !statuses.has(IssueStatus.DONE) && !statuses.has(IssueStatus.CANCELED);
  const canReopen =
    !statuses.has(IssueStatus.OPEN) && !statuses.has(IssueStatus.DONE);

  return (
    <div className="sticky bottom-0 w-full bg-white flex items-center gap-x-2 px-3 sm:px-4 py-2 border-y">
      <input
        type="checkbox"
        checked={allSelected}
        ref={(el) => {
          if (el) el.indeterminate = !allSelected;
        }}
        onChange={onToggleSelectAll}
        className="accent-accent"
      />
      <span className="text-sm text-control-light">
        {issues.length} {t("common.selected")}
      </span>
      <Tooltip
        content={
          !canClose
            ? t("issue.batch-transition.not-allowed-tips", {
                operation: t("issue.batch-transition.closed"),
              })
            : undefined
        }
      >
        <Button
          variant="outline"
          size="sm"
          disabled={!canClose}
          onClick={() => onStartAction("CLOSE")}
        >
          {t("issue.batch-transition.close")}
        </Button>
      </Tooltip>
      <Tooltip
        content={
          !canReopen
            ? t("issue.batch-transition.not-allowed-tips", {
                operation: t("issue.batch-transition.reopened"),
              })
            : undefined
        }
      >
        <Button
          variant="outline"
          size="sm"
          disabled={!canReopen}
          onClick={() => onStartAction("REOPEN")}
        >
          {t("issue.batch-transition.reopen")}
        </Button>
      </Tooltip>
    </div>
  );
}

// ===========================================================================
// BatchIssueStatusActionDrawer
// ===========================================================================

export function BatchIssueStatusActionDrawer({
  issues,
  action,
  onClose,
  onUpdated,
}: {
  issues: Issue[];
  action: "CLOSE" | "REOPEN" | undefined;
  onClose: () => void;
  onUpdated: () => void;
}) {
  const { t } = useTranslation();
  const [comment, setComment] = useState("");
  const [loading, setLoading] = useState(false);

  useEscapeKey(!!action, onClose);

  useEffect(() => {
    if (action) setComment("");
  }, [action]);

  const handleConfirm = async () => {
    if (!action || loading) return;
    setLoading(true);
    try {
      const statusMap: Record<string, IssueStatus> = {
        CLOSE: IssueStatus.CANCELED,
        REOPEN: IssueStatus.OPEN,
      };
      const request = create(BatchUpdateIssuesStatusRequestSchema, {
        parent: "projects/-",
        issues: issues.map((i) => i.name),
        status: statusMap[action],
        reason: comment,
      });
      await issueServiceClientConnect.batchUpdateIssuesStatus(request);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      onUpdated();
    } finally {
      setLoading(false);
    }
  };

  if (!action) return null;

  const title =
    action === "CLOSE"
      ? t("issue.batch-transition.close")
      : t("issue.batch-transition.reopen");

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="ml-auto relative bg-white w-[calc(100vw-8rem)] lg:w-160 max-w-[calc(100vw-8rem)] h-full shadow-lg flex flex-col">
        <div className="px-6 py-4 border-b border-control-border flex items-center justify-between">
          <span className="text-lg font-semibold">{title}</span>
          <button
            className="p-1 hover:bg-control-bg rounded-xs"
            onClick={onClose}
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-y-4">
          <div className="flex flex-col gap-y-1">
            <div className="font-medium text-control">{t("common.issues")}</div>
            <ul
              className={cn(
                "flex flex-col text-sm text-control-light",
                issues.length > 1 && "list-disc pl-4"
              )}
            >
              {issues.map((issue) => (
                <li key={issue.name}>{issue.title}</li>
              ))}
            </ul>
          </div>

          <div className="flex flex-col gap-y-1">
            <div className="font-medium text-control">
              {t("common.comment")}
            </div>
            <textarea
              className="w-full border border-control-border rounded-sm px-3 py-2 text-sm focus:outline-none focus:border-accent min-h-[6rem] resize-y"
              value={comment}
              placeholder={t("issue.leave-a-comment")}
              onChange={(e) => setComment(e.target.value)}
            />
          </div>
        </div>

        <div className="px-6 py-4 border-t border-control-border flex items-center justify-end gap-x-2">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button disabled={loading} onClick={handleConfirm}>
            {loading && <Loader2 className="w-4 h-4 mr-1 animate-spin" />}
            {t("common.confirm")}
          </Button>
        </div>
      </div>
    </div>
  );
}
