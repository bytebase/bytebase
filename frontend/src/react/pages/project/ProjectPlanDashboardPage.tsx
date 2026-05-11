import { create as createProto } from "@bufbuild/protobuf";
import {
  AlertCircle,
  CheckCircle,
  Database as DatabaseIcon,
  FolderTree,
  Loader2,
  Plus,
  X,
  XCircle,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { applyPlanTitleToQuery } from "@/components/Plan/logic/title";
import {
  AdvancedSearch,
  type ScopeOption,
  type SearchParams,
  type ValueOption,
} from "@/react/components/AdvancedSearch";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import { SearchInput } from "@/react/components/ui/search-input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { useSessionPageSize } from "@/react/hooks/useSessionPageSize";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useDBGroupStore,
  useEnvironmentV1Store,
  useProjectV1Store,
  useUIStateStore,
  useUserStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  buildPlanFindBySearchParams,
  usePlanStore,
} from "@/store/modules/v1/plan";
import {
  formatEnvironmentName,
  getTimeForPbTimestampProtoEs,
  isValidDatabaseGroupName,
  unknownUser,
} from "@/types";
import { ApprovalStatus, State } from "@/types/proto-es/v1/common_pb";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import {
  type Database,
  SyncStatus,
} from "@/types/proto-es/v1/database_service_pb";
import type {
  Plan,
  Plan_RolloutStageSummary,
  Plan_Spec,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractDatabaseGroupName,
  extractDatabaseResourceName,
  extractPlanUID,
  formatAbsoluteDateTime,
  generatePlanTitle,
  getDatabaseEnvironment,
  getDefaultPagination,
  getInstanceResource,
  humanizeTs,
  type SearchParams as VueSearchParams,
} from "@/utils";
import { extractStageUID } from "@/utils/v1/issue/rollout";

// Task status priority order for determining rollout stage status
const TASK_STATUS_FILTERS: Task_Status[] = [
  Task_Status.RUNNING,
  Task_Status.FAILED,
  Task_Status.PENDING,
  Task_Status.NOT_STARTED,
  Task_Status.CANCELED,
  Task_Status.DONE,
  Task_Status.SKIPPED,
];

