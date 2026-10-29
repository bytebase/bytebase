import { create as createProto } from "@bufbuild/protobuf";
import {
  Database as DatabaseIcon,
  FolderTree,
  Loader2,
  Plus,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { router } from "@/app/router";
import {
  markListScrollRestorationEntry,
  useListScrollRestorationKey,
  useListScrollRestorationLoadMore,
} from "@/app/router/NavigationScrollRestoration";
import { buildPlanCreateRoute } from "@/app/router/routeHelpers";
import {
  AdvancedSearch,
  type ScopeOption,
  type SearchParams,
  type ValueOption,
} from "@/components/AdvancedSearch";
import { DatabaseGroupTable } from "@/components/DatabaseGroupTable";
import { DatabaseTable } from "@/components/database";
import { HumanizeTs } from "@/components/HumanizeTs";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/components/PermissionGuard";
import {
  ProjectPageContent,
  ProjectPageFooter,
  ProjectPageInfo,
  ProjectPageLayout,
} from "@/components/ProjectPageLayout";
import { TimeRangePicker } from "@/components/TimeRangePicker";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SearchInput } from "@/components/ui/search-input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useCurrentUser } from "@/hooks/useAppState";
import { useEscapeKey } from "@/hooks/useEscapeKey";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { PagedTableFooter, usePagedData } from "@/hooks/usePagedData";
import { useProjectByName } from "@/hooks/useProjectByName";
import {
  createAdvancedSearchParser,
  serializeAdvancedSearch,
  useURLSearchParam,
} from "@/hooks/useURLSearchParam";
import { applyPlanTitleToQuery } from "@/lib/plan/title";
import { projectPlansPagedDataCacheScope } from "@/lib/projectPagedDataCache";
import { cn } from "@/lib/utils";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { buildPlanFindBySearchParams } from "@/stores/app/plan";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import { isValidDatabaseGroupName } from "@/types";
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
} from "@/utils";
import {
  createPlanStagesColumn,
  type PlanColumn,
  PlanLifecycleBadges,
  PlanTable,
} from "./PlanTable";
import { planStateScopeOption, planStatusScopeOption } from "./planListSearch";

// Below Tailwind's `sm` breakpoint (640px) we switch the plan list and the
// database picker to their compact mobile layouts.
const MOBILE_MEDIA_QUERY = "(max-width: 639px)";
const parsePlanSearchParams = createAdvancedSearchParser([
  "state",
  "creator",
  "status",
  "created",
]);

export function ProjectPlanDashboardPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const listScrollRestorationKey = useListScrollRestorationKey();
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
  const defaultSearchParams = useMemo<SearchParams>(
    () => ({
      query: "",
      scopes: [{ id: "state", value: "ACTIVE" }],
    }),
    []
  );
  const [searchParams, setSearchParams] = useURLSearchParam({
    param: "q",
    parse: parsePlanSearchParams,
    serialize: serializeAdvancedSearch,
    defaultValue: defaultSearchParams,
  });
  const viewCacheKey = useMemo(
    () =>
      JSON.stringify([
        "project-plans",
        projectName,
        serializeAdvancedSearch(searchParams),
      ]),
    [projectName, searchParams]
  );

  // Scope options
  const scopeOptions: ScopeOption[] = useMemo(
    () => [
      planStateScopeOption(t),
      planStatusScopeOption(t),
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

  const isMobile = useMediaQuery(MOBILE_MEDIA_QUERY);
  const columns = usePlanColumns(isMobile);

  const [canCreate] = usePermissionCheck(
    ["bb.plans.create", "bb.issues.create"],
    project
  );

  // Build plan filter
  const planFilter = useMemo(() => {
    const merged: SearchParams = {
      query: searchParams.query.trim().toLowerCase(),
      scopes: [
        ...searchParams.scopes.map((s) => ({ id: s.id, value: s.value })),
        { id: "project", value: projectId },
      ],
    };
    return buildPlanFindBySearchParams(merged, {
      specType: "change_database_config",
      hasRelease: false,
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
    cacheKey: viewCacheKey,
    cacheScope: projectPlansPagedDataCacheScope(projectId),
    cacheRestoreToken: listScrollRestorationKey,
    fetchList: fetchPlanList,
  });
  useListScrollRestorationLoadMore(paged);

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

      const targets =
        spec.config?.case === "changeDatabaseConfig"
          ? [...(spec.config.value.targets ?? [])]
          : [];
      const isDatabaseGroup = targets.every((target) =>
        isValidDatabaseGroupName(target)
      );
      const query: Record<string, string> = {};

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
          generatePlanTitle([extractDatabaseGroupName(databaseGroupName)])
        );
      } else {
        query.databaseList = targets.join(",");
        applyPlanTitleToQuery(query, project, () =>
          generatePlanTitle(
            targets.map((db) => {
              const { databaseName } = extractDatabaseResourceName(db);
              return databaseName;
            })
          )
        );
      }

      await router.push(buildPlanCreateRoute(projectId, query));
    },
    [project, projectId]
  );

  return (
    <ProjectPageLayout>
      <div className="flex flex-col gap-y-2">
        <ProjectPageInfo description={t("plan.list-description")} />
        <div className="w-full flex flex-col lg:flex-row items-start lg:items-center justify-between gap-2">
          <div className="w-full flex flex-1 flex-wrap items-center justify-between gap-2">
            <div className="min-w-0 basis-80 grow">
              <AdvancedSearch
                params={searchParams}
                onParamsChange={setSearchParams}
                scopeOptions={scopeOptions}
              />
            </div>
            <TimeRangePicker
              params={searchParams}
              onParamsChange={setSearchParams}
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
          <PlanTable
            plans={paged.dataList}
            projectId={projectId}
            columns={columns}
            isMobile={isMobile}
            onOpenPlan={markListScrollRestorationEntry}
          />
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
// Columns
// ---------------------------------------------------------------------------

function usePlanColumns(isMobile: boolean): PlanColumn[] {
  const { t } = useTranslation();
  return useMemo<PlanColumn[]>(
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
            <PlanLifecycleBadges ctx={ctx} />
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
              className="font-normal whitespace-nowrap"
            >
              {ctx.approvalTag.label}
            </Badge>
          ) : (
            <span className="text-control-light">-</span>
          ),
      },
      createPlanStagesColumn(t),
      {
        key: "updated",
        title: t("issue.table.updated"),
        defaultWidth: 152,
        minWidth: 100,
        resizable: true,
        render: (_plan, ctx) =>
          ctx.updateTimeMs === undefined ? (
            <span className="text-control-light">-</span>
          ) : (
            <HumanizeTs
              tsMs={ctx.updateTimeMs}
              className="text-control-light whitespace-nowrap"
            />
          ),
      },
    ],
    [t, isMobile]
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
        <Button
          appearance="secondary"
          size="md"
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
        </Button>
        <Button
          appearance="secondary"
          size="md"
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
        </Button>
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
