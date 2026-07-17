import { create as createProto } from "@bufbuild/protobuf";
import {
  Database as DatabaseIcon,
  FolderTree,
  Loader2,
  Plus,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import {
  AdvancedSearch,
  type ScopeOption,
  type SearchParams,
  type ValueOption,
} from "@/react/components/AdvancedSearch";
import { DatabaseGroupTable } from "@/react/components/DatabaseGroupTable";
import { DatabaseTable } from "@/react/components/database";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/react/components/PermissionGuard";
import {
  ProjectPageContent,
  ProjectPageFooter,
  ProjectPageInfo,
  ProjectPageLayout,
} from "@/react/components/ProjectPageLayout";
import { TaskStatusIcon } from "@/react/components/TaskStatusIcon";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import { SearchInput } from "@/react/components/ui/search-input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { useColumnWidths } from "@/react/hooks/useColumnWidths";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useMediaQuery } from "@/react/hooks/useMediaQuery";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { applyPlanTitleToQuery } from "@/react/lib/plan/title";
import { cn } from "@/react/lib/utils";
import { router } from "@/react/router";
import {
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
} from "@/react/router/handles";
import { useScrollRestorationLoadMore } from "@/react/router/NavigationScrollRestoration";
import { useAppStore } from "@/react/stores/app";
import { buildPlanFindBySearchParams } from "@/react/stores/app/plan";
import { pushNotification } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  formatEnvironmentName,
  getTimeForPbTimestampProtoEs,
  isValidDatabaseGroupName,
  unknownUser,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type { Plan, Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  extractDatabaseGroupName,
  extractDatabaseResourceName,
  extractPlanUID,
  generatePlanTitle,
  getDefaultPagination,
  type SearchParams as VueSearchParams,
} from "@/utils";
import {
  extractStageUID,
  getStageStatusFromCounts,
} from "@/utils/v1/issue/rollout";
import { isReleaseBackedPlan } from "./plan-detail/utils/spec";
import {
  getPlanDraftState,
  getReviewBadge,
  type PlanDraftState,
  type ReviewBadge,
} from "./utils/reviewBadge";

// Below Tailwind's `sm` breakpoint (640px) we switch the plan list and the
// database picker to their compact mobile layouts.
const MOBILE_MEDIA_QUERY = "(max-width: 639px)";

export function ProjectPlanDashboardPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const projectsByName = useAppStore((s) => s.projectsByName);
  const listUsers = useAppStore((state) => state.listUsers);
  const batchGetOrFetchUsers = useAppStore(
    (state) => state.batchGetOrFetchUsers
  );
  const projectName = `${projectNamePrefix}${projectId}`;
  // subscribe to re-render on project cache change
  void projectsByName;
  const project = useProjectByName(projectName);
  const me = useCurrentUser();

  const [showAddSpecDrawer, setShowAddSpecDrawer] = useState(false);

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
        onSearch: async (keyword: string) => {
          const resp = await listUsers({
            pageSize: getDefaultPagination(),
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
      },
    ],
    [t, listUsers, me]
  );

  const [canCreate] = usePermissionCheck(
    ["bb.plans.create", "bb.issues.create"],
    project
  );

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
      const { nextPageToken, plans } = await useAppStore.getState().listPlans({
        find: planFilter,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
      });
      return { list: plans, nextPageToken };
    },
    [planFilter]
  );

  const paged = usePagedData<Plan>({
    sessionKey: `bb.${projectName}.plan-table`,
    fetchList: fetchPlanList,
  });
  useScrollRestorationLoadMore(paged);

  useEffect(() => {
    if (paged.dataList.length === 0) {
      return;
    }
    void batchGetOrFetchUsers(paged.dataList.map((plan) => plan.creator));
  }, [batchGetOrFetchUsers, paged.dataList]);

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
    <ProjectPageLayout>
      <div className="flex flex-col gap-y-2">
        <ProjectPageInfo description={t("plan.subtitle")} />
        <div className="w-full flex flex-col lg:flex-row items-start lg:items-center justify-between gap-2">
          <div className="w-full flex flex-1 items-center justify-between gap-x-2">
            <AdvancedSearch
              params={searchParams}
              onParamsChange={setSearchParams}
              scopeOptions={scopeOptions}
            />
            <PermissionGuard
              permissions={["bb.plans.create", "bb.issues.create"]}
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
      <ProjectPageContent>
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
          <ProjectPageFooter>
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

      <AddSpecDrawer
        open={showAddSpecDrawer}
        onClose={() => setShowAddSpecDrawer(false)}
        projectName={projectName}
        onCreated={handleSpecCreated}
        title={t("plan.new-plan")}
      />
    </ProjectPageLayout>
  );
}

