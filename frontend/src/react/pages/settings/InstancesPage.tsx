import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { sortBy, uniq } from "lodash-es";
import {
  ChevronDown,
  ChevronUp,
  EllipsisVertical,
  GraduationCap,
  Plus,
  RefreshCw,
  SquareStack,
} from "lucide-react";
import type { RefObject } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EngineIconPath } from "@/components/InstanceForm/constants";
import {
  AdvancedSearch,
  getValueFromScopes,
  type ScopeOption,
  type SearchParams,
} from "@/react/components/AdvancedSearch";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { type ColumnDef, useColumnWidths } from "@/react/hooks/useColumnWidths";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { INSTANCE_ROUTE_CREATE } from "@/router/dashboard/instance";
import {
  featureToRef,
  pushNotification,
  useActuatorV1Store,
  useCurrentUserV1,
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
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import {
  engineNameV1,
  getDefaultPagination,
  hasWorkspacePermissionV2,
  hexToRgb,
  hostPortOfDataSource,
  hostPortOfInstanceV1,
  supportedEngineV1List,
} from "@/utils";

// ============================================================
// Pagination helpers
// ============================================================

const PAGE_SIZE_OPTIONS = [50, 100, 200, 500];

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
// Shared hooks
// ============================================================

function useClickOutside(
  ref: RefObject<HTMLElement | null>,
  active: boolean,
  onClose: () => void
) {
  useEffect(() => {
    if (!active) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [ref, active, onClose]);
}

function useEscapeKey(active: boolean, onClose: () => void) {
  useEffect(() => {
    if (!active) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [active, onClose]);
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
  useEscapeKey(open, onCancel);

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
          "relative bg-white rounded-sm shadow-lg max-w-lg w-full mx-4 border-t-4",
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
              "inline-flex items-center justify-center rounded-xs px-4 py-2 text-sm font-medium",
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
  const [open, setOpen] = useState(false);
  const [showArchiveConfirm, setShowArchiveConfirm] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [forceArchive, setForceArchive] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const canArchive = hasWorkspacePermissionV2("bb.instances.delete");
  const canRestore = hasWorkspacePermissionV2("bb.instances.undelete");
  const closeDropdown = useCallback(() => setOpen(false), []);
  useClickOutside(dropdownRef, open, closeDropdown);

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
    setOpen(false);
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
          <div className="absolute right-0 top-full mt-1 bg-white border border-control-border rounded-sm shadow-lg z-10 min-w-[120px]">
            {isActive && canArchive && (
              <button
                className="w-full text-left px-3 py-2 text-sm hover:bg-control-bg flex items-center gap-x-2"
                onClick={(e) => {
                  e.stopPropagation();
                  setOpen(false);
                  setShowArchiveConfirm(true);
                }}
              >
                {t("common.archive")}
              </button>
            )}
            {!isActive && canRestore && (
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
            {(canArchive || canRestore) && (
              <button
                className="w-full text-left px-3 py-2 text-sm hover:bg-control-bg flex items-center gap-x-2 text-error"
                onClick={(e) => {
                  e.stopPropagation();
                  setOpen(false);
                  setShowDeleteConfirm(true);
                }}
              >
                {t("common.delete")}
              </button>
            )}
          </div>
        )}
      </div>

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
          <input
            type="checkbox"
            checked={forceArchive}
            onChange={(e) => setForceArchive(e.target.checked)}
            className="rounded-xs border-control-border"
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
        <span key={key} className="rounded-xs bg-gray-100 py-0.5 px-2 text-sm">
          {key}:{value}
        </span>
      ))}
      {hasMore && <span>...</span>}
    </div>
  );
}

// ============================================================
// EditEnvironmentDrawer
// ============================================================

