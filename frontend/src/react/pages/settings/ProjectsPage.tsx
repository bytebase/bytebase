import { Code, ConnectError } from "@connectrpc/connect";
import { sortBy, uniq } from "lodash-es";
import {
  Archive,
  Check,
  ChevronDown,
  EllipsisVertical,
  Plus,
  Trash2,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AdvancedSearch,
  getValueFromScopes,
  type ScopeOption,
  type SearchParams,
} from "@/react/components/AdvancedSearch";
import {
  ResourceIdField,
  type ResourceIdFieldRef,
} from "@/react/components/ResourceIdField";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useAuthStore,
  useCurrentUserV1,
  useProjectV1Store,
  useUIStateStore,
} from "@/store";
import { getProjectName, projectNamePrefix } from "@/store/modules/v1/common";
import { unknownProject } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  extractProjectResourceName,
  getDefaultPagination,
  hasWorkspacePermissionV2,
} from "@/utils";

// ============================================================
// i18n helpers for vue-i18n compat
// ============================================================

// vue-i18n uses "@:key" for linked messages and "singular | plural" for pluralization.
// react-i18next doesn't support either, so we resolve them manually.

function resolveLinkedMessages(
  value: string,
  t: (key: string) => string
): string {
  return value.replace(/@:(?:\{'([^']+)'\}|(\S+))/g, (_match, quoted, plain) =>
    t(quoted ?? plain)
  );
}

function tVue(
  t: (key: string, options?: Record<string, unknown>) => string,
  key: string,
  options?: Record<string, unknown>
): string {
  const raw = t(key, { ...options, interpolation: { escapeValue: false } });
  // Handle vue-i18n pipe pluralization: "singular | plural"
  if (raw.includes(" | ")) {
    const parts = raw.split(" | ");
    const count = typeof options?.count === "number" ? options.count : 1;
    const chosen = count === 1 ? parts[0] : parts[parts.length - 1];
    // Re-interpolate {count} etc.
    const interpolated = chosen.replace(/\{(\w+)\}/g, (_m, k) =>
      String(options?.[k] ?? `{${k}}`)
    );
    return resolveLinkedMessages(interpolated, t);
  }
  return resolveLinkedMessages(raw, t);
}

// ============================================================
// Escape key stack for overlays
// ============================================================

// Track the topmost overlay's escape handler. Each overlay registers once
// on mount and unregisters on unmount. The ref ensures the latest callback
// is always invoked without re-registering the listener.
const escapeStack: React.RefObject<(() => void) | null>[] = [];

function useEscapeKey(onEscape: () => void) {
  const callbackRef = useRef(onEscape);
  callbackRef.current = onEscape;

  useEffect(() => {
    escapeStack.push(callbackRef);
    const handler = (e: KeyboardEvent) => {
      if (
        e.key === "Escape" &&
        escapeStack[escapeStack.length - 1] === callbackRef
      ) {
        callbackRef.current?.();
      }
    };
    document.addEventListener("keydown", handler);
    return () => {
      document.removeEventListener("keydown", handler);
      const idx = escapeStack.indexOf(callbackRef);
      if (idx >= 0) escapeStack.splice(idx, 1);
    };
  }, []);
}

// ============================================================
// Pagination helpers
// ============================================================

const PAGE_SIZE_OPTIONS = [10, 50, 100, 200, 500];

function getPageSizeOptions(): number[] {
  const defaultSize = getDefaultPagination();
  return sortBy(uniq([defaultSize, ...PAGE_SIZE_OPTIONS]));
}

