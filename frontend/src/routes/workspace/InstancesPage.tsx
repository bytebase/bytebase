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
import { router, useCurrentRoute } from "@/app/router";
import { INSTANCE_ROUTE_CREATE } from "@/app/router/handles";
import { useScrollRestorationLoadMore } from "@/app/router/NavigationScrollRestoration";
import {
  AdvancedSearch,
  getValueFromScopes,
  type ScopeOption,
  type SearchParams,
} from "@/components/AdvancedSearch";
import { EngineIcon } from "@/components/EngineIcon";
import { EnvironmentLabel } from "@/components/EnvironmentLabel";
import { InstanceAssignmentSheet } from "@/components/InstanceAssignmentSheet";
import { InstanceDeleteDialog } from "@/components/instance";
import { PermissionGuard } from "@/components/PermissionGuard";
import {
  type SelectionAction,
  SelectionActionBar,
} from "@/components/SelectionActionBar";
import { Alert } from "@/components/ui/alert";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { EllipsisText } from "@/components/ui/ellipsis-text";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  WorkspacePageContent,
  WorkspacePageFooter,
  WorkspacePageLayout,
  WorkspacePageToolbar,
} from "@/components/WorkspacePageLayout";
import { useColumnWidths } from "@/hooks/useColumnWidths";
import { PagedTableFooter } from "@/hooks/usePagedData";
import {
  getPageSizeOptions,
  useSessionPageSize,
} from "@/hooks/useSessionPageSize";
import {
  createAdvancedSearchParser,
  serializeAdvancedSearch,
  useURLSearchParam,
} from "@/hooks/useURLSearchParam";
import {
  CREATE_INSTANCE_PRODUCT_INTRO,
  useProductIntro,
} from "@/lib/productIntro";
import { cn } from "@/lib/utils";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import type { InstanceFilter } from "@/stores/app/types";
import { environmentNamePrefix } from "@/stores/modules/v1/common";
import {
  DEFAULT_ENVIRONMENT_COLOR,
  isValidInstanceName,
  UNKNOWN_ENVIRONMENT_NAME,
  unknownEnvironment,
} from "@/types";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { UpdateInstanceRequestSchema } from "@/types/proto-es/v1/instance_service_pb";
import {
  engineNameV1,
  hasWorkspacePermissionV2,
  hostPortOfDataSource,
  hostPortOfInstanceV1,
  supportedEngineV1List,
} from "@/utils";
import {
  defaultActiveStateSearchParams,
  getResourceStateFilter,
} from "./resourceStateFilter";