export function ProjectPlanDashboardPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const planStore = usePlanStore();
  const projectStore = useProjectV1Store();
  const userStore = useUserStore();
  const uiStateStore = useUIStateStore();
  const currentUser = useCurrentUserV1();

  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));
  const me = useVueState(() => currentUser.value);

  const [showAddSpecDrawer, setShowAddSpecDrawer] = useState(false);

  // Hint dismissal
  const HINT_KEY = "plan.hint-dismissed";
  const hideHint = useVueState(() => uiStateStore.getIntroStateByKey(HINT_KEY));
  const dismissHint = useCallback(() => {
    uiStateStore.saveIntroStateByKey({ key: HINT_KEY, newState: true });
  }, [uiStateStore]);

  // Search
  const defaultSearchParams = useCallback(
    (): SearchParams => ({
      query: "",
      scopes: [{ id: "state", value: "ACTIVE" }],
    }),
    []
  );

  const [searchParams, setSearchParams] =
    useState<SearchParams>(defaultSearchParams);

  const didMountRef = useRef(false);
  useEffect(() => {
    if (!didMountRef.current) {
      didMountRef.current = true;
      return;
    }
    setSearchParams(defaultSearchParams());
  }, [defaultSearchParams, projectId]);

  // Scope options
  const scopeOptions: ScopeOption[] = useMemo(
    () => [
      {
        id: "state",
        title: t("common.state"),
        description: t("issue.advanced-search.scope.state.description"),
        options: [
          {
            value: "ACTIVE",
            keywords: ["active", "ACTIVE"],
            render: () => <span>{t("common.active")}</span>,
          },
          {
            value: "DELETED",
            keywords: ["deleted", "DELETED", "closed", "CLOSED"],
            render: () => <span>{t("common.closed")}</span>,
          },
        ],
      },
      {
        id: "creator",
        title: t("issue.advanced-search.scope.creator.title"),
        description: t("issue.advanced-search.scope.creator.description"),
        search: async ({
          keyword,
          nextPageToken,
        }: {
          keyword: string;
          nextPageToken?: string;
        }) => {
          const resp = await userStore.fetchUserList({
            pageToken: nextPageToken,
            pageSize: getDefaultPagination(),
            filter: { query: keyword },
          });
          return {
            nextPageToken: resp.nextPageToken,
            options: resp.users.map<ValueOption>((user) => ({
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
            })),
          };
        },
      },
    ],
    [t, userStore, me]
  );

  const [canCreate] = usePermissionCheck(["bb.plans.create"], project);

  // Build plan filter
  const planFilter = useMemo(() => {
    const merged: VueSearchParams = {
      query: searchParams.query.trim().toLowerCase(),
      scopes: [
        ...searchParams.scopes.map((s) => ({ id: s.id, value: s.value })),
        { id: "project", value: projectId },
      ],
    };
    return buildPlanFindBySearchParams(merged, {
      specType: "change_database_config",
    });
  }, [searchParams, projectId]);

  const fetchPlanList = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const { nextPageToken, plans } = await planStore.listPlans({
        find: planFilter,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
      });
      return { list: plans, nextPageToken };
    },
    [planStore, planFilter]
  );

  const paged = usePagedData<Plan>({
    sessionKey: `bb.${projectName}.plan-table`,
    fetchList: fetchPlanList,
  });

  // Handle spec created from AddSpecDrawer
  const handleSpecCreated = useCallback(
    async (spec: Plan_Spec) => {
      if (!project) return;

      const template = "bb.plan.change-database";
      const targets =
        spec.config?.case === "changeDatabaseConfig"
          ? [...(spec.config.value.targets ?? [])]
          : [];
      const isDatabaseGroup = targets.every((target) =>
        isValidDatabaseGroupName(target)
      );
      const query: Record<string, string> = { template };

      // Check if the spec has a sheet with content
      if (
        spec.config?.case === "changeDatabaseConfig" &&
        spec.config.value.sheet
      ) {
        // The sheet is local (not yet created on server), so we skip content handling here.
        // The plan creation page will handle it.
      }

      if (isDatabaseGroup) {
        const databaseGroupName = targets[0];
        if (!databaseGroupName) return;
        query.databaseGroupName = databaseGroupName;
        applyPlanTitleToQuery(query, project, () =>
          generatePlanTitle(template, [
            extractDatabaseGroupName(databaseGroupName),
          ])
        );
      } else {
        query.databaseList = targets.join(",");
        applyPlanTitleToQuery(query, project, () =>
          generatePlanTitle(
            template,
            targets.map((db) => {
              const { databaseName } = extractDatabaseResourceName(db);
              return databaseName;
            })
          )
        );
      }

      await router.push({
        name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
        params: {
          projectId,
          planId: "create",
          specId: "placeholder",
        },
        query,
      });
    },
    [project, projectId]
  );

  return (
    <div className="py-4 w-full flex flex-col">
      <div className="px-4 flex flex-col gap-y-2 pb-2">
        {!hideHint && (
          <DismissibleAlert onClose={dismissHint}>
            {t("plan.subtitle")}
          </DismissibleAlert>
        )}
        <div className="w-full flex flex-col lg:flex-row items-start lg:items-center justify-between gap-2">
          <div className="w-full flex flex-1 items-center justify-between gap-x-2">
            <AdvancedSearch
              params={searchParams}
              onParamsChange={setSearchParams}
              scopeOptions={scopeOptions}
            />
            <PermissionGuard
              permissions={["bb.plans.create"]}
              project={project}
            >
              <Button
                disabled={!canCreate}
                onClick={() => setShowAddSpecDrawer(true)}
              >
                <Plus className="size-4 mr-1" />
                {t("plan.new-plan")}
              </Button>
            </PermissionGuard>
          </div>
        </div>
      </div>

      {/* Plan list */}
      <div className="mt-2">
        {paged.isLoading ? (
          <div className="flex justify-center py-8 text-control-light">
            <Loader2 className="size-5 animate-spin" />
          </div>
        ) : paged.dataList.length === 0 ? (
          <div className="flex justify-center py-8 text-control-light">
            {t("common.no-data")}
          </div>
        ) : (
          <PlanTable plans={paged.dataList} projectId={projectId} />
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

      <AddSpecDrawer
        open={showAddSpecDrawer}
        onClose={() => setShowAddSpecDrawer(false)}
        projectName={projectName}
        onCreated={handleSpecCreated}
        title={t("plan.new-plan")}
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
      <Info className="size-5 shrink-0 mt-0.5" />
      <div className="flex-1">{children}</div>
      <button
        className="p-0.5 hover:bg-accent/10 rounded-xs shrink-0"
        onClick={onClose}
      >
        <X className="size-4" />
      </button>
    </div>
  );
}

function Info(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      <circle cx="12" cy="12" r="10" />
      <path d="M12 16v-4" />
      <path d="M12 8h.01" />
    </svg>
  );
}

