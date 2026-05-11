import { Archive, Check, EllipsisVertical, Plus, Trash2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AdvancedSearch,
  getValueFromScopes,
  type ScopeOption,
  type SearchParams,
} from "@/react/components/AdvancedSearch";
import { ProjectCreateDialog } from "@/react/components/header/ProjectCreateDialog";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { ProjectTable } from "@/react/components/ProjectTable";
import {
  type SelectionAction,
  SelectionActionBar,
} from "@/react/components/SelectionActionBar";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { PagedTableFooter } from "@/react/hooks/usePagedData";
import {
  getPageSizeOptions,
  useSessionPageSize,
} from "@/react/hooks/useSessionPageSize";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_ISSUES } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useAuthStore,
  useProjectV1Store,
  useUIStateStore,
} from "@/store";
import { getProjectName } from "@/store/modules/v1/common";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  extractProjectResourceName,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
} from "@/utils";

export function projectIssuesRoute(project: Project) {
  return {
    name: PROJECT_V1_ROUTE_ISSUES,
    params: { projectId: getProjectName(project.name) },
  };
}

// ============================================================
// ConfirmDialog
// ============================================================

function ConfirmDialog({
  open,
  variant,
  title,
  description,
  okText,
  onOk,
  onCancel,
  children,
}: {
  open: boolean;
  variant: "warning" | "error";
  title: string;
  description: string;
  okText: string;
  onOk: () => void;
  onCancel: () => void;
  children?: React.ReactNode;
}) {
  const { t } = useTranslation();

  if (!open) return null;

  const borderColor = variant === "error" ? "border-error" : "border-warning";
  const okBg =
    variant === "error"
      ? "bg-error hover:bg-error-hover text-accent-text"
      : "bg-warning hover:bg-warning-hover text-accent-text";

  return (
    <AlertDialog open onOpenChange={(nextOpen) => !nextOpen && onCancel()}>
      <AlertDialogContent className={cn("max-w-lg border-t-4", borderColor)}>
        <AlertDialogTitle>{title}</AlertDialogTitle>
        <AlertDialogDescription className="mt-2">
          {description}
        </AlertDialogDescription>
        {children && <div className="mt-4">{children}</div>}
        <div className="mt-6 flex justify-end gap-x-2">
          <Button variant="outline" onClick={onCancel}>
            {t("common.cancel")}
          </Button>
          <button
            className={cn(
              "inline-flex items-center justify-center rounded-xs px-4 py-2 text-sm font-medium",
              okBg
            )}
            onClick={onOk}
          >
            {okText}
          </button>
        </div>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function ProjectListPreview({
  projects,
  iconColor,
}: {
  projects: Project[];
  iconColor: string;
}) {
  return (
    <div className="max-h-40 overflow-y-auto border rounded-sm p-2 bg-control-bg">
      <div className="flex flex-col gap-y-1">
        {projects.map((project) => (
          <div key={project.name} className="text-sm flex items-center gap-x-2">
            <Check className={cn("size-3", iconColor)} />
            <span>{project.title}</span>
            <span className="text-control-light">
              ({extractProjectResourceName(project.name)})
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ============================================================
// ProjectActionDropdown
// ============================================================

function ProjectActionDropdown({
  project,
  onAction,
}: {
  project: Project;
  onAction: () => void;
}) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();

  const canDelete = hasProjectPermissionV2(project, "bb.projects.delete");
  const canUndelete = hasProjectPermissionV2(project, "bb.projects.undelete");

  const handleArchive = useCallback(async () => {
    if (
      !window.confirm(
        t("project.settings.confirm-archive-project", {
          name: project.title,
        })
      )
    ) {
      return;
    }
    await projectStore.archiveProject(project);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.archived"),
    });
    onAction();
  }, [project, projectStore, t, onAction]);

  const handleRestore = useCallback(async () => {
    await projectStore.restoreProject(project);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.restored"),
    });
    onAction();
  }, [project, projectStore, t, onAction]);

  if (!canDelete && !canUndelete) return null;

  const isActive = project.state === State.ACTIVE;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className="p-1 rounded-xs hover:bg-control-bg outline-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <EllipsisVertical className="h-4 w-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        {isActive && canDelete && (
          <DropdownMenuItem
            onClick={(e) => {
              e.stopPropagation();
              handleArchive();
            }}
          >
            <Archive className="h-4 w-4" />
            {t("common.archive")}
          </DropdownMenuItem>
        )}
        {!isActive && canUndelete && (
          <DropdownMenuItem
            onClick={(e) => {
              e.stopPropagation();
              handleRestore();
            }}
          >
            {t("common.restore")}
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

// ============================================================
// ProjectsPage (main)
// ============================================================

export function ProjectsPage() {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const authStore = useAuthStore();

  // Search state — managed as SearchParams (query + scopes)
  const [searchParams, setSearchParams] = useState<SearchParams>(() => {
    const currentRoute = router.currentRoute.value;
    // Migrate old URL format
    const queryState = currentRoute.query.state as string;
    if (queryState === "archived" || queryState === "all") {
      const stateValue = queryState === "archived" ? "DELETED" : "ALL";
      return { query: "", scopes: [{ id: "state", value: stateValue }] };
    }
    const queryString = currentRoute.query.q as string;
    if (queryString) {
      const scopes: { id: string; value: string }[] = [];
      const queryParts: string[] = [];
      for (const token of queryString.split(/\s+/).filter(Boolean)) {
        const colonIdx = token.indexOf(":");
        if (colonIdx > 0) {
          const id = token.substring(0, colonIdx);
          const value = token.substring(colonIdx + 1);
          if (value && (id === "state" || id === "label")) {
            scopes.push({ id, value });
            continue;
          }
        }
        queryParts.push(token);
      }
      return { query: queryParts.join(" "), scopes };
    }
    return { query: "", scopes: [] };
  });

  // Scope options for the AdvancedSearch
  const canUndelete = hasWorkspacePermissionV2("bb.projects.undelete");
  const scopeOptions: ScopeOption[] = useMemo(() => {
    const stateOption: ScopeOption = {
      id: "state",
      title: t("common.state"),
      description: t("issue.advanced-search.scope.state.description"),
      options: [
        { value: "ACTIVE", keywords: ["active"] },
        ...(canUndelete
          ? [
              { value: "DELETED", keywords: ["archived", "deleted"] },
              { value: "ALL", keywords: ["all"] },
            ]
          : []),
      ],
    };
    const labelOption: ScopeOption = {
      id: "label",
      title: t("common.labels"),
      description: t("issue.advanced-search.scope.label.description"),
      allowMultiple: true,
    };
    return [labelOption, stateOption];
  }, [t, canUndelete]);

  // Derived values from searchParams
  const searchText = searchParams.query;
  const stateFilter = useMemo(() => {
    const val = getValueFromScopes(searchParams, "state");
    if (val === "DELETED") return "DELETED" as const;
    if (val === "ALL") return "ALL" as const;
    return "ACTIVE" as const;
  }, [searchParams]);
  const selectedLabels = useMemo(
    () =>
      searchParams.scopes.filter((s) => s.id === "label").map((s) => s.value),
    [searchParams]
  );

  // Mark project visit on mount
  useEffect(() => {
    const uiStateStore = useUIStateStore();
    if (!uiStateStore.getIntroStateByKey("project.visit")) {
      uiStateStore.saveIntroStateByKey({
        key: "project.visit",
        newState: true,
      });
    }
  }, []);

  // Sync search state to URL
  useEffect(() => {
    const parts: string[] = [];
    for (const scope of searchParams.scopes) {
      parts.push(`${scope.id}:${scope.value}`);
    }
    if (searchParams.query) parts.push(searchParams.query);
    const queryString = parts.join(" ");
    const currentQuery = router.currentRoute.value.query.q as string;
    if (queryString !== (currentQuery ?? "")) {
      router.replace({ query: { q: queryString } });
    }
  }, [searchParams]);

  // Data fetching state
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const nextPageTokenRef = useRef("");
  const [hasMore, setHasMore] = useState(false);
  const [isFetchingMore, setIsFetchingMore] = useState(false);
  const [pageSize, setPageSize] = useSessionPageSize("bb.project-table");
  const abortRef = useRef<AbortController | null>(null);
  const fetchIdRef = useRef(0);

  // Sort state
  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("asc");

  const orderBy = useMemo(() => {
    if (!sortKey) return "";
    return `${sortKey} ${sortOrder}`;
  }, [sortKey, sortOrder]);

  const toggleSort = useCallback(
    (key: string) => {
      if (sortKey === key) {
        if (sortOrder === "asc") setSortOrder("desc");
        else {
          setSortKey(null);
          setSortOrder("asc");
        }
      } else {
        setSortKey(key);
        setSortOrder("asc");
      }
    },
    [sortKey, sortOrder]
  );

  const selectedState = useMemo(() => {
    if (stateFilter === "DELETED") return State.DELETED;
    if (stateFilter === "ALL") return undefined;
    return State.ACTIVE;
  }, [stateFilter]);

  const fetchProjects = useCallback(
    async (isRefresh: boolean) => {
      if (!authStore.isLoggedIn) return;

      abortRef.current?.abort();
      abortRef.current = new AbortController();
      const currentFetchId = ++fetchIdRef.current;

      if (isRefresh) {
        setLoading(true);
      } else {
        setIsFetchingMore(true);
      }

      try {
        const token = isRefresh ? "" : nextPageTokenRef.current;
        const result = await projectStore.fetchProjectList({
          pageToken: token,
          pageSize,
          filter: {
            query: searchText,
            excludeDefault: true,
            state: selectedState,
            labels: selectedLabels.length > 0 ? selectedLabels : undefined,
          },
          orderBy,
          cache: true,
        });

        if (currentFetchId !== fetchIdRef.current) return;

        if (isRefresh) {
          setProjects(result.projects);
        } else {
          setProjects((prev) => [...prev, ...result.projects]);
        }
        nextPageTokenRef.current = result.nextPageToken ?? "";
        setHasMore(Boolean(result.nextPageToken));
      } catch (e) {
        if (e instanceof Error && e.name === "AbortError") return;
        console.error(e);
      } finally {
        if (currentFetchId === fetchIdRef.current) {
          setLoading(false);
          setIsFetchingMore(false);
        }
      }
    },
    [
      authStore.isLoggedIn,
      pageSize,
      searchText,
      selectedState,
      selectedLabels,
      orderBy,
      projectStore,
    ]
  );

  // Fetch on mount + re-fetch on filter/sort/pageSize changes (debounced after first load)
  const prevDepsRef = useRef<string>("");
  const isFirstLoad = useRef(true);
  useEffect(() => {
    const depsKey = `${searchText}|${stateFilter}|${selectedLabels.join(",")}|${pageSize}|${orderBy}`;
    if (prevDepsRef.current === depsKey) return;
    prevDepsRef.current = depsKey;

    if (isFirstLoad.current) {
      isFirstLoad.current = false;
      fetchProjects(true);
      return () => abortRef.current?.abort();
    }
    const timer = setTimeout(() => fetchProjects(true), 300);
    return () => clearTimeout(timer);
  }, [
    searchText,
    stateFilter,
    selectedLabels,
    pageSize,
    orderBy,
    fetchProjects,
  ]);

  const loadMore = useCallback(() => {
    if (nextPageTokenRef.current && !isFetchingMore) {
      fetchProjects(false);
    }
  }, [isFetchingMore, fetchProjects]);

  // Selection state
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set());

  const selectedProjectList = useMemo(() => {
    if (selectedNames.size === 0) return [];
    return Array.from(selectedNames)
      .map((name) => projectStore.getProjectByName(name))
      .filter((p): p is Project => p !== undefined);
  }, [selectedNames, projectStore]);

  const handleBatchOperation = useCallback(() => {
    setSelectedNames(new Set());
    fetchProjects(true);
  }, [fetchProjects]);

  // Batch confirm dialog state + handlers (lifted from old BatchOperationsBar)
  const [showArchiveConfirm, setShowArchiveConfirm] = useState(false);
  const [showRestoreConfirm, setShowRestoreConfirm] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const hasActiveProjects = selectedProjectList.some(
    (p) => p.state === State.ACTIVE
  );
  const hasArchivedProjects = selectedProjectList.some(
    (p) => p.state === State.DELETED
  );

  const handleBatchArchive = useCallback(async () => {
    try {
      const activeProjects = selectedProjectList.filter(
        (p) => p.state === State.ACTIVE
      );
      await projectStore.batchDeleteProjects(activeProjects.map((p) => p.name));
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.batch.archive.success", {
          count: activeProjects.length,
        }),
      });
      setShowArchiveConfirm(false);
      handleBatchOperation();
    } catch (error: unknown) {
      const err = error as { message?: string };
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("project.batch.archive.error"),
        description: err.message,
      });
    }
  }, [selectedProjectList, projectStore, t, handleBatchOperation]);

  const handleBatchRestore = useCallback(async () => {
    try {
      const archivedProjects = selectedProjectList.filter(
        (p) => p.state === State.DELETED
      );
      await Promise.all(
        archivedProjects.map((project) => projectStore.restoreProject(project))
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.restored"),
      });
      setShowRestoreConfirm(false);
      handleBatchOperation();
    } catch (error: unknown) {
      const err = error as { message?: string };
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.restore"),
        description: err.message,
      });
    }
  }, [selectedProjectList, projectStore, t, handleBatchOperation]);

  const handleBatchDelete = useCallback(async () => {
    try {
      await projectStore.batchPurgeProjects(
        selectedProjectList.map((p) => p.name)
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.batch.delete.success", {
          count: selectedProjectList.length,
        }),
      });
      setShowDeleteConfirm(false);
      handleBatchOperation();
    } catch (error: unknown) {
      const err = error as { message?: string };
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("project.batch.delete.error"),
        description: err.message,
      });
    }
  }, [selectedProjectList, projectStore, t, handleBatchOperation]);

  const batchActions: SelectionAction[] = [
    {
      key: "archive",
      label: t("common.archive"),
      icon: Archive,
      onClick: () => setShowArchiveConfirm(true),
      hidden: !hasActiveProjects,
    },
    {
      key: "restore",
      label: t("common.restore"),
      onClick: () => setShowRestoreConfirm(true),
      hidden: !hasArchivedProjects,
    },
    {
      key: "delete",
      label: t("common.delete"),
      icon: Trash2,
      onClick: () => setShowDeleteConfirm(true),
      tone: "destructive",
    },
  ];

  // Create drawer
  const [showCreateDrawer, setShowCreateDrawer] = useState(false);
  const canCreate = hasWorkspacePermissionV2("bb.projects.create");
  const canDelete = hasWorkspacePermissionV2("bb.projects.delete");

  const handleCreated = useCallback((project: Project) => {
    router.push(projectIssuesRoute(project));
  }, []);

  const handleRowClick = useCallback(
    (project: Project, e: React.MouseEvent) => {
      const route = router.resolve(projectIssuesRoute(project));
      if (e.ctrlKey || e.metaKey) {
        window.open(route.fullPath, "_blank");
      } else {
        router.push(projectIssuesRoute(project));
      }
    },
    []
  );

  const handleProjectAction = useCallback(() => {
    fetchProjects(true);
  }, [fetchProjects]);

  const pageSizeOptions = getPageSizeOptions();

  return (
    <div className="py-4 flex flex-col">
      {/* Header: Search + Create */}
      <div className="flex items-center justify-between px-4 pb-2 gap-x-2">
        <AdvancedSearch
          params={searchParams}
          scopeOptions={scopeOptions}
          placeholder={t("project.filter-projects")}
          onParamsChange={setSearchParams}
        />
        <PermissionGuard permissions={["bb.projects.create"]}>
          <Button
            disabled={!canCreate}
            onClick={() => setShowCreateDrawer(true)}
          >
            <Plus className="h-4 w-4 mr-1" />
            {t("common.create")}
          </Button>
        </PermissionGuard>
      </div>

      <div className="overflow-x-auto border-y border-block-border">
        <ProjectTable
          className="min-w-[700px]"
          projectList={projects}
          keyword={searchText}
          loading={loading}
          showSelection={canDelete}
          showLabels
          showActions
          renderActions={(project) => (
            <ProjectActionDropdown
              project={project}
              onAction={handleProjectAction}
            />
          )}
          selectedProjectNames={Array.from(selectedNames)}
          onSelectedChange={(names) => setSelectedNames(new Set(names))}
          sortKey={sortKey}
          sortOrder={sortOrder}
          onSortChange={(key) => toggleSort(key)}
          onRowClick={(project, e) => handleRowClick(project, e)}
        />
      </div>

      <div className="mt-4 mx-2">
        <PagedTableFooter
          pageSize={pageSize}
          pageSizeOptions={pageSizeOptions}
          onPageSizeChange={setPageSize}
          hasMore={hasMore}
          isFetchingMore={isFetchingMore}
          onLoadMore={loadMore}
        />
      </div>

      {/* Batch operations bar (sticky at bottom; rendered after the
          table so selection doesn't shift table position) */}
      {canDelete && (
        <SelectionActionBar
          count={selectedProjectList.length}
          label={t("common.n-selected", { n: selectedProjectList.length })}
          allSelected={
            projects.length > 0 &&
            projects.every((p) => selectedNames.has(p.name))
          }
          onToggleSelectAll={() => {
            const allOnPage =
              projects.length > 0 &&
              projects.every((p) => selectedNames.has(p.name));
            if (allOnPage) setSelectedNames(new Set());
            else setSelectedNames(new Set(projects.map((p) => p.name)));
          }}
          actions={batchActions}
        />
      )}

      {/* Modals (portaled, position-independent) */}
      <ProjectCreateDialog
        open={showCreateDrawer}
        onClose={() => setShowCreateDrawer(false)}
        onCreated={handleCreated}
      />

      <ConfirmDialog
        open={showArchiveConfirm}
        variant="warning"
        title={t("project.batch.archive.title", {
          count: selectedProjectList.length,
        })}
        description={t("project.batch.archive.description")}
        okText={t("common.archive")}
        onOk={handleBatchArchive}
        onCancel={() => setShowArchiveConfirm(false)}
      >
        <ProjectListPreview
          projects={selectedProjectList}
          iconColor="text-success"
        />
      </ConfirmDialog>

      <ConfirmDialog
        open={showRestoreConfirm}
        variant="warning"
        title={t("common.restore")}
        description={t("common.restore")}
        okText={t("common.restore")}
        onOk={handleBatchRestore}
        onCancel={() => setShowRestoreConfirm(false)}
      >
        <ProjectListPreview
          projects={selectedProjectList}
          iconColor="text-success"
        />
      </ConfirmDialog>

      <ConfirmDialog
        open={showDeleteConfirm}
        variant="error"
        title={t("project.batch.delete.title", {
          count: selectedProjectList.length,
        })}
        description={t("project.batch.delete.description")}
        okText={t("common.delete")}
        onOk={handleBatchDelete}
        onCancel={() => setShowDeleteConfirm(false)}
      >
        <div className="flex flex-col gap-y-3">
          <ProjectListPreview
            projects={selectedProjectList}
            iconColor="text-error"
          />
          <div className="rounded-sm border border-error bg-error/5 p-3">
            <p className="text-sm font-medium text-error">
              {t("common.cannot-undo-this-action")}
            </p>
            <p className="text-sm text-error/80 mt-1">
              {t("project.batch.delete.warning")}
            </p>
          </div>
        </div>
      </ConfirmDialog>
    </div>
  );
}