function EditEnvironmentDrawer({
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
  useEscapeKey(open, onClose);
  useEffect(() => {
    if (open) setSelected("");
  }, [open]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="ml-auto relative bg-white w-[24rem] max-w-[100vw] h-full shadow-lg flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-control-border">
          <h2 className="text-lg font-semibold">
            {t("database.edit-environment")}
          </h2>
          <button className="p-1 hover:bg-control-bg rounded" onClick={onClose}>
            &times;
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-6">
          <div className="flex flex-col gap-y-2">
            {environments.map((env) => (
              <label
                key={env.name}
                className={cn(
                  "flex items-center gap-x-3 px-3 py-2 rounded-xs cursor-pointer border",
                  selected === env.name
                    ? "border-accent bg-accent/5"
                    : "border-transparent hover:bg-gray-50"
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
                      className="inline-block w-3 h-3 rounded-full"
                      style={{ backgroundColor: env.color }}
                    />
                  )}
                  <span>{env.title}</span>
                </div>
              </label>
            ))}
          </div>
        </div>
        <div className="flex justify-end items-center gap-x-2 px-6 py-4 border-t border-control-border">
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
            {t("common.confirm")}
          </Button>
        </div>
      </div>
    </div>
  );
}

// ============================================================
// BatchOperationsBar
// ============================================================

function SyncDropdown({
  disabled,
  onSync,
}: {
  disabled: boolean;
  onSync: (enableFullSync: boolean) => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const closeDropdown = useCallback(() => setOpen(false), []);
  useClickOutside(dropdownRef, open, closeDropdown);

  return (
    <div ref={dropdownRef} className="relative">
      <Button
        variant="ghost"
        size="sm"
        disabled={disabled}
        onClick={() => setOpen(!open)}
      >
        <RefreshCw className={cn("h-4 w-4 mr-1", disabled && "animate-spin")} />
        {disabled ? t("instance.syncing") : t("instance.sync.self")}
        <ChevronDown className="h-3 w-3 ml-1" />
      </Button>
      {open && (
        <div className="absolute left-0 top-full mt-1 bg-white border border-control-border rounded-sm shadow-lg z-10 min-w-[200px]">
          <button
            className="w-full text-left px-3 py-2 text-sm hover:bg-control-bg"
            title={t("instance.sync.sync-all-tip")}
            onClick={() => {
              setOpen(false);
              onSync(true);
            }}
          >
            {t("instance.sync.sync-all")}
          </button>
          <button
            className="w-full text-left px-3 py-2 text-sm hover:bg-control-bg"
            onClick={() => {
              setOpen(false);
              onSync(false);
            }}
          >
            {t("instance.sync.sync-new")}
          </button>
        </div>
      )}
    </div>
  );
}

function BatchOperationsBar({
  selectedInstances,
  syncing,
  onSync,
  onEditEnvironment,
  showAssignLicense,
}: {
  selectedInstances: Instance[];
  syncing: boolean;
  onSync: (enableFullSync: boolean) => void;
  onEditEnvironment: () => void;
  showAssignLicense?: boolean;
}) {
  const { t } = useTranslation();
  const canSync = hasWorkspacePermissionV2("bb.instances.sync");
  const canUpdate = hasWorkspacePermissionV2("bb.instances.update");

  if (selectedInstances.length === 0) return null;

  return (
    <div className="relative z-10 text-sm flex flex-col lg:flex-row items-start lg:items-center bg-blue-100 py-3 px-4 text-main gap-y-2 gap-x-4 overflow-visible">
      <span className="whitespace-nowrap">
        {t("instance.selected-n-instances", {
          count: selectedInstances.length,
        })}
      </span>
      <div className="flex items-center gap-x-2">
        <SyncDropdown disabled={!canSync || syncing} onSync={onSync} />
        <Button
          variant="ghost"
          size="sm"
          disabled={!canUpdate}
          onClick={onEditEnvironment}
        >
          <SquareStack className="h-4 w-4 mr-1" />
          {t("database.edit-environment")}
        </Button>
        {showAssignLicense && (
          <Button variant="ghost" size="sm" disabled={!canUpdate}>
            <GraduationCap className="h-4 w-4 mr-1" />
            {t("subscription.instance-assignment.assign-license")}
          </Button>
        )}
      </div>
    </div>
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
        description: t("common.environment"),
        options: [unknownEnvironment(), ...environments].map((env) => {
          const isUnknown = env.name === UNKNOWN_ENVIRONMENT_NAME;
          return {
            value: env.id,
            keywords: isUnknown
              ? ["unassigned", "none", env.id]
              : [env.id, env.title],
            render: isUnknown
              ? () => (
                  <span className="italic text-control-light">Unassigned</span>
                )
              : undefined,
            custom: isUnknown,
          };
        }),
      },
      {
        id: "engine",
        title: t("database.engine"),
        description: t("database.engine"),
        options: supportedEngineV1List().map((engine) => ({
          value: Engine[engine],
          keywords: [Engine[engine].toLowerCase(), engineNameV1(engine)],
        })),
        allowMultiple: true,
      },
      {
        id: "label",
        title: t("common.labels"),
        description: t("common.labels"),
        allowMultiple: true,
      },
      {
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
    const currentQuery = router.currentRoute.value.query.q as string;
    if (queryString !== (currentQuery ?? "")) {
      router.replace({ query: queryString ? { q: queryString } : {} });
    }
  }, [searchParams]);

  // Instance count warning
  const instanceCountLimit = useVueState(
    () => subscriptionStore.instanceCountLimit
  );
  const totalInstanceCount = useVueState(
    () => actuatorStore.totalInstanceCount
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
          title: t("database.edit-environment"),
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

  const canCreate = hasWorkspacePermissionV2("bb.instances.create");
  const allSelected =
    instances.length > 0 && selectedNames.size === instances.length;
  const someSelected =
    selectedNames.size > 0 && selectedNames.size < instances.length;
  const headerCheckboxRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (headerCheckboxRef.current) {
      headerCheckboxRef.current.indeterminate = someSelected;
    }
  }, [someSelected]);
  const pageSizeOptions = getPageSizeOptions();

  const columns: ColumnDef[] = useMemo(
    () => [
      { key: "checkbox", defaultWidth: 48, minWidth: 48, resizable: false },
      { key: "name", defaultWidth: 240, minWidth: 160 },
      { key: "environment", defaultWidth: 200, minWidth: 120 },
      { key: "address", defaultWidth: 200, minWidth: 120 },
      { key: "labels", defaultWidth: 260, minWidth: 120 },
      { key: "license", defaultWidth: 100, minWidth: 80 },
      { key: "actions", defaultWidth: 50, minWidth: 50, resizable: false },
    ],
    []
  );

  const { widths, totalWidth, onResizeStart } = useColumnWidths(
    columns,
    "bb.instances-table-widths"
  );

  return (
    <div className="py-4 flex flex-col">
      {/* Instance count warning */}
      {quotaExhausted && (
        <div className="mx-4 mb-2 p-3 rounded-xs border border-warning bg-warning/5">
          <p className="text-sm font-medium text-warning">
            {t("subscription.usage.instance-count.title")}
          </p>
          <p className="text-sm text-warning/80 mt-1">
            {t("subscription.usage.instance-count.runoutof", {
              total: instanceCountLimit,
            })}
          </p>
        </div>
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
            {t("quick-action.add-instance")}
          </Button>
        </PermissionGuard>
      </div>

      {/* Batch operations */}
      <BatchOperationsBar
        selectedInstances={selectedInstanceList}
        syncing={syncing}
        onSync={handleSync}
        onEditEnvironment={() => setShowEditEnvDrawer(true)}
        showAssignLicense={subscriptionStore.currentPlan !== PlanType.FREE}
      />

      <EditEnvironmentDrawer
        open={showEditEnvDrawer}
        onClose={() => setShowEditEnvDrawer(false)}
        onUpdate={handleEnvironmentUpdate}
      />

      {/* Table */}
      <div className="flex flex-col gap-y-4">
        <div className="">
          <Table style={{ minWidth: `${totalWidth}px` }}>
            <colgroup>
              {widths.map((w, i) => (
                <col key={columns[i].key} style={{ width: w + "px" }} />
              ))}
            </colgroup>
            <TableHeader>
              <TableRow className="bg-gray-50 border-b border-control-border">
                <TableHead>
                  <input
                    ref={headerCheckboxRef}
                    type="checkbox"
                    checked={allSelected}
                    onChange={toggleSelectAll}
                    className="rounded-xs border-control-border"
                  />
                </TableHead>
                <TableHead
                  className="cursor-pointer select-none"
                  onClick={() => toggleSort("title")}
                  resizable
                  onResizeStart={(e) => onResizeStart(1, e)}
                >
                  <div className="flex items-center gap-x-1">
                    {t("common.name")}
                    {renderSortIndicator("title")}
                  </div>
                </TableHead>
                <TableHead
                  className="cursor-pointer select-none"
                  onClick={() => toggleSort("environment")}
                  resizable
                  onResizeStart={(e) => onResizeStart(2, e)}
                >
                  <div className="flex items-center gap-x-1">
                    {t("common.environment")}
                    {renderSortIndicator("environment")}
                  </div>
                </TableHead>
                <TableHead resizable onResizeStart={(e) => onResizeStart(3, e)}>
                  {t("common.address")}
                </TableHead>
                <TableHead
                  className="hidden md:table-cell"
                  resizable
                  onResizeStart={(e) => onResizeStart(4, e)}
                >
                  {t("common.labels")}
                </TableHead>
                <TableHead resizable onResizeStart={(e) => onResizeStart(5, e)}>
                  {t("subscription.instance-assignment.license")}
                </TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading && instances.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={7}
                    className="py-8 text-center text-control-placeholder"
                  >
                    <div className="flex items-center justify-center gap-x-2">
                      <div className="animate-spin h-4 w-4 border-2 border-accent border-t-transparent rounded-full" />
                      {t("common.loading")}
                    </div>
                  </TableCell>
                </TableRow>
              ) : instances.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={7}
                    className="py-8 text-center text-control-placeholder"
                  >
                    {t("common.no-data")}
                  </TableCell>
                </TableRow>
              ) : (
                instances.map((instance, i) => {
                  const isSelected = selectedNames.has(instance.name);
                  const isExpanded = expandedDataSources.has(instance.name);
                  const hasMultipleDS = instance.dataSources.length > 1;

                  return (
                    <TableRow
                      key={instance.name}
                      className={cn(
                        "cursor-pointer hover:bg-gray-50",
                        i % 2 === 1 && "bg-gray-50/50"
                      )}
                      onClick={(e) => handleRowClick(instance, e)}
                    >
                      <TableCell>
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => toggleSelection(instance.name)}
                          onClick={(e) => e.stopPropagation()}
                          className="rounded-xs border-control-border"
                        />
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-x-2">
                          <img
                            className="h-5 w-5"
                            src={EngineIconPath[instance.engine]}
                            alt=""
                          />
                          <span className="truncate">{instance.title}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <EnvironmentName
                          environmentName={instance.environment ?? ""}
                        />
                      </TableCell>
                      <TableCell>
                        <div className="flex items-start gap-x-2">
                          <span className="truncate">
                            {isExpanded
                              ? instance.dataSources.map((ds, idx) => (
                                  <div key={idx}>
                                    {hostPortOfDataSource(ds)}
                                  </div>
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
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <LabelsDisplay labels={instance.labels} />
                      </TableCell>
                      <TableCell>{instance.activation ? "Y" : ""}</TableCell>
                      <TableCell>
                        <div
                          className="flex justify-end"
                          onClick={(e) => e.stopPropagation()}
                        >
                          <InstanceActionDropdown
                            instance={instance}
                            onAction={handleRowAction}
                          />
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </div>

        {/* Pagination footer */}
        <div className="flex items-center justify-end gap-x-2 mx-4">
          <div className="flex items-center gap-x-2">
            <span className="text-sm text-control-light">
              {t("common.rows-per-page")}
            </span>
            <select
              className="border border-control-border rounded-sm text-sm pl-2 pr-6 py-1 min-w-[5rem]"
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
    </div>
  );
}