// ---------------------------------------------------------------------------
// PlanTable
// ---------------------------------------------------------------------------

function PlanTable({ plans, projectId }: { plans: Plan[]; projectId: string }) {
  const { t } = useTranslation();

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm min-w-[1000px]">
        <thead>
          <tr className="border-b text-left text-control-light">
            <th className="py-2 px-4 font-medium min-w-80">
              {t("issue.table.name")}
            </th>
            <th className="py-2 px-4 font-medium w-50">
              {t("plan.checks.self")}
            </th>
            <th className="py-2 px-4 font-medium w-35">
              {t("plan.navigator.review")}
            </th>
            <th className="py-2 px-4 font-medium w-65">
              {t("rollout.stage.self", { count: 2 })}
            </th>
            <th className="py-2 px-4 font-medium w-38">
              {t("issue.table.updated")}
            </th>
            <th className="py-2 px-4 font-medium w-38">
              {t("issue.table.creator")}
            </th>
          </tr>
        </thead>
        <tbody>
          {plans.map((plan) => (
            <PlanRow key={plan.name} plan={plan} projectId={projectId} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ---------------------------------------------------------------------------
// PlanRow
// ---------------------------------------------------------------------------

function PlanRow({ plan, projectId }: { plan: Plan; projectId: string }) {
  const { t } = useTranslation();
  const userStore = useUserStore();
  const environmentStore = useEnvironmentV1Store();

  const isDeleted = plan.state === State.DELETED;
  const showDraftTag = plan.issue === "" && !plan.hasRollout;

  const creator =
    userStore.getUserByIdentifier(plan.creator) || unknownUser(plan.creator);

  const updateTimeTs = Math.floor(
    getTimeForPbTimestampProtoEs(plan.updateTime, 0) / 1000
  );

  const planUrl = useMemo(() => {
    return router.resolve({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: {
        projectId,
        planId: extractPlanUID(plan.name),
      },
    }).fullPath;
  }, [plan.name, projectId]);

  const onRowClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.ctrlKey || e.metaKey) {
        window.open(planUrl, "_blank");
      } else {
        router.push(planUrl);
      }
    },
    [planUrl]
  );

  // Approval status
  const approvalTag = useMemo(() => {
    if (plan.issue === "") return undefined;
    switch (plan.approvalStatus) {
      case ApprovalStatus.CHECKING:
        return { label: t("task.checking"), variant: "default" as const };
      case ApprovalStatus.APPROVED:
        return {
          label: t("issue.table.approved"),
          variant: "success" as const,
        };
      case ApprovalStatus.SKIPPED:
        return { label: t("common.skipped"), variant: "default" as const };
      case ApprovalStatus.REJECTED:
        return { label: t("common.rejected"), variant: "warning" as const };
      case ApprovalStatus.PENDING:
        return { label: t("common.under-review"), variant: "info" as const };
      default:
        return undefined;
    }
  }, [plan.approvalStatus, plan.issue, t]);

  // Plan check status summary
  const checkSummary = useMemo(() => {
    const statusCount = plan.planCheckRunStatusCount || {};
    const running = statusCount["RUNNING"] || 0;
    const success = statusCount["SUCCESS"] || 0;
    const warning = statusCount["WARNING"] || 0;
    const error = (statusCount["ERROR"] || 0) + (statusCount["FAILED"] || 0);
    return { running, success, warning, error };
  }, [plan.planCheckRunStatusCount]);

  const hasAnyCheck =
    checkSummary.running +
      checkSummary.success +
      checkSummary.warning +
      checkSummary.error >
    0;

  // Rollout stages
  const getRolloutStageStatus = (
    summary: Plan_RolloutStageSummary
  ): Task_Status => {
    for (const status of TASK_STATUS_FILTERS) {
      if (summary.taskStatusCounts.some((item) => item.status === status)) {
        return status;
      }
    }
    return Task_Status.STATUS_UNSPECIFIED;
  };

  return (
    <tr
      className={cn(
        "border-b cursor-pointer hover:bg-control-bg",
        isDeleted && "opacity-60"
      )}
      onClick={onRowClick}
    >
      {/* Title */}
      <td className="py-2 px-4">
        <div className="flex items-center gap-x-2 overflow-hidden">
          <span className="whitespace-nowrap text-control opacity-60">
            {extractPlanUID(plan.name)}
          </span>
          {plan.title ? (
            <span className="truncate normal-nums">{plan.title}</span>
          ) : (
            <span className="opacity-60 italic">{t("common.untitled")}</span>
          )}
          {isDeleted && (
            <span className="inline-flex items-center rounded-full bg-warning/10 text-warning px-2 py-0.5 text-xs shrink-0">
              {t("common.closed")}
            </span>
          )}
          {showDraftTag && !isDeleted && (
            <span className="inline-flex items-center rounded-full bg-control-bg text-control-light px-2 py-0.5 text-xs shrink-0">
              {t("common.draft")}
            </span>
          )}
        </div>
      </td>

      {/* Checks */}
      <td className="py-2 px-4">
        {hasAnyCheck ? (
          <div className="flex items-center gap-3 flex-wrap">
            {checkSummary.running > 0 && (
              <div className="flex items-center gap-1 text-control">
                <Loader2 className="size-4 animate-spin" />
                <span>{t("task.status.running")}</span>
              </div>
            )}
            {checkSummary.error > 0 && (
              <div className="flex items-center gap-1 text-error">
                <XCircle className="size-4" />
                <span>{checkSummary.error}</span>
              </div>
            )}
            {checkSummary.warning > 0 && (
              <div className="flex items-center gap-1 text-warning">
                <AlertCircle className="size-4" />
                <span>{checkSummary.warning}</span>
              </div>
            )}
            {checkSummary.success > 0 && (
              <div className="flex items-center gap-1 text-success">
                <CheckCircle className="size-4" />
                <span>{checkSummary.success}</span>
              </div>
            )}
          </div>
        ) : (
          <span className="text-control-light">-</span>
        )}
      </td>

      {/* Approval */}
      <td className="py-2 px-4">
        {approvalTag ? (
          <StatusTag label={approvalTag.label} variant={approvalTag.variant} />
        ) : (
          <span className="text-control-light">-</span>
        )}
      </td>

      {/* Stages */}
      <td className="py-2 px-4">
        {plan.rolloutStageSummaries.length === 0 ? (
          <span className="text-control-light">-</span>
        ) : (
          <div className="flex items-center gap-1 flex-wrap">
            {plan.rolloutStageSummaries.map((summary, index) => {
              const envName = formatEnvironmentName(
                extractStageUID(summary.stage)
              );
              const environment =
                environmentStore.getEnvironmentByName(envName);
              const stageStatus = getRolloutStageStatus(summary);
              return (
                <div key={summary.stage} className="flex items-center gap-1">
                  <div className="flex items-center gap-1">
                    <TaskStatusIcon status={stageStatus} />
                    <span className="text-sm">
                      {environment?.title || envName}
                    </span>
                  </div>
                  {index < plan.rolloutStageSummaries.length - 1 && (
                    <span className="mx-1 text-control-light">&rarr;</span>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </td>

      {/* Updated */}
      <td className="py-2 px-4">
        <Tooltip content={formatAbsoluteDateTime(updateTimeTs * 1000)}>
          <span className="text-control-light whitespace-nowrap">
            {humanizeTs(updateTimeTs)}
          </span>
        </Tooltip>
      </td>

      {/* Creator */}
      <td className="py-2 px-4">
        <div className="flex items-center gap-x-1.5">
          <span className="text-sm truncate">{creator.title}</span>
        </div>
      </td>
    </tr>
  );
}

// ---------------------------------------------------------------------------
// StatusTag
// ---------------------------------------------------------------------------

function StatusTag({
  label,
  variant = "default",
}: {
  label: string;
  variant?: "default" | "success" | "warning" | "info";
}) {
  const variantClasses: Record<string, string> = {
    default: "bg-control-bg text-control-light",
    success: "bg-success/10 text-success",
    warning: "bg-warning/10 text-warning",
    info: "bg-accent/10 text-accent",
  };
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs",
        variantClasses[variant]
      )}
    >
      {label}
    </span>
  );
}

// ---------------------------------------------------------------------------
// TaskStatusIcon
// ---------------------------------------------------------------------------

function TaskStatusIcon({ status }: { status: Task_Status }) {
  const size = "size-4";
  switch (status) {
    case Task_Status.DONE:
      return <CheckCircle className={cn(size, "text-success")} />;
    case Task_Status.RUNNING:
      return <Loader2 className={cn(size, "text-info animate-spin")} />;
    case Task_Status.FAILED:
      return <XCircle className={cn(size, "text-error")} />;
    case Task_Status.CANCELED:
      return <XCircle className={cn(size, "text-control-placeholder")} />;
    case Task_Status.PENDING:
    case Task_Status.NOT_STARTED:
      return (
        <span
          className={cn(
            size,
            "inline-flex items-center justify-center rounded-full border-2 border-control-border"
          )}
        />
      );
    case Task_Status.SKIPPED:
      return (
        <span
          className={cn(
            size,
            "inline-flex items-center justify-center rounded-full bg-control-bg-hover text-control-light"
          )}
        >
          <span className="w-2 h-px bg-current" />
        </span>
      );
    default:
      return (
        <span
          className={cn(
            size,
            "inline-flex items-center justify-center rounded-full border-2 border-block-border"
          )}
        />
      );
  }
}

// ---------------------------------------------------------------------------
// AddSpecDrawer
// ---------------------------------------------------------------------------

function AddSpecDrawer({
  open,
  onClose,
  projectName,
  onCreated,
  title,
}: {
  open: boolean;
  onClose: () => void;
  projectName: string;
  onCreated: (spec: Plan_Spec) => void | Promise<void>;
  title: string;
}) {
  const { t } = useTranslation();
  const dbStore = useDatabaseV1Store();

  const [creating, setCreating] = useState(false);
  const [changeSource, setChangeSource] = useState<"DATABASE" | "GROUP">(
    "DATABASE"
  );
  const [selectedDatabaseNames, setSelectedDatabaseNames] = useState<
    Set<string>
  >(new Set());
  const [selectedDatabaseGroup, setSelectedDatabaseGroup] = useState<
    string | undefined
  >();

  const targets = useMemo(() => {
    if (changeSource === "DATABASE") {
      return [...selectedDatabaseNames];
    }
    return selectedDatabaseGroup ? [selectedDatabaseGroup] : [];
  }, [changeSource, selectedDatabaseNames, selectedDatabaseGroup]);

  const canSubmit = targets.length > 0;

  // Reset on open
  useEffect(() => {
    if (open) {
      setCreating(false);
      setChangeSource("DATABASE");
      setSelectedDatabaseNames(new Set());
      setSelectedDatabaseGroup(undefined);
    }
  }, [open]);

  useEscapeKey(open, onClose);

  const handleConfirm = async () => {
    if (!canSubmit || creating) return;
    setCreating(true);
    try {
      // Preload database information
      if (changeSource === "DATABASE" && selectedDatabaseNames.size > 0) {
        await dbStore.batchGetOrFetchDatabases([...selectedDatabaseNames]);
      }

      const spec = createProto(Plan_SpecSchema, {
        id: uuidv4(),
        config: {
          case: "changeDatabaseConfig",
          value: createProto(Plan_ChangeDatabaseConfigSchema, {
            targets,
          }),
        },
      });

      await onCreated(spec);
      onClose();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: error instanceof Error ? error.message : String(error),
      });
    } finally {
      setCreating(false);
    }
  };

  if (!open) return null;

  return (
    <Sheet open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <SheetContent width="workspace">
        {/* Header */}
        <SheetHeader>
          <SheetTitle>{title}</SheetTitle>
        </SheetHeader>

        {/* Content */}
        <SheetBody className="p-6">
          <DatabaseAndGroupSelector
            projectName={projectName}
            changeSource={changeSource}
            onChangeSourceChange={setChangeSource}
            selectedDatabaseNames={selectedDatabaseNames}
            onSelectedDatabaseNamesChange={setSelectedDatabaseNames}
            selectedDatabaseGroup={selectedDatabaseGroup}
            onSelectedDatabaseGroupChange={setSelectedDatabaseGroup}
          />
        </SheetBody>

        {/* Footer */}
        <SheetFooter>
          <Button variant="outline" onClick={onClose}>
            {t("common.close")}
          </Button>
          <Button disabled={!canSubmit || creating} onClick={handleConfirm}>
            {creating && <Loader2 className="size-4 mr-1 animate-spin" />}
            {t("common.confirm")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ---------------------------------------------------------------------------
// DatabaseAndGroupSelector
// ---------------------------------------------------------------------------

function DatabaseAndGroupSelector({
  projectName,
  changeSource,
  onChangeSourceChange,
  selectedDatabaseNames,
  onSelectedDatabaseNamesChange,
  selectedDatabaseGroup,
  onSelectedDatabaseGroupChange,
}: {
  projectName: string;
  changeSource: "DATABASE" | "GROUP";
  onChangeSourceChange: (source: "DATABASE" | "GROUP") => void;
  selectedDatabaseNames: Set<string>;
  onSelectedDatabaseNamesChange: (names: Set<string>) => void;
  selectedDatabaseGroup: string | undefined;
  onSelectedDatabaseGroupChange: (name: string | undefined) => void;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-y-3">
      <div className="flex border-b border-control-border">
        <button
          type="button"
          className={cn(
            "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
            changeSource === "DATABASE"
              ? "border-accent text-accent"
              : "border-transparent text-control-light hover:text-control"
          )}
          onClick={() => onChangeSourceChange("DATABASE")}
        >
          <span className="inline-flex items-center gap-x-1.5">
            <DatabaseIcon className="size-4" />
            {t("common.databases")}
          </span>
        </button>
        <button
          type="button"
          className={cn(
            "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
            changeSource === "GROUP"
              ? "border-accent text-accent"
              : "border-transparent text-control-light hover:text-control"
          )}
          onClick={() => onChangeSourceChange("GROUP")}
        >
          <span className="inline-flex items-center gap-x-1.5">
            <FolderTree className="size-4" />
            {t("common.database-group")}
          </span>
        </button>
      </div>

      {changeSource === "DATABASE" ? (
        <DatabaseSelector
          projectName={projectName}
          selectedNames={selectedDatabaseNames}
          onSelectedNamesChange={onSelectedDatabaseNamesChange}
        />
      ) : (
        <DatabaseGroupSelector
          projectName={projectName}
          selectedGroup={selectedDatabaseGroup}
          onSelectedGroupChange={onSelectedDatabaseGroupChange}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// DatabaseSelector
// ---------------------------------------------------------------------------

function DatabaseSelector({
  projectName,
  selectedNames,
  onSelectedNamesChange,
}: {
  projectName: string;
  selectedNames: Set<string>;
  onSelectedNamesChange: (names: Set<string>) => void;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();

  const [databases, setDatabases] = useState<Database[]>([]);
  const [loading, setLoading] = useState(true);
  const [hasMore, setHasMore] = useState(false);
  const [isFetchingMore, setIsFetchingMore] = useState(false);
  const [query, setQuery] = useState("");
  const [pageSize] = useSessionPageSize("bb.plan-db-selector");
  const nextPageTokenRef = useRef("");
  const fetchIdRef = useRef(0);

  const doFetch = useCallback(
    async (isRefresh: boolean) => {
      const currentFetchId = ++fetchIdRef.current;
      if (isRefresh) {
        setLoading(true);
      } else {
        setIsFetchingMore(true);
      }
      try {
        const token = isRefresh ? "" : nextPageTokenRef.current;
        const result = await databaseStore.fetchDatabases({
          parent: projectName,
          pageSize,
          pageToken: token || undefined,
          filter: { query },
        });
        if (currentFetchId !== fetchIdRef.current) return;
        setDatabases((prev) =>
          isRefresh ? result.databases : [...prev, ...result.databases]
        );
        nextPageTokenRef.current = result.nextPageToken;
        setHasMore(Boolean(result.nextPageToken));
      } finally {
        if (currentFetchId === fetchIdRef.current) {
          setLoading(false);
          setIsFetchingMore(false);
        }
      }
    },
    [databaseStore, projectName, pageSize, query]
  );

  const isFirstLoad = useRef(true);
  useEffect(() => {
    if (isFirstLoad.current) {
      isFirstLoad.current = false;
      doFetch(true);
      return;
    }
    const timer = setTimeout(() => doFetch(true), 300);
    return () => clearTimeout(timer);
  }, [doFetch]);

  const toggleDatabase = (name: string) => {
    const next = new Set(selectedNames);
    if (next.has(name)) {
      next.delete(name);
    } else {
      next.add(name);
    }
    onSelectedNamesChange(next);
  };

  const toggleAll = () => {
    const allSelected = databases.every((db) => selectedNames.has(db.name));
    if (allSelected) {
      onSelectedNamesChange(new Set());
    } else {
      onSelectedNamesChange(new Set(databases.map((db) => db.name)));
    }
  };

  const allSelected =
    databases.length > 0 && databases.every((db) => selectedNames.has(db.name));
  const someSelected =
    databases.some((db) => selectedNames.has(db.name)) && !allSelected;

  return (
    <div className="flex flex-col gap-y-2">
      <SearchInput
        placeholder={t("database.filter-database")}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />

      {loading ? (
        <div className="flex justify-center py-8 text-control-light">
          <Loader2 className="size-5 animate-spin" />
        </div>
      ) : databases.length === 0 ? (
        <div className="flex justify-center py-8 text-control-light">
          {t("common.no-data")}
        </div>
      ) : (
        <>
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-control-light">
                <th className="py-2 pr-2 w-8">
                  <Checkbox
                    checked={someSelected ? "indeterminate" : allSelected}
                    onCheckedChange={toggleAll}
                  />
                </th>
                <th className="py-2 pr-4 font-medium">
                  {t("common.database")}
                </th>
                <th className="py-2 pr-4 font-medium">
                  {t("common.instance")}
                </th>
                <th className="py-2 pr-4 font-medium">
                  {t("common.environment")}
                </th>
                <th className="py-2 pr-4 font-medium whitespace-nowrap">
                  {t("common.status")}
                </th>
              </tr>
            </thead>
            <tbody>
              {databases.map((db) => {
                const { databaseName } = extractDatabaseResourceName(db.name);
                const inst = getInstanceResource(db);
                const env = getDatabaseEnvironment(db);
                const isSelected = selectedNames.has(db.name);
                return (
                  <tr
                    key={db.name}
                    className={cn(
                      "border-b cursor-pointer hover:bg-control-bg",
                      isSelected && "bg-accent/5"
                    )}
                    onClick={() => toggleDatabase(db.name)}
                  >
                    <td className="py-2 pr-2">
                      <Checkbox checked={isSelected} />
                    </td>
                    <td className="py-2 pr-4">
                      <div className="flex items-center gap-x-1.5">
                        {inst && (
                          <EngineIcon engine={inst.engine} className="size-4" />
                        )}
                        <span>{databaseName}</span>
                      </div>
                    </td>
                    <td className="py-2 pr-4">{inst?.title}</td>
                    <td className="py-2 pr-4">
                      {env && <EnvironmentLabel environmentName={env.name} />}
                    </td>
                    <td className="py-2 pr-4">
                      {db.syncStatus === SyncStatus.FAILED ? (
                        <Tooltip
                          content={
                            db.syncError || t("database.sync-status-failed")
                          }
                        >
                          <XCircle className="size-4 text-error" />
                        </Tooltip>
                      ) : (
                        <CheckCircle className="size-4 text-success" />
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>

          {hasMore && (
            <div className="flex justify-center">
              <Button
                variant="ghost"
                size="sm"
                disabled={isFetchingMore}
                onClick={() => doFetch(false)}
              >
                {isFetchingMore ? t("common.loading") : t("common.load-more")}
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// DatabaseGroupSelector
// ---------------------------------------------------------------------------

function DatabaseGroupSelector({
  projectName,
  selectedGroup,
  onSelectedGroupChange,
}: {
  projectName: string;
  selectedGroup: string | undefined;
  onSelectedGroupChange: (name: string | undefined) => void;
}) {
  const { t } = useTranslation();
  const dbGroupStore = useDBGroupStore();
  const [groups, setGroups] = useState<DatabaseGroup[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    dbGroupStore
      .fetchDBGroupListByProjectName(projectName, DatabaseGroupView.BASIC)
      .then((result) => {
        setGroups(result);
      })
      .finally(() => setLoading(false));
  }, [projectName, dbGroupStore]);

  if (loading) {
    return (
      <div className="flex justify-center py-8 text-control-light">
        <Loader2 className="size-5 animate-spin" />
      </div>
    );
  }

  if (groups.length === 0) {
    return (
      <div className="flex justify-center py-8 text-control-light">
        {t("common.no-data")}
      </div>
    );
  }

  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="border-b text-left text-control-light">
          <th className="py-2 pr-2 w-8" />
          <th className="py-2 pr-4 font-medium">
            {t("common.database-group")}
          </th>
        </tr>
      </thead>
      <tbody>
        {groups.map((group) => {
          const isSelected = selectedGroup === group.name;
          return (
            <tr
              key={group.name}
              className={cn(
                "border-b cursor-pointer hover:bg-control-bg",
                isSelected && "bg-accent/5"
              )}
              onClick={() =>
                onSelectedGroupChange(isSelected ? undefined : group.name)
              }
            >
              <td className="py-2 pr-2">
                <input
                  type="radio"
                  checked={isSelected}
                  readOnly
                  className="accent-accent"
                />
              </td>
              <td className="py-2 pr-4">
                <div className="flex items-center gap-x-1.5">
                  <FolderTree className="size-4 text-control-light shrink-0" />
                  <span>{group.title}</span>
                </div>
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