function useSessionPageSize(
  sessionKey: string
): [number, (size: number) => void] {
  const currentUser = useCurrentUserV1();
  const email = useVueState(() => currentUser.value.email);
  const storageKey = `bb.paged-table.${sessionKey}.${email}`;

  const [pageSize, setPageSize] = useState<number>(() => {
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored) {
        const parsed = JSON.parse(stored);
        const size = parsed?.pageSize;
        const options = getPageSizeOptions();
        if (typeof size === "number" && options.includes(size)) {
          return Math.max(options[0], size);
        }
      }
    } catch {
      // ignore
    }
    return getPageSizeOptions()[0];
  });

  const updatePageSize = useCallback(
    (size: number) => {
      setPageSize(size);
      try {
        localStorage.setItem(storageKey, JSON.stringify({ pageSize: size }));
      } catch {
        // ignore
      }
    },
    [storageKey]
  );

  return [pageSize, updatePageSize];
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
  const onClose = useCallback(() => onCancel(), [onCancel]);
  useEscapeKey(onClose);

  if (!open) return null;

  const borderColor = variant === "error" ? "border-error" : "border-warning";
  const okBg =
    variant === "error"
      ? "bg-error hover:bg-error-hover text-white"
      : "bg-warning hover:bg-warning-hover text-white";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onCancel} />
      <div
        className={cn(
          "relative bg-white rounded-lg shadow-lg max-w-lg w-full mx-4 border-t-4",
          borderColor
        )}
      >
        <div className="p-6">
          <h3 className="text-lg font-semibold mb-2">{title}</h3>
          <p className="text-sm text-control-light mb-4">{description}</p>
          {children}
        </div>
        <div className="flex justify-end gap-x-2 px-6 pb-6">
          <Button variant="outline" onClick={onCancel}>
            {t("common.cancel")}
          </Button>
          <button
            className={cn(
              "inline-flex items-center justify-center rounded-md px-4 py-2 text-sm font-medium",
              okBg
            )}
            onClick={onOk}
          >
            {okText}
          </button>
        </div>
      </div>
    </div>
  );
}

// ============================================================
// CreateProjectDrawer
// ============================================================

function CreateProjectDrawer({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: (project: Project) => void;
}) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const [title, setTitle] = useState("");
  const [resourceId, setResourceId] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [isResourceIdValid, setIsResourceIdValid] = useState(false);
  const resourceIdFieldRef = useRef<ResourceIdFieldRef>(null);

  const closeDrawer = useCallback(() => {
    onClose();
    setTitle("");
    setResourceId("");
    setIsCreating(false);
  }, [onClose]);

  useEscapeKey(closeDrawer);

  const allowCreate = useMemo(() => {
    if (!title.trim()) return false;
    if (!isResourceIdValid) return false;
    if (!hasWorkspacePermissionV2("bb.projects.create")) return false;
    return true;
  }, [title, isResourceIdValid]);

  const handleCreate = useCallback(async () => {
    if (!allowCreate || isCreating) return;
    try {
      setIsCreating(true);
      const project = { ...unknownProject(), title };
      const created = await projectStore.createProject(project, resourceId);

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.create-modal.success-prompt", {
          name: created.title,
        }),
      });

      onCreated(created);
      closeDrawer();
    } catch (error) {
      if (error instanceof ConnectError && error.code === Code.AlreadyExists) {
        resourceIdFieldRef.current?.addValidationError(
          (error as ConnectError).message
        );
      } else {
        throw error;
      }
    } finally {
      setIsCreating(false);
    }
  }, [
    allowCreate,
    isCreating,
    title,
    resourceId,
    projectStore,
    t,
    onCreated,
    closeDrawer,
  ]);

  const validate = useCallback(
    async (id: string) => {
      try {
        await projectStore.getOrFetchProjectByName(
          `${projectNamePrefix}${id}`,
          true
        );
        return [
          {
            type: "error" as const,
            message: t("resource-id.validation.duplicated", {
              resource: "project",
            }),
          },
        ];
      } catch {
        return [];
      }
    },
    [projectStore, t]
  );

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="fixed inset-0 bg-black/50" onClick={closeDrawer} />
      <div className="ml-auto relative bg-white w-[40rem] max-w-[100vw] h-full shadow-lg flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-control-border">
          <h2 className="text-lg font-semibold">
            {tVue(t, "quick-action.create-project")}
          </h2>
          <button
            className="p-1 hover:bg-control-bg rounded"
            onClick={closeDrawer}
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-6">
          <div className="flex flex-col gap-y-6">
            <div>
              <label className="text-base leading-6 font-medium text-control">
                {tVue(t, "project.create-modal.project-name")}
                <span className="text-error ml-0.5">*</span>
              </label>
              <Input
                className="mt-2 mb-1"
                value={title}
                maxLength={200}
                placeholder={tVue(t, "project.create-modal.project-name")}
                onChange={(e) => setTitle(e.target.value)}
              />
              <ResourceIdField
                ref={resourceIdFieldRef}
                value={resourceId}
                resourceType="project"
                resourceName={title}
                resourceTitle={title}
                validate={validate}
                onChange={setResourceId}
                onValidationChange={setIsResourceIdValid}
              />
            </div>
          </div>

          {isCreating && (
            <div className="absolute inset-0 bg-white/50 flex justify-center items-center">
              <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
            </div>
          )}
        </div>

        <div className="flex justify-end items-center gap-x-3 px-6 py-4 border-t border-control-border">
          <Button variant="ghost" onClick={closeDrawer}>
            {t("common.cancel")}
          </Button>
          <Button disabled={!allowCreate} onClick={handleCreate}>
            {t("common.create")}
          </Button>
        </div>
      </div>
    </div>
  );
}

