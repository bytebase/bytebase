import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import {
  ChevronDown,
  ChevronUp,
  EllipsisVertical,
  GraduationCap,
  Plus,
  RefreshCw,
  SquareStack,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AdvancedSearch,
  getValueFromScopes,
  type ScopeOption,
  type SearchParams,
} from "@/react/components/AdvancedSearch";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { InstanceAssignmentSheet } from "@/react/components/InstanceAssignmentSheet";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import {
  type SelectionAction,
  SelectionActionBar,
} from "@/react/components/SelectionActionBar";
import { Alert } from "@/react/components/ui/alert";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
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
import { useColumnWidths } from "@/react/hooks/useColumnWidths";
import { PagedTableFooter } from "@/react/hooks/usePagedData";
import {
  getPageSizeOptions,
  useSessionPageSize,
} from "@/react/hooks/useSessionPageSize";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { INSTANCE_ROUTE_CREATE } from "@/router/dashboard/instance";
import {
  featureToRef,
  pushNotification,
  useActuatorV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useSubscriptionV1Store,
  useUIStateStore,
} from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import type { InstanceFilter } from "@/store/modules/v1/instance";
import {
  isValidInstanceName,
  NULL_ENVIRONMENT_NAME,
  UNKNOWN_ENVIRONMENT_NAME,
  unknownEnvironment,
} from "@/types";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { UpdateInstanceRequestSchema } from "@/types/proto-es/v1/instance_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  engineNameV1,
  hasWorkspacePermissionV2,
  hexToRgb,
  hostPortOfDataSource,
  hostPortOfInstanceV1,
  supportedEngineV1List,
} from "@/utils";

const ASSIGN_LICENSE_QUERY = "assignLicense";
const ASSIGN_LICENSE_INSTANCES_QUERY = "instances";