// ---------------------------------------------------------------------------
// PlanTable
// ---------------------------------------------------------------------------

interface PlanColumn {
  key: string;
  title: React.ReactNode;
  defaultWidth: number;
  minWidth?: number;
  resizable?: boolean;
  render: (plan: Plan, ctx: PlanRowContext) => React.ReactNode;
}

interface PlanRowContext {
  creator: { title: string; name: string };
  updateTimeTs: number;
  approvalTag: { label: string; variant: ReviewBadge["variant"] } | undefined;
  isDeleted: boolean;
  draftState: PlanDraftState;
}

function PlanTable({ plans, projectId }: { plans: Plan[]; projectId: string }) {
  const { t } = useTranslation();
  const isMobile = useMediaQuery(MOBILE_MEDIA_QUERY);
  // Subscribe so stage cells re-render when the environment cache loads.
  void useAppStore((s) => s.environmentList);

  const columns = useMemo<PlanColumn[]>(
    () => [
      {
        key: "name",
        title: t("issue.table.name"),
        // On phones the name column keeps a fixed, compact width so it does
        // not dominate the row or push the whole table wider than the
        // viewport; the remaining columns scroll horizontally.
        defaultWidth: isMobile ? 200 : 400,
        minWidth: 200,
        resizable: !isMobile,
        render: (plan, ctx) => (
          <div className="flex items-center gap-x-2 overflow-hidden">
            <span className="whitespace-nowrap text-control opacity-60">
              {extractPlanUID(plan.name)}
            </span>
            {plan.title ? (
              <span className="truncate normal-nums min-w-0">{plan.title}</span>
            ) : (
              <span className="opacity-60 italic">{t("common.untitled")}</span>
            )}
            {ctx.isDeleted && (
              <span className="inline-flex items-center rounded-full bg-warning/10 text-warning px-2 py-0.5 text-xs shrink-0">
                {t("common.closed")}
              </span>
            )}
            {ctx.draftState === "draft" && !ctx.isDeleted && (
              <span className="inline-flex items-center rounded-full bg-control-bg text-control-light px-2 py-0.5 text-xs shrink-0">
                {t("common.draft")}
              </span>
            )}
            {ctx.draftState === "incomplete" && !ctx.isDeleted && (
              <Badge variant="destructive" className="shrink-0 px-2 text-xs">
                {t("plan.lifecycle.incomplete")}
              </Badge>
            )}
          </div>
        ),
      },
      {
        key: "creator",
        title: t("issue.table.creator"),
        defaultWidth: 160,
        minWidth: 100,
        resizable: true,
        render: (_plan, ctx) => (
          <span className="block truncate">{ctx.creator.title}</span>
        ),
      },
      {
        key: "review",
        title: t("plan.navigator.review"),
        defaultWidth: 140,
        minWidth: 100,
        resizable: true,
        render: (_plan, ctx) =>
          ctx.approvalTag ? (
            <Badge
              variant={ctx.approvalTag.variant}
              className="whitespace-nowrap"
            >
              {ctx.approvalTag.label}
            </Badge>
          ) : (
            <span className="text-control-light">-</span>
          ),
      },
      {
        key: "stages",
        title: t("rollout.stage.self", { count: 2 }),
        defaultWidth: 260,
        minWidth: 140,
        resizable: true,
        render: (plan, _ctx) =>
          plan.rolloutStageSummaries.length === 0 ? (
            <span className="text-control-light">-</span>
          ) : (
            <div className="flex items-center gap-1 flex-wrap">
              {plan.rolloutStageSummaries.map((summary, index) => {
                const envName = formatEnvironmentName(
                  extractStageUID(summary.stage)
                );
                const environment = useAppStore
                  .getState()
                  .getEnvironmentByName(envName);
                const stageStatus = getStageStatusFromCounts(
                  summary.taskStatusCounts
                );
                return (
                  <div key={summary.stage} className="flex items-center gap-1">
                    <div className="flex items-center gap-1">
                      <TaskStatusIcon size="tiny" status={stageStatus} />
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
          ),
      },
      {
        key: "updated",
        title: t("issue.table.updated"),
        defaultWidth: 152,
        minWidth: 100,
        resizable: true,
        render: (_plan, ctx) => (
          <HumanizeTs
            ts={ctx.updateTimeTs}
            className="text-control-light whitespace-nowrap"
          />
        ),
      },
    ],
    [t, isMobile]
  );

  const { widths, totalWidth, onResizeStart, setWidths } =
    useColumnWidths(columns);

  // useColumnWidths seeds its state only on first render, so a viewport that
  // crosses the `sm` breakpoint after mount (resize / rotate) would keep the
  // stale name-column width. Re-seed from the rebuilt column defaults whenever
  // the breakpoint flips.
  const wasMobile = useRef(isMobile);
  useEffect(() => {
    if (wasMobile.current !== isMobile) {
      wasMobile.current = isMobile;
      setWidths(columns.map((c) => c.defaultWidth));
    }
  }, [isMobile, columns, setWidths]);

  return (
    <div className="overflow-x-auto rounded-sm border border-block-border">
      <Table className="table-fixed" style={{ minWidth: `${totalWidth}px` }}>
        <colgroup>
          {widths.map((w, i) => (
            <col key={columns[i].key} style={{ width: `${w}px` }} />
          ))}
        </colgroup>
        <TableHeader>
          <TableRow className="bg-control-bg">
            {columns.map((col, colIdx) => (
              <TableHead
                key={col.key}
                resizable={col.resizable}
                onResizeStart={
                  col.resizable ? (e) => onResizeStart(colIdx, e) : undefined
                }
              >
                {col.title}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {plans.map((plan) => (
            <PlanRow
              key={plan.name}
              plan={plan}
              projectId={projectId}
              columns={columns}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

// ---------------------------------------------------------------------------
// PlanRow
// ---------------------------------------------------------------------------

function PlanRow({
  plan,
  projectId,
  columns,
}: Readonly<{
  plan: Plan;
  projectId: string;
  columns: PlanColumn[];
}>) {
  const { t } = useTranslation();

  const isDeleted = plan.state === State.DELETED;
  const draftState = getPlanDraftState({
    approvalStatus: plan.approvalStatus,
    hasRollout: plan.hasRollout,
    isGitOpsPlan: isReleaseBackedPlan(plan.specs),
    issueName: plan.issue,
  });

  const creatorUser = useAppStore((state) =>
    state.getUserByIdentifier(plan.creator)
  );
  const creator = creatorUser || unknownUser(plan.creator);

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
  // Plan List passes issueStatus=undefined because the Plan proto does not
  // expose issue_status. Two divergence categories vs Plan Detail remain by
  // design (see the spec's "Residual divergences" table):
  //   Category A — CANCELED issues render the approval-derived badge here
  //     instead of "Closed".
  //   Category C₂ — issue manually marked DONE with no rollout while approval
  //     is still PENDING renders "Under Review" here instead of "Bypassed".
  // To close those gaps, expose issue_status on the Plan proto and pass it
  // through here. Bug history: BYT-9551.
  const approvalTag = useMemo(() => {
    const badge = getReviewBadge({
      hasIssue: plan.issue !== "",
      issueStatus: undefined,
      hasRollout: plan.hasRollout,
      approvalStatus: plan.approvalStatus,
    });
    if (!badge) return undefined;
    return { label: t(badge.labelKey), variant: badge.variant };
  }, [plan.approvalStatus, plan.hasRollout, plan.issue, t]);

  const ctx: PlanRowContext = {
    creator,
    updateTimeTs,
    approvalTag,
    isDeleted,
    draftState,
  };

  return (
    <TableRow
      className={cn("cursor-pointer", isDeleted && "opacity-60")}
      onClick={onRowClick}
    >
      {columns.map((col) => (
        <TableCell key={col.key} className="overflow-hidden">
          {col.render(plan, ctx)}
        </TableCell>
      ))}
    </TableRow>
  );
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
        await useAppStore
          .getState()
          .batchGetOrFetchDatabases([...selectedDatabaseNames]);
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
          <Button appearance="outline" onClick={onClose}>
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
        <DatabaseGroupTable
          projectName={projectName}
          view={DatabaseGroupView.BASIC}
          showSelection
          singleSelection
          selectedDatabaseGroupNames={
            selectedDatabaseGroup ? [selectedDatabaseGroup] : []
          }
          onSelectedDatabaseGroupNamesChange={(names) =>
            onSelectedDatabaseGroupChange(names[0])
          }
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
  const [query, setQuery] = useState("");

  return (
    <div className="flex flex-col gap-y-3">
      <SearchInput
        placeholder={t("database.filter-database")}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <DatabaseTable
        filter={{ query }}
        parent={projectName}
        mode="PROJECT"
        selectOnRowClick
        selectedNames={selectedNames}
        onSelectedNamesChange={onSelectedNamesChange}
      />
    </div>
  );
}