// ============================================================
// BatchOperationsBar
// ============================================================

function BatchOperationsBar({
  selectedProjects,
  onUpdate,
}: {
  selectedProjects: Project[];
  onUpdate: () => void;
}) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const [showArchiveConfirm, setShowArchiveConfirm] = useState(false);
  const [showRestoreConfirm, setShowRestoreConfirm] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  // Determine which actions to show based on selected projects' actual state
  const hasActiveProjects = selectedProjects.some(
    (p) => p.state === State.ACTIVE
  );
  const hasArchivedProjects = selectedProjects.some(
    (p) => p.state === State.DELETED
  );

  const handleBatchArchive = useCallback(async () => {
    try {
      const activeProjects = selectedProjects.filter(
        (p) => p.state === State.ACTIVE
      );
      await projectStore.batchDeleteProjects(activeProjects.map((p) => p.name));
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: tVue(t, "project.batch.archive.success", {
          count: activeProjects.length,
        }),
      });
      setShowArchiveConfirm(false);
      onUpdate();
    } catch (error: unknown) {
      const err = error as { message?: string };
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("project.batch.archive.error"),
        description: err.message,
      });
    }
  }, [selectedProjects, projectStore, t, onUpdate]);

  const handleBatchRestore = useCallback(async () => {
    try {
      const archivedProjects = selectedProjects.filter(
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
      onUpdate();
    } catch (error: unknown) {
      const err = error as { message?: string };
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.restore"),
        description: err.message,
      });
    }
  }, [selectedProjects, projectStore, t, onUpdate]);

  const handleBatchDelete = useCallback(async () => {
    try {
      await projectStore.batchPurgeProjects(
        selectedProjects.map((p) => p.name)
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: tVue(t, "project.batch.delete.success", {
          count: selectedProjects.length,
        }),
      });
      setShowDeleteConfirm(false);
      onUpdate();
    } catch (error: unknown) {
      const err = error as { message?: string };
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("project.batch.delete.error"),
        description: err.message,
      });
    }
  }, [selectedProjects, projectStore, t, onUpdate]);

  if (selectedProjects.length === 0) return null;

  return (
    <>
      <div className="text-sm flex flex-col lg:flex-row items-start lg:items-center bg-blue-100 py-3 px-4 text-main gap-y-2 gap-x-4 overflow-x-auto">
        <span className="whitespace-nowrap">
          {tVue(t, "project.batch.selected", {
            count: selectedProjects.length,
          })}
        </span>
        <div className="flex items-center gap-x-2">
          {hasActiveProjects && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowArchiveConfirm(true)}
            >
              <Archive className="h-4 w-4 mr-1" />
              {t("common.archive")}
            </Button>
          )}
          {hasArchivedProjects && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowRestoreConfirm(true)}
            >
              {t("common.restore")}
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShowDeleteConfirm(true)}
          >
            <Trash2 className="h-4 w-4 mr-1" />
            {t("common.delete")}
          </Button>
        </div>
      </div>

      <ConfirmDialog
        open={showArchiveConfirm}
        variant="warning"
        title={tVue(t, "project.batch.archive.title", {
          count: selectedProjects.length,
        })}
        description={t("project.batch.archive.description")}
        okText={t("common.archive")}
        onOk={handleBatchArchive}
        onCancel={() => setShowArchiveConfirm(false)}
      >
        <ProjectListPreview
          projects={selectedProjects}
          iconColor="text-green-600"
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
          projects={selectedProjects}
          iconColor="text-green-600"
        />
      </ConfirmDialog>

      <ConfirmDialog
        open={showDeleteConfirm}
        variant="error"
        title={tVue(t, "project.batch.delete.title", {
          count: selectedProjects.length,
        })}
        description={t("project.batch.delete.description")}
        okText={t("common.delete")}
        onOk={handleBatchDelete}
        onCancel={() => setShowDeleteConfirm(false)}
      >
        <div className="flex flex-col gap-y-3">
          <ProjectListPreview
            projects={selectedProjects}
            iconColor="text-red-600"
          />
          <div className="rounded-md border border-error bg-error/5 p-3">
            <p className="text-sm font-medium text-error">
              {t("common.cannot-undo-this-action")}
            </p>
            <p className="text-sm text-error/80 mt-1">
              {t("project.batch.delete.warning")}
            </p>
          </div>
        </div>
      </ConfirmDialog>
    </>
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
    <div className="max-h-40 overflow-y-auto border rounded-sm p-2 bg-gray-50">
      <div className="flex flex-col gap-y-1">
        {projects.map((project) => (
          <div key={project.name} className="text-sm flex items-center gap-x-2">
            <Check className={cn("w-3 h-3", iconColor)} />
            <span>{project.title}</span>
            <span className="text-gray-500">
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
  const [open, setOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const canDelete = hasWorkspacePermissionV2("bb.projects.delete");
  const canUndelete = hasWorkspacePermissionV2("bb.projects.undelete");

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  const handleArchive = useCallback(async () => {
    setOpen(false);
    await projectStore.archiveProject(project);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.archived"),
    });
    onAction();
  }, [project, projectStore, t, onAction]);

  const handleRestore = useCallback(async () => {
    setOpen(false);
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
    <div ref={dropdownRef} className="relative">
      <button
        className="p-1 hover:bg-control-bg rounded"
        onClick={(e) => {
          e.stopPropagation();
          setOpen(!open);
        }}
      >
        <EllipsisVertical className="h-4 w-4" />
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-1 bg-white border border-control-border rounded-md shadow-lg z-10 min-w-[120px]">
          {isActive && canDelete && (
            <button
              className="w-full text-left px-3 py-2 text-sm hover:bg-control-bg flex items-center gap-x-2"
              onClick={(e) => {
                e.stopPropagation();
                handleArchive();
              }}
            >
              <Archive className="h-4 w-4" />
              {t("common.archive")}
            </button>
          )}
          {!isActive && canUndelete && (
            <button
              className="w-full text-left px-3 py-2 text-sm hover:bg-control-bg flex items-center gap-x-2"
              onClick={(e) => {
                e.stopPropagation();
                handleRestore();
              }}
            >
              {t("common.restore")}
            </button>
          )}
        </div>
      )}
    </div>
  );
}