interface InstanceColumn {
  key: string;
  title: React.ReactNode;
  defaultWidth: number;
  minWidth?: number;
  resizable?: boolean;
  sortable?: boolean;
  sortKey?: string;
  cellClassName?: string;
  render: (instance: Instance) => React.ReactNode;
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

// ============================================================
// EnvironmentName
// ============================================================

function EnvironmentName({ environmentName }: { environmentName: string }) {
  const { t } = useTranslation();
  const environmentStore = useEnvironmentV1Store();
  const environment = useVueState(() =>
    environmentStore.getEnvironmentByName(
      environmentName || NULL_ENVIRONMENT_NAME
    )
  );

  const isUnset =
    environment.name === UNKNOWN_ENVIRONMENT_NAME ||
    environment.name === NULL_ENVIRONMENT_NAME;

  const hasEnvTierFeature = useVueState(
    () => featureToRef(PlanFeature.FEATURE_ENVIRONMENT_TIERS).value
  );
  const isProtected =
    hasEnvTierFeature && environment.tags?.protected === "protected";

  const bgColorRgb = environment.color ? hexToRgb(environment.color) : null;

  return (
    <span
      className="inline-flex items-center gap-x-1"
      style={
        bgColorRgb && !isUnset
          ? {
              backgroundColor: `rgba(${bgColorRgb.join(", ")}, 0.1)`,
              color: `rgb(${bgColorRgb.join(", ")})`,
              padding: "0 6px",
              borderRadius: "4px",
            }
          : undefined
      }
    >
      <span className="truncate">
        {isUnset ? (
          <span className="text-control-light italic">
            {t("common.unassigned")}
          </span>
        ) : (
          environment.title
        )}
      </span>
      {isProtected && !isUnset && (
        <svg
          className="w-4 h-4 shrink-0 text-current"
          viewBox="0 0 20 20"
          fill="currentColor"
        >
          <path
            fillRule="evenodd"
            d="M10 1.944A11.954 11.954 0 012.166 5C2.056 5.649 2 6.319 2 7c0 5.225 3.34 9.67 8 11.317C14.66 16.67 18 12.225 18 7c0-.682-.057-1.351-.166-2.001A11.954 11.954 0 0110 1.944zM11 14a1 1 0 11-2 0 1 1 0 012 0zm0-7a1 1 0 10-2 0v3a1 1 0 102 0V7z"
            clipRule="evenodd"
          />
        </svg>
      )}
    </span>
  );
}

// ============================================================
// InstanceActionDropdown
// ============================================================

function InstanceActionDropdown({
  instance,
  onAction,
}: {
  instance: Instance;
  onAction: () => void;
}) {
  const { t } = useTranslation();
  const instanceStore = useInstanceV1Store();
  const [showArchiveConfirm, setShowArchiveConfirm] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [forceArchive, setForceArchive] = useState(false);

  const canArchive = hasWorkspacePermissionV2("bb.instances.delete");
  const canRestore = hasWorkspacePermissionV2("bb.instances.undelete");

  const handleArchive = useCallback(async () => {
    try {
      await instanceStore.archiveInstance(instance, forceArchive);
      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("instance.successfully-archived-instance", {
          0: instance.title,
        }),
      });
      setShowArchiveConfirm(false);
      setForceArchive(false);
      onAction();
    } catch (error: unknown) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.archive"),
        description: (error as { message?: string }).message,
      });
    }
  }, [instance, instanceStore, forceArchive, t, onAction]);

  const handleRestore = useCallback(async () => {
    try {
      await instanceStore.restoreInstance(instance);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("instance.successfully-restored-instance", {
          0: instance.title,
        }),
      });
      onAction();
    } catch (error: unknown) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.restore"),
        description: (error as { message?: string }).message,
      });
    }
  }, [instance, instanceStore, t, onAction]);

  const handleDelete = useCallback(async () => {
    try {
      await instanceStore.deleteInstance(instance.name);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.deleted"),
      });
      setShowDeleteConfirm(false);
      onAction();
    } catch (error: unknown) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.delete"),
        description: (error as { message?: string }).message,
      });
    }
  }, [instance, instanceStore, t, onAction]);

  if (!canArchive && !canRestore) return null;

  const isActive = instance.state === State.ACTIVE;

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          className="p-1 rounded-xs hover:bg-control-bg outline-hidden"
          onClick={(e) => e.stopPropagation()}
        >
          <EllipsisVertical className="h-4 w-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          {isActive && canArchive && (
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation();
                setShowArchiveConfirm(true);
              }}
            >
              {t("common.archive")}
            </DropdownMenuItem>
          )}
          {!isActive && canRestore && (
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation();
                handleRestore();
              }}
            >
              {t("common.restore")}
            </DropdownMenuItem>
          )}
          {(canArchive || canRestore) && (
            <DropdownMenuItem
              className="text-error"
              onClick={(e) => {
                e.stopPropagation();
                setShowDeleteConfirm(true);
              }}
            >
              {t("common.delete")}
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      <ConfirmDialog
        open={showArchiveConfirm}
        variant="warning"
        title={t("instance.archive-instance-instance-name", {
          0: instance.title,
        })}
        description={t("instance.archived-instances-will-not-be-displayed")}
        okText={t("common.archive")}
        onOk={handleArchive}
        onCancel={() => {
          setShowArchiveConfirm(false);
          setForceArchive(false);
        }}
      >
        <label className="flex items-center gap-x-2 text-sm text-control-light mt-2">
          <Checkbox
            checked={forceArchive}
            onCheckedChange={(checked) => setForceArchive(checked)}
          />
          {t("instance.force-archive-description")}
        </label>
      </ConfirmDialog>

      <ConfirmDialog
        open={showDeleteConfirm}
        variant="error"
        title={t("common.delete-resource", {
          type: instance.title,
        })}
        description={t("common.cannot-undo-this-action")}
        okText={t("common.delete")}
        onOk={handleDelete}
        onCancel={() => setShowDeleteConfirm(false)}
      />
    </>
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
        <span
          key={key}
          className="rounded-xs bg-control-bg py-0.5 px-2 text-sm"
        >
          {key}:{value}
        </span>
      ))}
      {hasMore && <span>...</span>}
    </div>
  );
}

// ============================================================
// EditEnvironmentSheet
// ============================================================