const ASSIGN_LICENSE_QUERY = "assignLicense";
const ASSIGN_LICENSE_INSTANCES_QUERY = "instances";
const parseInstanceSearch = createAdvancedSearchParser([
  "environment",
  "engine",
  "label",
  "state",
  "host",
  "port",
]);

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
  onCellClick?: (instance: Instance, e: React.MouseEvent) => void;
  onHeaderClick?: (e: React.MouseEvent) => void;
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
  const okClassName =
    variant === "warning"
      ? "bg-warning text-accent-text hover:bg-warning-hover"
      : undefined;

  return (
    <AlertDialog open onOpenChange={(nextOpen) => !nextOpen && onCancel()}>
      <AlertDialogContent className={cn("max-w-lg border-t-4", borderColor)}>
        <AlertDialogTitle>{title}</AlertDialogTitle>
        <AlertDialogDescription className="mt-2">
          {description}
        </AlertDialogDescription>
        {children && <div className="mt-4">{children}</div>}
        <div className="mt-6 flex justify-end gap-x-2">
          <Button appearance="outline" onClick={onCancel}>
            {t("common.cancel")}
          </Button>
          <Button
            variant={variant === "error" ? "destructive" : "default"}
            className={okClassName}
            onClick={onOk}
          >
            {okText}
          </Button>
        </div>
      </AlertDialogContent>
    </AlertDialog>
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
  const [showArchiveConfirm, setShowArchiveConfirm] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [forceArchive, setForceArchive] = useState(false);

  const canArchive = hasWorkspacePermissionV2("bb.instances.delete");
  const canRestore = hasWorkspacePermissionV2("bb.instances.undelete");

  const handleArchive = useCallback(async () => {
    try {
      await useAppStore.getState().archiveInstance(instance, forceArchive);
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
  }, [instance, forceArchive, t, onAction]);

  const handleRestore = useCallback(async () => {
    try {
      await useAppStore.getState().restoreInstance(instance);
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
  }, [instance, t, onAction]);

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
          {canArchive && (
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

      <InstanceDeleteDialog
        open={showDeleteConfirm}
        instance={instance}
        onOpenChange={setShowDeleteConfirm}
        onDeleted={onAction}
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
  const environments = useAppStore((s) => s.environmentList ?? []);
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
          <RadioGroup
            className="flex-col items-stretch gap-y-2"
            value={selected}
            onValueChange={(value) => setSelected(value as string)}
          >
            {environments.map((env) => (
              <RadioGroupItem
                key={env.name}
                value={env.name}
                className={cn(
                  "flex items-center gap-x-3 px-3 py-2 rounded-xs cursor-pointer border",
                  selected === env.name
                    ? "border-accent bg-accent/5"
                    : "border-transparent hover:bg-control-bg"
                )}
              >
                <div className="flex items-center gap-x-2">
                  <span
                    className="inline-block size-3 rounded-full"
                    style={{
                      backgroundColor: env.color || DEFAULT_ENVIRONMENT_COLOR,
                    }}
                  />
                  <span>{env.title}</span>
                </div>
              </RadioGroupItem>
            ))}
          </RadioGroup>
        </SheetBody>
        <SheetFooter>
          <Button appearance="secondary" onClick={onClose}>
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
// InstancesPage (main)
// ============================================================

export function InstancesPage() {
  const { t } = useTranslation();
  const route = useCurrentRoute();

  const defaultSearchParams = useMemo<SearchParams>(
    () => defaultActiveStateSearchParams(route.query.state),
    [route.query.state]
  );
  const [searchParams, setSearchParams] = useURLSearchParam<SearchParams>({
    param: "q",
    parse: parseInstanceSearch,
    serialize: serializeAdvancedSearch,
    defaultValue: defaultSearchParams,
  });

  // Scope options
  const canUndelete = hasWorkspacePermissionV2("bb.instances.undelete");
  const environments = useAppStore((s) => s.environmentList ?? []);
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
          {
            value: "ACTIVE",
            keywords: ["active"],
            custom: true,
            render: () => <span>{t("common.active")}</span>,
          },
          ...(canUndelete
            ? [
                {
                  value: "DELETED",
                  keywords: ["archived", "deleted"],
                  custom: true,
                  render: () => <span>{t("common.archived")}</span>,
                },
                {
                  value: "ALL",
                  keywords: ["all"],
                  custom: true,
                  render: () => <span>{t("common.all")}</span>,
                },
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
  const selectedState = getResourceStateFilter(stateFilterVal);

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
  useEffect(() => {
    const store = useAppStore.getState();
    if (!store.getIntroStateByKey("instance.visit")) {
      store.saveIntroStateByKey({
        key: "instance.visit",
        newState: true,
      });
    }
  }, []);

  // Instance count warning
  const instanceCountLimit = useAppStore((s) => s.instanceCountLimit());
  const totalInstanceCount = useAppStore((s) => s.totalInstanceCount());
  const hasSplitInstanceLicense = useAppStore((s) =>
    s.hasSplitInstanceLicense()
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
        const result = await useAppStore.getState().fetchInstanceList({
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
    [pageSize, filter, orderBy]
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
  useScrollRestorationLoadMore({
    dataList: instances,
    hasMore,
    isFetchingMore,
    isLoading: loading,
    loadMore,
  });

  // Selection state
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set());

  const selectedInstanceList = useMemo(() => {
    if (selectedNames.size === 0) return [];
    return Array.from(selectedNames)
      .filter((name) => isValidInstanceName(name))
      .map((name) => useAppStore.getState().getInstanceByName(name));
  }, [selectedNames]);

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
        await useAppStore
          .getState()
          .batchSyncInstances(Array.from(selectedNames), enableFullSync);
        pushNotification({
          module: "bytebase",
          style: "INFO",
          title: t("db.start-to-sync-schema"),
        });
      } finally {
        setSyncing(false);
      }
    },
    [selectedNames, t]
  );

  const handleEnvironmentUpdate = useCallback(
    async (environment: string) => {
      try {
        await useAppStore.getState().batchUpdateInstances(
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
    [selectedInstanceList, t, fetchInstances]
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

  // Depend on the two primitive params, not `route.query` — the route object
  // is rebuilt every render, which would re-run this effect each time.
  const assignLicenseFlag = route.query[ASSIGN_LICENSE_QUERY];
  const rawAssignLicenseInstances = route.query[ASSIGN_LICENSE_INSTANCES_QUERY];
  useEffect(() => {
    if (assignLicenseFlag !== "1") {
      return;
    }

    const names =
      typeof rawAssignLicenseInstances === "string"
        ? rawAssignLicenseInstances.split(",").filter(Boolean)
        : [];
    handleAssignLicense(names);

    const nextQuery = { ...router.currentRoute.value.query };
    delete nextQuery[ASSIGN_LICENSE_QUERY];
    delete nextQuery[ASSIGN_LICENSE_INSTANCES_QUERY];
    router.replace({ query: nextQuery });
  }, [assignLicenseFlag, rawAssignLicenseInstances, handleAssignLicense]);

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

  const activatedInstanceCount = useAppStore((s) => s.activatedInstanceCount());
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
  useProductIntro({
    id: CREATE_INSTANCE_PRODUCT_INTRO,
    title: t("workspace-setup-guide.intro.instance-title"),
    description: t("workspace-setup-guide.intro.instance-description"),
    disabled: !canCreate,
  });
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
          onClick={(e) => e.stopPropagation()}
        />
      ),
      defaultWidth: 48,
      cellClassName: "px-4 py-2",
      onCellClick: (instance, e) => {
        e.stopPropagation();
        toggleSelection(instance.name);
      },
      onHeaderClick: (e) => {
        e.stopPropagation();
        toggleSelectAll();
      },
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
        <EnvironmentLabel environmentName={instance.environment ?? ""} />
      ),
    },
    {
      key: "state",
      title: t("common.state"),
      defaultWidth: 120,
      minWidth: 100,
      resizable: true,
      render: (instance) =>
        instance.state === State.DELETED ? (
          <Badge variant="warning" className="text-xs">
            {t("common.archived")}
          </Badge>
        ) : (
          <Badge variant="success" className="text-xs">
            {t("common.active")}
          </Badge>
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
              <Button
                type="button"
                appearance="secondary"
                size="xs"
                className="size-5 shrink-0 p-0"
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
              </Button>
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
    <WorkspacePageLayout padding="flush">
      {/* Instance count warning */}
      {quotaExhausted && (
        <Alert
          variant="warning"
          className="mb-4"
          title={t("subscription.usage.instance-count.title")}
          description={t("subscription.usage.instance-count.runoutof", {
            total: instanceCountLimit,
          })}
        />
      )}

      {/* Header: Search + Create */}
      <WorkspacePageToolbar className="px-4">
        <AdvancedSearch
          params={searchParams}
          scopeOptions={scopeOptions}
          placeholder={t("instance.filter-instance-name")}
          onParamsChange={setSearchParams}
        />
        <PermissionGuard permissions={["bb.instances.create"]}>
          <Button
            data-product-intro-target={CREATE_INSTANCE_PRODUCT_INTRO}
            disabled={!canCreate}
            onClick={navigateToCreate}
          >
            <Plus className="h-4 w-4 mr-1" />
            {t("instance.connect-instance")}
          </Button>
        </PermissionGuard>
      </WorkspacePageToolbar>

      <WorkspacePageContent className="overflow-x-auto border rounded-sm">
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
                  className={cn(col.onHeaderClick && "cursor-pointer")}
                  onClick={col.onHeaderClick}
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
                      className={cn(
                        "overflow-hidden",
                        col.cellClassName,
                        col.onCellClick && "cursor-pointer"
                      )}
                      onClick={
                        col.onCellClick
                          ? (e) => col.onCellClick!(instance, e)
                          : undefined
                      }
                    >
                      {col.render(instance)}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </WorkspacePageContent>

      <WorkspacePageFooter>
        <PagedTableFooter
          pageSize={pageSize}
          pageSizeOptions={pageSizeOptions}
          onPageSizeChange={setPageSize}
          hasMore={hasMore}
          isFetchingMore={isFetchingMore}
          onLoadMore={loadMore}
        />
      </WorkspacePageFooter>

      {/* Batch operations bar is fixed within the visible main content. */}
      {(() => {
        const canSync = hasWorkspacePermissionV2("bb.instances.sync");
        const canUpdate = hasWorkspacePermissionV2("bb.instances.update");
        const syncDisabled = !canSync || syncing;
        const instanceActions: SelectionAction[] = [
          {
            key: "sync-new",
            label: syncing
              ? t("instance.syncing")
              : t("instance.sync.sync-new"),
            icon: RefreshCw,
            onClick: () => handleSync(false),
            disabled: syncDisabled,
          },
          {
            key: "sync-all",
            label: t("instance.sync.sync-all"),
            icon: RefreshCw,
            onClick: () => handleSync(true),
            disabled: syncDisabled,
          },
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
            label={t("common.n-selected", { n: selectedInstanceList.length })}
            allSelected={allSelected}
            onToggleSelectAll={() => {
              if (allSelected) setSelectedNames(new Set());
              else setSelectedNames(new Set(instances.map((i) => i.name)));
            }}
            actions={instanceActions}
          />
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
    </WorkspacePageLayout>
  );
}