// ============================================================
// HighlightText
// ============================================================

function HighlightText({ text, keyword }: { text: string; keyword: string }) {
  if (!keyword) return <>{text}</>;

  const lowerText = text.toLowerCase();
  const lowerKeyword = keyword.toLowerCase();
  const index = lowerText.indexOf(lowerKeyword);

  if (index === -1) return <>{text}</>;

  return (
    <>
      {text.substring(0, index)}
      <span className="bg-yellow-100">
        {text.substring(index, index + keyword.length)}
      </span>
      {text.substring(index + keyword.length)}
    </>
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
      description: t("common.state"),
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
      description: t("common.labels"),
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
      router.replace({ query: queryString ? { q: queryString } : {} });
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
      const selectableProjects = projects.filter(
        (p) => extractProjectResourceName(p.name) !== "default"
      );
      if (prev.size === selectableProjects.length) {
        return new Set();
      }
      return new Set(selectableProjects.map((p) => p.name));
    });
  }, [projects]);

  const handleBatchOperation = useCallback(() => {
    setSelectedNames(new Set());
    fetchProjects(true);
  }, [fetchProjects]);

  // Create drawer
  const [showCreateDrawer, setShowCreateDrawer] = useState(false);
  const canCreate = hasWorkspacePermissionV2("bb.projects.create");
  const canDelete = hasWorkspacePermissionV2("bb.projects.delete");

  const handleCreated = useCallback((project: Project) => {
    router.push({ path: `/${project.name}` });
  }, []);

  const handleRowClick = useCallback(
    (project: Project, e: React.MouseEvent) => {
      const route = router.resolve({
        name: PROJECT_V1_ROUTE_DETAIL,
        params: { projectId: getProjectName(project.name) },
      });
      if (e.ctrlKey || e.metaKey) {
        window.open(route.fullPath, "_blank");
      } else {
        router.push(route);
      }
    },
    []
  );

  const handleProjectAction = useCallback(() => {
    fetchProjects(true);
  }, [fetchProjects]);

  const renderSortIndicator = (columnKey: string) => {
    if (sortKey !== columnKey) {
      return <ChevronDown className="h-3 w-3 text-gray-300" />;
    }
    return (
      <ChevronDown
        className={cn(
          "h-3 w-3 text-accent transition-transform",
          sortOrder === "asc" && "rotate-180"
        )}
      />
    );
  };

  const selectableProjects = useMemo(
    () =>
      projects.filter((p) => extractProjectResourceName(p.name) !== "default"),
    [projects]
  );

  const allSelected =
    selectableProjects.length > 0 &&
    selectedNames.size === selectableProjects.length;

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
        {canCreate && (
          <Button onClick={() => setShowCreateDrawer(true)}>
            <Plus className="h-4 w-4 mr-1" />
            {tVue(t, "quick-action.new-project")}
          </Button>
        )}
      </div>

      {/* Batch operations */}
      {canDelete && (
        <BatchOperationsBar
          selectedProjects={selectedProjectList}
          onUpdate={handleBatchOperation}
        />
      )}

      {/* Table */}
      <div className="flex flex-col gap-y-4">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-gray-50 border-b border-control-border">
              {canDelete && (
                <th className="w-12 px-4 py-2">
                  <input
                    type="checkbox"
                    checked={allSelected}
                    onChange={toggleSelectAll}
                    className="rounded border-control-border"
                  />
                </th>
              )}
              <th className="px-4 py-2 text-left font-medium min-w-[128px]">
                {t("common.id")}
              </th>
              <th
                className="px-4 py-2 text-left font-medium min-w-[200px] cursor-pointer select-none"
                onClick={() => toggleSort("title")}
              >
                <div className="flex items-center gap-x-1">
                  {tVue(t, "project.table.name")}
                  {renderSortIndicator("title")}
                </div>
              </th>
              <th className="px-4 py-2 text-left font-medium min-w-[240px] hidden md:table-cell">
                {t("common.labels")}
              </th>
              <th className="w-[50px]" />
            </tr>
          </thead>
          <tbody>
            {loading && projects.length === 0 ? (
              <tr>
                <td
                  colSpan={canDelete ? 5 : 4}
                  className="px-4 py-8 text-center text-control-placeholder"
                >
                  <div className="flex items-center justify-center gap-x-2">
                    <div className="animate-spin h-4 w-4 border-2 border-accent border-t-transparent rounded-full" />
                    {t("common.loading")}
                  </div>
                </td>
              </tr>
            ) : projects.length === 0 ? (
              <tr>
                <td
                  colSpan={canDelete ? 5 : 4}
                  className="px-4 py-8 text-center text-control-placeholder"
                >
                  {t("common.no-data")}
                </td>
              </tr>
            ) : (
              projects.map((project, i) => {
                const resourceName = extractProjectResourceName(project.name);
                const isDefault = resourceName === "default";
                const isSelected = selectedNames.has(project.name);

                return (
                  <tr
                    key={project.name}
                    className={cn(
                      "border-b last:border-b-0 cursor-pointer hover:bg-gray-50",
                      i % 2 === 1 && "bg-gray-50/50"
                    )}
                    onClick={(e) => handleRowClick(project, e)}
                  >
                    {canDelete && (
                      <td className="w-12 px-4 py-2">
                        <input
                          type="checkbox"
                          checked={isSelected}
                          disabled={isDefault}
                          onChange={() => toggleSelection(project.name)}
                          onClick={(e) => e.stopPropagation()}
                          className="rounded border-control-border disabled:opacity-50"
                        />
                      </td>
                    )}
                    <td className="px-4 py-2">
                      <HighlightText text={resourceName} keyword={searchText} />
                    </td>
                    <td className="px-4 py-2">
                      <HighlightText
                        text={project.title}
                        keyword={searchText}
                      />
                    </td>
                    <td className="px-4 py-2 hidden md:table-cell">
                      <LabelsDisplay labels={project.labels} />
                    </td>
                    <td className="px-4 py-2">
                      <div
                        className="flex justify-end"
                        onClick={(e) => e.stopPropagation()}
                      >
                        <ProjectActionDropdown
                          project={project}
                          onAction={handleProjectAction}
                        />
                      </div>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>

        {/* Pagination footer */}
        <div className="flex items-center justify-end gap-x-2 mx-4">
          <div className="flex items-center gap-x-2">
            <span className="text-sm text-control-light">
              {t("common.rows-per-page")}
            </span>
            <select
              className="border border-control-border rounded-md text-sm pl-2 pr-6 py-1 min-w-[5rem]"
              value={pageSize}
              onChange={(e) => setPageSize(Number(e.target.value))}
            >
              {pageSizeOptions.map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
          </div>
          {hasMore && (
            <Button
              variant="ghost"
              size="sm"
              disabled={isFetchingMore}
              onClick={loadMore}
            >
              <span className="text-sm text-control-light">
                {isFetchingMore ? t("common.loading") : t("common.load-more")}
              </span>
            </Button>
          )}
        </div>
      </div>

      {/* Create drawer */}
      <CreateProjectDrawer
        open={showCreateDrawer}
        onClose={() => setShowCreateDrawer(false)}
        onCreated={handleCreated}
      />
    </div>
  );
}

// ============================================================
// LabelsDisplay
// ============================================================

function LabelsDisplay({ labels }: { labels: { [key: string]: string } }) {
  const entries = Object.entries(labels);
  if (entries.length === 0)
    return <span className="text-control-placeholder">-</span>;

  const displayEntries = entries.slice(0, 3);
  const hasMore = entries.length > 3;

  return (
    <div className="flex items-center gap-x-1">
      {displayEntries.map(([key, value]) => (
        <span key={key} className="rounded-lg bg-gray-100 py-0.5 px-2 text-sm">
          {key}:{value}
        </span>
      ))}
      {hasMore && <span>...</span>}
    </div>
  );
}