function EditEnvironmentSheet({
  open,
  onClose,
  onUpdate,
}: {
  open: boolean;
  onClose: () => void;
  onUpdate: (environment: string) => Promise<void>;
}) {
  const { t } = useTranslation();
  const environmentStore = useEnvironmentV1Store();
  const environments = useVueState(
    () => environmentStore.environmentList ?? []
  );
  const [selected, setSelected] = useState("");
  useEffect(() => {
    if (open) setSelected("");
  }, [open]);

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="narrow">
        <SheetHeader>
          <SheetTitle>{t("common.environment")}</SheetTitle>
        </SheetHeader>
        <SheetBody>
          <div className="flex flex-col gap-y-2">
            {environments.map((env) => (
              <label
                key={env.name}
                className={cn(
                  "flex items-center gap-x-3 px-3 py-2 rounded-xs cursor-pointer border",
                  selected === env.name
                    ? "border-accent bg-accent/5"
                    : "border-transparent hover:bg-control-bg"
                )}
              >
                <input
                  type="radio"
                  name="environment"
                  checked={selected === env.name}
                  onChange={() => setSelected(env.name)}
                  className="accent-accent"
                />
                <div className="flex items-center gap-x-2">
                  {env.color && (
                    <span
                      className="inline-block size-3 rounded-full"
                      style={{ backgroundColor: env.color }}
                    />
                  )}
                  <span>{env.title}</span>
                </div>
              </label>
            ))}
          </div>
        </SheetBody>
        <SheetFooter>
          <Button variant="ghost" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={!selected}
            onClick={async () => {
              await onUpdate(selected);
              onClose();
            }}
          >
            {t("common.update")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ============================================================
// SyncDropdown
// ============================================================

function SyncDropdown({
  disabled,
  onSync,
}: {
  disabled: boolean;
  onSync: (enableFullSync: boolean) => void;
}) {
  const { t } = useTranslation();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="outline"
            size="sm"
            className="rounded-full"
            disabled={disabled}
          >
            <RefreshCw className={cn("size-4", disabled && "animate-spin")} />
            {disabled ? t("instance.syncing") : t("instance.sync.self")}
            <ChevronDown className="size-3" />
          </Button>
        }
      />
      <DropdownMenuContent align="start" className="min-w-[200px]">
        <DropdownMenuItem
          title={t("instance.sync.sync-all-tip")}
          onClick={() => onSync(true)}
        >
          {t("instance.sync.sync-all")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onSync(false)}>
          {t("instance.sync.sync-new")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

// ============================================================
// InstancesPage (main)
// ============================================================

export function InstancesPage() {
  const { t } = useTranslation();
  const instanceStore = useInstanceV1Store();
  const subscriptionStore = useSubscriptionV1Store();
  const actuatorStore = useActuatorV1Store();

  // Search state
  const [searchParams, setSearchParams] = useState<SearchParams>(() => {
    const currentRoute = router.currentRoute.value;
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
          if (
            value &&
            [
              "environment",
              "engine",
              "label",
              "state",
              "host",
              "port",
            ].includes(id)
          ) {
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

  // Scope options
  const canUndelete = hasWorkspacePermissionV2("bb.instances.undelete");
  const environmentStore = useEnvironmentV1Store();
  const environments = useVueState(
    () => environmentStore.environmentList ?? []
  );
  const scopeOptions: ScopeOption[] = useMemo(() => {
    const options: ScopeOption[] = [
      {
        id: "environment",
        title: t("common.environment"),
        description: t("issue.advanced-search.scope.environment.description"),
        options: [unknownEnvironment(), ...environments].map((env) => {
          const isUnknown = env.name === UNKNOWN_ENVIRONMENT_NAME;
          return {
            value: env.id,
            keywords: isUnknown
              ? ["unassigned", "none", env.id]
              : [env.id, env.title],
            custom: true,
            render: () => <EnvironmentLabel environment={env} />,
          };
        }),
      },
      {
        id: "engine",
        title: t("database.engine"),
        description: t("issue.advanced-search.scope.engine.description"),
        options: supportedEngineV1List().map((engine) => ({
          value: Engine[engine],
          keywords: [Engine[engine].toLowerCase(), engineNameV1(engine)],
          custom: true,
          render: () => (
            <span className="inline-flex items-center gap-x-1.5">
              <EngineIcon engine={engine} className="h-4 w-4" />
              <span>{engineNameV1(engine)}</span>
            </span>
          ),
        })),
        allowMultiple: true,
      },
      {
        id: "label",
        title: t("common.labels"),
        description: t("issue.advanced-search.scope.label.description"),
        allowMultiple: true,
      },
      {
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
      },
      {
        id: "host",
        title: t("instance.advanced-search.scope.host.title"),
        description: t("instance.advanced-search.scope.host.description"),
      },
      {
        id: "port",
        title: t("instance.advanced-search.scope.port.title"),
        description: t("instance.advanced-search.scope.port.description"),
      },
    ];
    return options;
  }, [t, canUndelete, environments]);

  // Derived filter values — trivial lookups, no memo needed
  const searchText = searchParams.query;

  const stateFilterVal = getValueFromScopes(searchParams, "state");
  const selectedState =
    stateFilterVal === "DELETED"
      ? State.DELETED
      : stateFilterVal === "ALL"
        ? undefined
        : State.ACTIVE;

  const envVal = getValueFromScopes(searchParams, "environment");
  const selectedEnvironment = envVal
    ? `${environmentNamePrefix}${envVal}`
    : undefined;

  const selectedHost = getValueFromScopes(searchParams, "host") || undefined;
  const selectedPort = getValueFromScopes(searchParams, "port") || undefined;

  const selectedEngines = useMemo(
    () =>
      searchParams.scopes
        .filter((s) => s.id === "engine")
        .map((s) => Engine[s.value as keyof typeof Engine])
        .filter((e): e is Engine => e !== undefined),
    [searchParams]
  );

  const selectedLabels = useMemo(
    () =>
      searchParams.scopes.filter((s) => s.id === "label").map((s) => s.value),
    [searchParams]
  );

  // Mark instance visit on mount
  const uiStateStore = useUIStateStore();
  useEffect(() => {
    if (!uiStateStore.getIntroStateByKey("instance.visit")) {
      uiStateStore.saveIntroStateByKey({
        key: "instance.visit",
        newState: true,
      });
    }
  }, [uiStateStore]);

  // Sync search state to URL
  useEffect(() => {
    const parts: string[] = [];
    for (const scope of searchParams.scopes) {
      parts.push(`${scope.id}:${scope.value}`);
    }
    if (searchParams.query) parts.push(searchParams.query);
    const queryString = parts.join(" ");
    const currentQuery = router.currentRoute.value.query;
    const currentQueryString = currentQuery.q as string;
    if (queryString !== (currentQueryString ?? "")) {
      const nextQuery = { ...currentQuery, q: queryString };
      router.replace({ query: nextQuery });
    }
  }, [searchParams]);

  // Instance count warning
  const instanceCountLimit = useVueState(
    () => subscriptionStore.instanceCountLimit
  );
  const totalInstanceCount = useVueState(
    () => actuatorStore.totalInstanceCount
  );
  const hasSplitInstanceLicense = useVueState(
    () => subscriptionStore.hasSplitInstanceLicense
  );
  const quotaExhausted = totalInstanceCount >= instanceCountLimit;

  // Data fetching
  const [instances, setInstances] = useState<Instance[]>([]);
  const [loading, setLoading] = useState(true);
  const nextPageTokenRef = useRef("");
  const [hasMore, setHasMore] = useState(false);
  const [isFetchingMore, setIsFetchingMore] = useState(false);
  const [pageSize, setPageSize] = useSessionPageSize("bb.instance-table");
  const fetchIdRef = useRef(0);

  // Sort state
  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("asc");

  const orderBy = sortKey ? `${sortKey} ${sortOrder}` : "";

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

  const filter: InstanceFilter = useMemo(
    () => ({
      environment: selectedEnvironment,
      host: selectedHost,
      port: selectedPort,
      query: searchText,
      engines: selectedEngines,
      labels: selectedLabels.length > 0 ? selectedLabels : undefined,
      state: selectedState,
    }),
    [
      selectedEnvironment,
      selectedHost,
      selectedPort,
      searchText,
      selectedEngines,
      selectedLabels,
      selectedState,
    ]
  );

  const fetchInstances = useCallback(
    async (isRefresh: boolean) => {
      const currentFetchId = ++fetchIdRef.current;

      if (isRefresh) {
        setLoading(true);
      } else {
        setIsFetchingMore(true);
      }

      try {
        const token = isRefresh ? "" : nextPageTokenRef.current;
        const result = await instanceStore.fetchInstanceList({
          pageToken: token,
          pageSize,
          filter,
          orderBy,
        });

        if (currentFetchId !== fetchIdRef.current) return;

        if (isRefresh) {
          setInstances(result.instances);
        } else {
          setInstances((prev) => [...prev, ...result.instances]);
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
    [pageSize, filter, orderBy, instanceStore]
  );

  // Fetch on mount + re-fetch on filter/sort/pageSize changes
  const isFirstLoad = useRef(true);
  useEffect(() => {
    if (isFirstLoad.current) {
      isFirstLoad.current = false;
      fetchInstances(true);
      return;
    }
    const timer = setTimeout(() => fetchInstances(true), 300);
    return () => clearTimeout(timer);
  }, [fetchInstances]);

  const loadMore = useCallback(() => {
    if (nextPageTokenRef.current && !isFetchingMore) {
      fetchInstances(false);
    }
  }, [isFetchingMore, fetchInstances]);

  // Selection state
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set());

  const selectedInstanceList = useMemo(() => {
    if (selectedNames.size === 0) return [];
    return Array.from(selectedNames)
      .filter((name) => isValidInstanceName(name))
      .map((name) => instanceStore.getInstanceByName(name));
  }, [selectedNames, instanceStore]);

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
      if (prev.size === instances.length) return new Set();
      return new Set(instances.map((i) => i.name));
    });
  }, [instances]);

  // Batch operations — all mutation logic lives here in the parent
  const [syncing, setSyncing] = useState(false);
  const [showEditEnvDrawer, setShowEditEnvDrawer] = useState(false);
  const [showAssignLicenseSheet, setShowAssignLicenseSheet] = useState(false);
  const [assignLicenseNames, setAssignLicenseNames] = useState<string[]>([]);

  const handleSync = useCallback(
    async (enableFullSync: boolean) => {
      setSyncing(true);
      try {
        await instanceStore.batchSyncInstances(
          Array.from(selectedNames),
          enableFullSync
        );
        pushNotification({
          module: "bytebase",
          style: "INFO",
          title: t("db.start-to-sync-schema"),
        });
      } finally {
        setSyncing(false);
      }
    },
    [selectedNames, instanceStore, t]
  );

  const handleEnvironmentUpdate = useCallback(
    async (environment: string) => {
      try {
        await instanceStore.batchUpdateInstances(
          selectedInstanceList.map((instance) =>
            create(UpdateInstanceRequestSchema, {
              instance: { ...instance, environment },
              updateMask: create(FieldMaskSchema, { paths: ["environment"] }),
            })
          )
        );
        // Re-fetch the full list to get fresh data
        fetchInstances(true);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      } catch (error: unknown) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.environment"),
          description: (error as { message?: string }).message,
        });
      }
    },
    [selectedInstanceList, instanceStore, t, fetchInstances]
  );

  const handleRowAction = useCallback(() => {
    fetchInstances(true);
    setSelectedNames(new Set());
  }, [fetchInstances]);

  const handleAssignLicense = useCallback(
    (names: string[]) => {
      if (!hasSplitInstanceLicense) {
        return;
      }
      setAssignLicenseNames(names);
      setShowAssignLicenseSheet(true);
    },
    [hasSplitInstanceLicense]
  );

  useEffect(() => {
    const query = router.currentRoute.value.query;
    if (query[ASSIGN_LICENSE_QUERY] !== "1") {
      return;
    }

    const rawInstances = query[ASSIGN_LICENSE_INSTANCES_QUERY];
    const names =
      typeof rawInstances === "string"
        ? rawInstances.split(",").filter(Boolean)
        : Array.isArray(rawInstances)
          ? rawInstances.filter(
              (name): name is string => typeof name === "string"
            )
          : [];
    handleAssignLicense(names);

    const nextQuery = { ...query };
    delete nextQuery[ASSIGN_LICENSE_QUERY];
    delete nextQuery[ASSIGN_LICENSE_INSTANCES_QUERY];
    router.replace({ query: nextQuery });
  }, [handleAssignLicense]);

  // Data source toggle
  const [expandedDataSources, setExpandedDataSources] = useState<Set<string>>(
    new Set()
  );

  const toggleDataSource = useCallback((name: string) => {
    setExpandedDataSources((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }, []);

  const activatedInstanceCount = useVueState(
    () => actuatorStore.activatedInstanceCount
  );
  const navigateToCreate = useCallback(() => {
    if (instanceCountLimit <= activatedInstanceCount) {
      // Limit reached — the warning banner is already visible
      return;
    }
    router.push({ name: INSTANCE_ROUTE_CREATE });
  }, [instanceCountLimit, activatedInstanceCount]);

  const handleRowClick = useCallback(
    (instance: Instance, e: React.MouseEvent) => {
      const url = `/${instance.name}`;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
    []
  );

  const canCreate = hasWorkspacePermissionV2("bb.instances.create");
  const allSelected =
    instances.length > 0 && selectedNames.size === instances.length;
  const someSelected =
    selectedNames.size > 0 && selectedNames.size < instances.length;
  const pageSizeOptions = getPageSizeOptions();

  const columns: InstanceColumn[] = [
    {
      key: "select",
      title: (
        <Checkbox
          checked={someSelected ? "indeterminate" : allSelected}
          onCheckedChange={toggleSelectAll}
        />
      ),
      defaultWidth: 48,
      cellClassName: "px-4 py-2",
      render: (instance) => (
        <Checkbox
          checked={selectedNames.has(instance.name)}
          onCheckedChange={() => toggleSelection(instance.name)}
          onClick={(e) => e.stopPropagation()}
        />
      ),
    },
    {
      key: "title",
      title: t("common.name"),
      defaultWidth: 280,
      minWidth: 160,
      resizable: true,
      sortable: true,
      render: (instance) => (
        <div className="flex items-center gap-x-2 min-w-0">
          <EngineIcon engine={instance.engine} className="h-5 w-5" />
          <EllipsisText text={instance.title} />
        </div>
      ),
    },
    {
      key: "environment",
      title: t("common.environment"),
      defaultWidth: 220,
      minWidth: 120,
      resizable: true,
      sortable: true,
      render: (instance) => (
        <EnvironmentName environmentName={instance.environment ?? ""} />
      ),
    },
    {
      key: "address",
      title: t("common.address"),
      defaultWidth: 280,
      minWidth: 150,
      resizable: true,
      render: (instance) => {
        const isExpanded = expandedDataSources.has(instance.name);
        const hasMultipleDS = instance.dataSources.length > 1;
        return (
          <div className="flex items-start gap-x-2">
            <span className="truncate">
              {isExpanded
                ? instance.dataSources.map((ds, idx) => (
                    <div key={idx}>{hostPortOfDataSource(ds)}</div>
                  ))
                : hostPortOfInstanceV1(instance)}
            </span>
            {hasMultipleDS && (
              <button
                className="p-0.5 hover:bg-control-bg rounded-xs shrink-0"
                onClick={(e) => {
                  e.stopPropagation();
                  toggleDataSource(instance.name);
                }}
              >
                {isExpanded ? (
                  <ChevronUp className="w-4 h-4" />
                ) : (
                  <ChevronDown className="w-4 h-4" />
                )}
              </button>
            )}
          </div>
        );
      },
    },
    {
      key: "labels",
      title: t("common.labels"),
      defaultWidth: 240,
      minWidth: 150,
      resizable: true,
      render: (instance) => <LabelsDisplay labels={instance.labels} />,
    },
    {
      key: "license",
      title: t("subscription.instance-assignment.license"),
      defaultWidth: 100,
      render: (instance) => (instance.activation ? "Y" : ""),
    },
    {
      key: "actions",
      title: "",
      defaultWidth: 50,
      render: (instance) => (
        <div className="flex justify-end" onClick={(e) => e.stopPropagation()}>
          <InstanceActionDropdown
            instance={instance}
            onAction={handleRowAction}
          />
        </div>
      ),
    },
  ];

  const { widths, totalWidth, onResizeStart } = useColumnWidths(columns);

  return (
    <div className="py-4 flex flex-col">
      {/* Instance count warning */}
      {quotaExhausted && (
        <Alert
          variant="warning"
          className="mx-4 mb-4"
          title={t("subscription.usage.instance-count.title")}
          description={t("subscription.usage.instance-count.runoutof", {
            total: instanceCountLimit,
          })}
        />
      )}

      {/* Header: Search + Create */}
      <div className="flex items-center justify-between px-4 pb-2 gap-x-2">
        <AdvancedSearch
          params={searchParams}
          scopeOptions={scopeOptions}
          placeholder={t("instance.filter-instance-name")}
          onParamsChange={setSearchParams}
        />
        <PermissionGuard permissions={["bb.instances.create"]}>
          <Button disabled={!canCreate} onClick={navigateToCreate}>
            <Plus className="h-4 w-4 mr-1" />
            {t("common.create")}
          </Button>
        </PermissionGuard>
      </div>

      <div className="overflow-x-auto border-y border-block-border">
        <Table className="table-fixed" style={{ minWidth: `${totalWidth}px` }}>
          <colgroup>
            {widths.map((w, i) => (
              <col key={columns[i].key} style={{ width: `${w}px` }} />
            ))}
          </colgroup>
          <TableHeader>
            <TableRow>
              {columns.map((col, colIdx) => (
                <TableHead
                  key={col.key}
                  sortable={col.sortable}
                  sortActive={
                    col.sortable && sortKey === (col.sortKey ?? col.key)
                  }
                  sortDir={sortOrder}
                  onSort={
                    col.sortable
                      ? () => toggleSort(col.sortKey ?? col.key)
                      : undefined
                  }
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
            {loading && instances.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="px-4 py-8 text-center text-control-placeholder"
                >
                  <div className="flex items-center justify-center gap-x-2">
                    <div className="animate-spin size-4 border-2 border-accent border-t-transparent rounded-full" />
                    {t("common.loading")}
                  </div>
                </TableCell>
              </TableRow>
            ) : instances.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="px-4 py-8 text-center text-control-placeholder"
                >
                  {t("common.no-data")}
                </TableCell>
              </TableRow>
            ) : (
              instances.map((instance) => (
                <TableRow
                  key={instance.name}
                  className="cursor-pointer"
                  onClick={(e) => handleRowClick(instance, e)}
                >
                  {columns.map((col) => (
                    <TableCell
                      key={col.key}
                      className={cn("overflow-hidden", col.cellClassName)}
                    >
                      {col.render(instance)}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
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
      {(() => {
        const canSync = hasWorkspacePermissionV2("bb.instances.sync");
        const canUpdate = hasWorkspacePermissionV2("bb.instances.update");
        const instanceActions: SelectionAction[] = [
          {
            key: "edit-env",
            label: t("database.edit-environment"),
            icon: SquareStack,
            onClick: () => setShowEditEnvDrawer(true),
            disabled: !canUpdate,
          },
          {
            key: "assign-license",
            label: t("subscription.instance-assignment.assign-license"),
            icon: GraduationCap,
            onClick: () =>
              handleAssignLicense(
                selectedInstanceList.map((instance) => instance.name)
              ),
            disabled: !canUpdate,
            hidden: !hasSplitInstanceLicense,
          },
        ];
        const allSelected =
          instances.length > 0 &&
          instances.every((i) => selectedNames.has(i.name));
        return (
          <SelectionActionBar
            count={selectedInstanceList.length}
            label={t("instance.selected-n-instances", {
              count: selectedInstanceList.length,
            })}
            allSelected={allSelected}
            onToggleSelectAll={() => {
              if (allSelected) setSelectedNames(new Set());
              else setSelectedNames(new Set(instances.map((i) => i.name)));
            }}
            actions={instanceActions}
          >
            <SyncDropdown disabled={!canSync || syncing} onSync={handleSync} />
          </SelectionActionBar>
        );
      })()}

      {/* Modals (portaled, position-independent) */}
      <EditEnvironmentSheet
        open={showEditEnvDrawer}
        onClose={() => setShowEditEnvDrawer(false)}
        onUpdate={handleEnvironmentUpdate}
      />
      {hasSplitInstanceLicense && (
        <InstanceAssignmentSheet
          open={showAssignLicenseSheet}
          selectedInstanceList={assignLicenseNames}
          onOpenChange={setShowAssignLicenseSheet}
          onUpdated={handleRowAction}
        />
      )}
    </div>
  );
}
