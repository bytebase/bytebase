import { file_google_rpc_error_details } from "@buf/googleapis_googleapis.bufbuild_es/google/rpc/error_details_pb";
import { create, createRegistry, toJsonString } from "@bufbuild/protobuf";
import { AnySchema } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import {
  ArrowRight,
  Calendar,
  ChevronDown,
  Download,
  ExternalLink,
  Maximize2,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { auditLogServiceClientConnect } from "@/connect";
import { ALL_METHODS_WITH_AUDIT } from "@/connect/methods";
import {
  AdvancedSearch,
  type ScopeOption,
  type SearchParams,
} from "@/react/components/AdvancedSearch";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { Button } from "@/react/components/ui/button";
import {
  getPageSizeOptions,
  useSessionPageSize,
} from "@/react/hooks/useSessionPageSize";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { pushNotification, useSubscriptionV1Store } from "@/store";
import {
  extractUserEmail,
  getProjectIdPlanUidStageUidFromRolloutName,
  planNamePrefix,
  projectNamePrefix,
  userNamePrefix,
} from "@/store/modules/v1/common";
import { getDateForPbTimestampProtoEs } from "@/types";
import { StatusSchema } from "@/types/proto-es/google/rpc/status_pb";
import type { AuditLog } from "@/types/proto-es/v1/audit_log_service_pb";
import {
  AuditDataSchema,
  AuditLog_Severity,
  ExportAuditLogsRequestSchema,
  SearchAuditLogsRequestSchema,
} from "@/types/proto-es/v1/audit_log_service_pb";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { IssueService } from "@/types/proto-es/v1/issue_service_pb";
import {
  file_v1_plan_service,
  PlanService,
} from "@/types/proto-es/v1/plan_service_pb";
import { RolloutService } from "@/types/proto-es/v1/rollout_service_pb";
import { SettingSchema } from "@/types/proto-es/v1/setting_service_pb";
import { SQLService } from "@/types/proto-es/v1/sql_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { formatAbsoluteDateTime, humanizeDurationV1 } from "@/utils";

dayjs.extend(utc);

const registry = createRegistry(
  file_google_rpc_error_details,
  file_v1_plan_service,
  AuditDataSchema,
  SettingSchema
);

// ============================================================
// Filter helpers
// ============================================================

interface AuditLogFilter {
  method?: string;
  level?: AuditLog_Severity;
  userEmail?: string;
  createdTsAfter?: number;
  createdTsBefore?: number;
}

function buildFilterString(filter: AuditLogFilter): string {
  const parts: string[] = [];
  if (filter.method) parts.push(`method == "${filter.method}"`);
  if (filter.level !== undefined)
    parts.push(`severity == "${AuditLog_Severity[filter.level]}"`);
  if (filter.userEmail)
    parts.push(`user == "${userNamePrefix}${filter.userEmail}"`);
  if (filter.createdTsAfter)
    parts.push(
      `create_time >= "${dayjs(filter.createdTsAfter).utc().format()}"`
    );
  if (filter.createdTsBefore)
    parts.push(
      `create_time <= "${dayjs(filter.createdTsBefore).utc().format()}"`
    );
  return parts.join(" && ");
}

function buildAuditLogFilter(params: SearchParams): AuditLogFilter {
  const filter: AuditLogFilter = {};
  const method = params.scopes.find((s) => s.id === "method")?.value;
  if (method) filter.method = method;
  const actor = params.scopes.find((s) => s.id === "actor")?.value;
  if (actor) filter.userEmail = actor;
  const level = params.scopes.find((s) => s.id === "level")?.value;
  if (level)
    filter.level = AuditLog_Severity[level as keyof typeof AuditLog_Severity];
  const created = params.scopes.find((s) => s.id === "created")?.value;
  if (created) {
    const parts = created.split(",");
    if (parts.length === 2) {
      filter.createdTsAfter = parseInt(parts[0], 10);
      filter.createdTsBefore = parseInt(parts[1], 10);
    }
  }
  return filter;
}

// ============================================================
// View link helper
// ============================================================

function getViewLink(auditLog: AuditLog): string | null {
  let parsedRequest: Record<string, unknown>;
  let parsedResponse: Record<string, unknown>;
  try {
    parsedRequest = JSON.parse(auditLog.request || "{}") as Record<
      string,
      unknown
    >;
    parsedResponse = JSON.parse(auditLog.response || "{}") as Record<
      string,
      unknown
    >;
  } catch {
    return null;
  }
  if (Boolean(parsedRequest["validateOnly"])) return null;
  const sections = auditLog.method.split("/").filter((i) => i);
  switch (sections[0]) {
    case RolloutService.typeName:
    case PlanService.typeName:
    case IssueService.typeName:
      return (parsedResponse["name"] as string) || null;
    case SQLService.typeName: {
      if (sections[1] !== "Export") return null;
      const name = parsedRequest["name"] as string | undefined;
      if (!name) return null;
      const [projectId, planId] =
        getProjectIdPlanUidStageUidFromRolloutName(name);
      if (!projectId || !planId) return null;
      return `${projectNamePrefix}${projectId}/${planNamePrefix}${planId}/rollout`;
    }
  }
  return null;
}

// ============================================================
// JSONStringView
// ============================================================

function JSONStringView({ jsonString }: { jsonString: string }) {
  const { t } = useTranslation();
  const [showModal, setShowModal] = useState(false);

  const formatted = useMemo(() => {
    try {
      return JSON.stringify(JSON.parse(jsonString), null, 2);
    } catch {
      return "-";
    }
  }, [jsonString]);

  useEffect(() => {
    if (!showModal) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") setShowModal(false);
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [showModal]);

  return (
    <>
      <div className="group grow-0 w-full flex flex-row justify-start items-center gap-2">
        <p className="line-clamp-2">
          <code className="text-sm break-all">{jsonString}</code>
        </p>
        <div className="hidden group-hover:block shrink-0 h-[22px]">
          <button
            className="p-0.5 border border-control-border rounded-xs hover:bg-control-bg"
            onClick={() => setShowModal(true)}
          >
            <Maximize2 className="w-3 h-3" />
          </button>
        </div>
      </div>
      {showModal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
          onClick={() => setShowModal(false)}
        >
          <div
            className="bg-white rounded-sm shadow-lg flex flex-col"
            style={{
              width: "calc(100vw - 12rem)",
              height: "calc(100vh - 12rem)",
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between px-4 py-3 border-b">
              <h3 className="text-base font-medium">
                {t("common.view-details")}
              </h3>
              <button
                className="p-1 hover:bg-control-bg rounded-full"
                onClick={() => setShowModal(false)}
              >
                <X className="w-4 h-4" />
              </button>
            </div>
            <div className="flex-1 overflow-auto p-4">
              <pre className="text-sm font-mono whitespace-pre-wrap break-all">
                {formatted}
              </pre>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

// ============================================================
// TimeRangePicker
// ============================================================

function TimeRangePicker({
  params,
  onParamsChange,
}: {
  params: SearchParams;
  onParamsChange: (params: SearchParams) => void;
}) {
  const { t } = useTranslation();
  const [showPicker, setShowPicker] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setShowPicker(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const createdScope = params.scopes.find((s) => s.id === "created");
  const [fromTs, toTs] = useMemo(() => {
    if (!createdScope) return [undefined, undefined];
    const parts = createdScope.value.split(",");
    if (parts.length !== 2) return [undefined, undefined];
    return [parseInt(parts[0], 10), parseInt(parts[1], 10)];
  }, [createdScope]);

  const fromDatetime = fromTs
    ? dayjs(fromTs).format("YYYY-MM-DDTHH:mm:ss")
    : "";
  const toDatetime = toTs ? dayjs(toTs).format("YYYY-MM-DDTHH:mm:ss") : "";

  const displayFrom = fromTs ? dayjs(fromTs).format("YYYY-MM-DD HH:mm:ss") : "";
  const displayTo = toTs ? dayjs(toTs).format("YYYY-MM-DD HH:mm:ss") : "";

  const updateRange = useCallback(
    (from: string, to: string) => {
      const fromVal = from ? dayjs(from).valueOf() : undefined;
      const toVal = to ? dayjs(to).valueOf() : undefined;
      const scopes = params.scopes.filter((s) => s.id !== "created");
      if (fromVal !== undefined && toVal !== undefined) {
        scopes.push({
          id: "created",
          value: `${fromVal},${toVal}`,
          readonly: true,
        });
      }
      onParamsChange({ ...params, scopes });
    },
    [params, onParamsChange]
  );

  return (
    <div ref={containerRef} className="relative shrink-0">
      <button
        className="h-9 flex items-center gap-x-2 border border-control-border rounded-xs px-3 text-sm hover:bg-control-bg whitespace-nowrap"
        onClick={() => setShowPicker(!showPicker)}
      >
        {displayFrom && displayTo ? (
          <>
            <span>{displayFrom}</span>
            <ArrowRight className="w-3.5 h-3.5 text-control-light shrink-0" />
            <span>{displayTo}</span>
          </>
        ) : (
          <span className="text-control-placeholder">{t("common.select")}</span>
        )}
        <Calendar className="w-4 h-4 text-control-light ml-1 shrink-0" />
      </button>
      {showPicker && (
        <div className="absolute right-0 top-[42px] bg-white border border-control-border rounded-sm shadow-lg z-50 p-3 flex flex-col gap-y-2 min-w-[300px]">
          <div className="flex items-center gap-x-2">
            <label className="text-sm text-control-light whitespace-nowrap w-10">
              {t("common.from")}
            </label>
            <input
              type="datetime-local"
              step="1"
              className="flex-1 border border-control-border rounded-xs px-2 py-1 text-sm"
              value={fromDatetime}
              onChange={(e) => updateRange(e.target.value, toDatetime)}
            />
          </div>
          <div className="flex items-center gap-x-2">
            <label className="text-sm text-control-light whitespace-nowrap w-10">
              {t("common.to")}
            </label>
            <input
              type="datetime-local"
              step="1"
              className="flex-1 border border-control-border rounded-xs px-2 py-1 text-sm"
              value={toDatetime}
              onChange={(e) => updateRange(fromDatetime, e.target.value)}
            />
          </div>
        </div>
      )}
    </div>
  );
}

// ============================================================
// ExportDropdown
// ============================================================

function ExportDropdown({
  disabled,
  tooltip,
  onExport,
}: {
  disabled: boolean;
  tooltip: string;
  onExport: (format: ExportFormat) => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const formats = [
    { format: ExportFormat.CSV, label: "CSV" },
    { format: ExportFormat.JSON, label: "JSON" },
    { format: ExportFormat.XLSX, label: "XLSX" },
  ];

  return (
    <div ref={containerRef} className="relative shrink-0">
      <Button
        disabled={disabled}
        title={tooltip}
        onClick={() => setOpen(!open)}
      >
        <Download className="w-4 h-4 mr-1" />
        {t("common.export")}
      </Button>
      {open && !disabled && (
        <div className="absolute right-0 top-[42px] bg-white border border-control-border rounded-sm shadow-lg z-50 py-1 min-w-[100px]">
          {formats.map(({ format, label }) => (
            <button
              key={label}
              className="block w-full text-left px-3 py-1.5 text-sm hover:bg-control-bg"
              onClick={() => {
                onExport(format);
                setOpen(false);
              }}
            >
              {label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

// ============================================================
// Column definitions
// ============================================================

interface ColumnDef {
  key: string;
  title: string;
  defaultWidth: number;
  minWidth?: number;
  resizable?: boolean;
  sortable?: boolean;
  render: (auditLog: AuditLog) => React.ReactNode;
}

function useColumnDefs(): ColumnDef[] {
  const { t } = useTranslation();
  return useMemo(
    () => [
      {
        key: "create_time",
        title: t("audit-log.table.created-ts"),
        defaultWidth: 220,
        minWidth: 160,
        resizable: true,
        sortable: true,
        render: (log: AuditLog) =>
          formatAbsoluteDateTime(
            getDateForPbTimestampProtoEs(log.createTime)?.getTime() ?? 0
          ),
      },
      {
        key: "severity",
        title: t("audit-log.table.level"),
        defaultWidth: 60,
        minWidth: 50,
        resizable: true,
        render: (log: AuditLog) => AuditLog_Severity[log.severity],
      },
      {
        key: "method",
        title: t("audit-log.table.method"),
        defaultWidth: 280,
        minWidth: 150,
        resizable: true,
        render: (log: AuditLog) => log.method,
      },
      {
        key: "actor",
        title: t("audit-log.table.actor"),
        defaultWidth: 200,
        minWidth: 120,
        resizable: true,
        render: (log: AuditLog) => {
          if (!log.user) return <span>-</span>;
          const email = extractUserEmail(log.user);
          return (
            <a href={`mailto:${email}`} className="text-accent hover:underline">
              {email}
            </a>
          );
        },
      },
      {
        key: "request",
        title: t("audit-log.table.request"),
        defaultWidth: 300,
        minWidth: 180,
        resizable: true,
        render: (log: AuditLog) =>
          log.request.length > 0 ? (
            <JSONStringView jsonString={log.request} />
          ) : (
            "-"
          ),
      },
      {
        key: "response",
        title: t("audit-log.table.response"),
        defaultWidth: 300,
        minWidth: 180,
        resizable: true,
        render: (log: AuditLog) =>
          log.response.length > 0 ? (
            <JSONStringView jsonString={log.response} />
          ) : (
            "-"
          ),
      },
      {
        key: "status",
        title: t("audit-log.table.status"),
        defaultWidth: 80,
        minWidth: 60,
        resizable: true,
        render: (log: AuditLog) =>
          log.status ? (
            <JSONStringView
              jsonString={toJsonString(StatusSchema, log.status, {
                registry,
              })}
            />
          ) : (
            "-"
          ),
      },
      {
        key: "latency",
        title: t("audit-log.table.latency"),
        defaultWidth: 90,
        minWidth: 60,
        resizable: true,
        render: (log: AuditLog) => (
          <span className="whitespace-nowrap">
            {humanizeDurationV1(log.latency)}
          </span>
        ),
      },
      {
        key: "service-data",
        title: t("audit-log.table.service-data"),
        defaultWidth: 240,
        minWidth: 120,
        resizable: true,
        render: (log: AuditLog) =>
          log.serviceData ? (
            <JSONStringView
              jsonString={toJsonString(AnySchema, log.serviceData, {
                registry,
              })}
            />
          ) : (
            "-"
          ),
      },
      {
        key: "view",
        title: t("common.view"),
        defaultWidth: 50,
        render: (log: AuditLog) => {
          let link = getViewLink(log);
          if (!link) return null;
          if (!link.startsWith("/")) link = `/${link}`;
          return (
            <a href={link} target="_blank" rel="noreferrer">
              <ExternalLink className="w-4 h-4 text-accent" />
            </a>
          );
        },
      },
    ],
    [t]
  );
}

// ============================================================
// useColumnWidths
// ============================================================

function useColumnWidths(columns: ColumnDef[]) {
  const [widths, setWidths] = useState<number[]>(() =>
    columns.map((c) => c.defaultWidth)
  );
  const dragRef = useRef<{
    colIndex: number;
    startX: number;
    startWidth: number;
  } | null>(null);

  const onMouseDown = useCallback(
    (colIndex: number, e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      dragRef.current = {
        colIndex,
        startX: e.clientX,
        startWidth: widths[colIndex],
      };

      const onMouseMove = (ev: MouseEvent) => {
        if (!dragRef.current) return;
        const delta = ev.clientX - dragRef.current.startX;
        const min = columns[dragRef.current.colIndex].minWidth ?? 40;
        const newWidth = Math.max(min, dragRef.current.startWidth + delta);
        setWidths((prev) => {
          const next = [...prev];
          next[dragRef.current!.colIndex] = newWidth;
          return next;
        });
      };
      const onMouseUp = () => {
        dragRef.current = null;
        document.removeEventListener("mousemove", onMouseMove);
        document.removeEventListener("mouseup", onMouseUp);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
      };
      document.addEventListener("mousemove", onMouseMove);
      document.addEventListener("mouseup", onMouseUp);
      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
    },
    [widths, columns]
  );

  const totalWidth = widths.reduce((sum, w) => sum + w, 0);

  return { widths, totalWidth, onResizeStart: onMouseDown };
}

// ============================================================
// AuditLogTable — shared component
// ============================================================

export interface AuditLogTableProps {
  /** Resource parent for the API call (e.g. "projects/-" or "projects/my-project"). */
  parent: string;
  /** Whether the caller can export audit logs. */
  canExport: boolean;
  /** Additional readonly scopes injected into the search params (e.g. project scope). */
  readonlyScopes?: Array<{ id: string; value: string }>;
}

export function AuditLogTable({
  parent,
  canExport,
  readonlyScopes,
}: AuditLogTableProps) {
  const { t } = useTranslation();
  const subscriptionStore = useSubscriptionV1Store();
  const hasAuditLogFeature = useVueState(() =>
    subscriptionStore.hasFeature(PlanFeature.FEATURE_AUDIT_LOG)
  );
  const columns = useColumnDefs();
  const { widths, totalWidth, onResizeStart } = useColumnWidths(columns);

  const buildDefaultParams = useCallback((): SearchParams => {
    const to = dayjs().endOf("day");
    const from = to.add(-30, "day");
    return {
      query: "",
      scopes: [
        ...(readonlyScopes?.map((s) => ({ ...s, readonly: true })) ?? []),
        {
          id: "created",
          value: `${from.valueOf()},${to.valueOf()}`,
          readonly: true,
        },
      ],
    };
  }, [readonlyScopes]);

  const [searchParams, setSearchParams] =
    useState<SearchParams>(buildDefaultParams);

  // Reset search params when readonlyScopes change (e.g. navigating between projects).
  const prevReadonlyScopesRef = useRef(readonlyScopes);
  useEffect(() => {
    if (prevReadonlyScopesRef.current !== readonlyScopes) {
      prevReadonlyScopesRef.current = readonlyScopes;
      setSearchParams(buildDefaultParams());
    }
  }, [readonlyScopes, buildDefaultParams]);

  const filter = useMemo(
    () => buildAuditLogFilter(searchParams),
    [searchParams]
  );

  // Data fetching
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const nextPageTokenRef = useRef("");
  const [hasMore, setHasMore] = useState(false);
  const [isFetchingMore, setIsFetchingMore] = useState(false);
  const [pageSize, setPageSize] = useSessionPageSize("bb.audit-log-table");
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

  const fetchAuditLogs = useCallback(
    async (isRefresh: boolean) => {
      if (!hasAuditLogFeature) return;
      const currentFetchId = ++fetchIdRef.current;
      if (isRefresh) setLoading(true);
      else setIsFetchingMore(true);
      try {
        const token = isRefresh ? "" : nextPageTokenRef.current;
        const request = create(SearchAuditLogsRequestSchema, {
          parent,
          filter: buildFilterString(filter),
          orderBy,
          pageSize,
          pageToken: token,
        });
        const resp =
          await auditLogServiceClientConnect.searchAuditLogs(request);
        if (currentFetchId !== fetchIdRef.current) return;
        if (isRefresh) setAuditLogs(resp.auditLogs);
        else setAuditLogs((prev) => [...prev, ...resp.auditLogs]);
        nextPageTokenRef.current = resp.nextPageToken ?? "";
        setHasMore(Boolean(resp.nextPageToken));
      } catch (e) {
        if (e instanceof Error && e.name === "AbortError") return;
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.failed"),
          description: (e as { message?: string }).message,
        });
      } finally {
        if (currentFetchId === fetchIdRef.current) {
          setLoading(false);
          setIsFetchingMore(false);
        }
      }
    },
    [hasAuditLogFeature, parent, filter, orderBy, pageSize, t]
  );

  const isFirstLoad = useRef(true);
  useEffect(() => {
    if (isFirstLoad.current) {
      isFirstLoad.current = false;
      fetchAuditLogs(true);
      return;
    }
    const timer = setTimeout(() => fetchAuditLogs(true), 300);
    return () => clearTimeout(timer);
  }, [fetchAuditLogs]);

  const loadMore = useCallback(() => {
    if (nextPageTokenRef.current && !isFetchingMore && !loading)
      fetchAuditLogs(false);
  }, [isFetchingMore, loading, fetchAuditLogs]);

  // Export
  const [exporting, setExporting] = useState(false);
  const handleExport = useCallback(
    async (format: ExportFormat) => {
      setExporting(true);
      try {
        let pageToken = "";
        const blobs: Uint8Array[] = [];
        do {
          const request = create(ExportAuditLogsRequestSchema, {
            parent,
            filter: buildFilterString(filter),
            orderBy,
            format,
            pageSize: 5000,
            pageToken,
          });
          const resp =
            await auditLogServiceClientConnect.exportAuditLogs(request);
          blobs.push(resp.content);
          pageToken = resp.nextPageToken;
        } while (pageToken);

        for (let j = 0; j < blobs.length; j++) {
          const blob = new Blob([new Uint8Array(blobs[j]).buffer]);
          const url = URL.createObjectURL(blob);
          const a = document.createElement("a");
          a.href = url;
          a.download = `audit-log.file${j + 1}.${dayjs().format("YYYY-MM-DDTHH-mm-ss")}.${ExportFormat[format].toLowerCase()}`;
          a.click();
          URL.revokeObjectURL(url);
        }
      } catch (e) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.failed"),
          description: (e as { message?: string }).message,
        });
      } finally {
        setExporting(false);
      }
    },
    [parent, filter, orderBy, t]
  );

  const disableExportTip = useMemo(() => {
    if (!filter.createdTsAfter || !filter.createdTsBefore)
      return t("audit-log.export-tooltip");
    if (
      filter.createdTsBefore - filter.createdTsAfter >
      30 * 24 * 60 * 60 * 1000
    )
      return t("audit-log.export-tooltip");
    return "";
  }, [filter, t]);

  const scopeOptions = useMemo((): ScopeOption[] => {
    return [
      {
        id: "actor",
        title: t("audit-log.advanced-search.scope.actor.title"),
        description: t("audit-log.advanced-search.scope.actor.description"),
      },
      {
        id: "method",
        title: t("audit-log.advanced-search.scope.method.title"),
        description: t("audit-log.advanced-search.scope.method.description"),
        options: ALL_METHODS_WITH_AUDIT.map((method) => ({
          value: method,
          keywords: [method],
        })),
      },
      {
        id: "level",
        title: t("audit-log.advanced-search.scope.level.title"),
        description: t("audit-log.advanced-search.scope.level.description"),
        options: Object.keys(AuditLog_Severity)
          .filter((v) => Number.isNaN(Number(v)))
          .map((severity) => ({
            value: severity,
            keywords: [severity],
          })),
      },
    ];
  }, [t]);

  const renderSortIndicator = (columnKey: string) => {
    if (sortKey !== columnKey)
      return <ChevronDown className="h-4 w-4 text-control-light" />;
    return (
      <ChevronDown
        className={cn(
          "h-4 w-4 text-accent transition-transform",
          sortOrder === "asc" && "rotate-180"
        )}
      />
    );
  };

  const pageSizeOptions = getPageSizeOptions();

  return (
    <div className="flex flex-col">
      {/* Header */}
      <div className="px-4 py-4 flex flex-col gap-y-2">
        <FeatureAttention feature={PlanFeature.FEATURE_AUDIT_LOG} />
        <div className="flex items-center gap-x-2">
          <AdvancedSearch
            params={searchParams}
            scopeOptions={scopeOptions}
            onParamsChange={setSearchParams}
          />
          <TimeRangePicker
            params={searchParams}
            onParamsChange={setSearchParams}
          />
          {canExport && (
            <ExportDropdown
              disabled={!hasAuditLogFeature || !!disableExportTip || exporting}
              tooltip={disableExportTip}
              onExport={handleExport}
            />
          )}
        </div>
      </div>

      {/* Table */}
      {hasAuditLogFeature ? (
        <div>
          <div className="overflow-x-auto">
            <table
              className="text-sm border-t border-block-border table-fixed"
              style={{ width: `${totalWidth}px` }}
            >
              <colgroup>
                {widths.map((w, i) => (
                  <col key={columns[i].key} style={{ width: `${w}px` }} />
                ))}
              </colgroup>
              <thead>
                <tr className="border-b border-block-border">
                  {columns.map((col, colIdx) => (
                    <th
                      key={col.key}
                      className={cn(
                        "relative px-4 py-3 text-left text-sm font-medium text-main whitespace-nowrap",
                        col.sortable && "cursor-pointer select-none"
                      )}
                      onClick={
                        col.sortable ? () => toggleSort(col.key) : undefined
                      }
                    >
                      <div className="flex items-center gap-x-1">
                        {col.title}
                        {col.sortable && renderSortIndicator(col.key)}
                      </div>
                      {col.resizable && (
                        <div
                          className="absolute right-0 top-1/4 h-1/2 w-[3px] cursor-col-resize rounded-full bg-gray-200 hover:bg-accent/60 active:bg-accent transition-colors"
                          onMouseDown={(e) => onResizeStart(colIdx, e)}
                        />
                      )}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {loading && auditLogs.length === 0 ? (
                  <tr>
                    <td
                      colSpan={columns.length}
                      className="text-center py-8 text-control-placeholder"
                    >
                      {t("common.loading")}
                    </td>
                  </tr>
                ) : auditLogs.length === 0 ? (
                  <tr>
                    <td
                      colSpan={columns.length}
                      className="text-center py-8 text-control-placeholder"
                    >
                      {t("common.no-data")}
                    </td>
                  </tr>
                ) : (
                  auditLogs.map((log, idx) => (
                    <tr
                      key={log.name || idx}
                      className={cn(
                        "border-b border-block-border",
                        idx % 2 === 1 && "bg-gray-50/50"
                      )}
                    >
                      {columns.map((col) => (
                        <td
                          key={col.key}
                          className="px-4 py-3 text-sm align-top overflow-hidden"
                        >
                          {col.render(log)}
                        </td>
                      ))}
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination footer */}
          <div className="flex items-center justify-end gap-x-2 mx-4 py-2">
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
                disabled={isFetchingMore || loading}
                onClick={loadMore}
              >
                <span className="text-sm text-control-light">
                  {isFetchingMore || loading
                    ? t("common.loading")
                    : t("common.load-more")}
                </span>
              </Button>
            )}
          </div>
        </div>
      ) : (
        <div className="mx-4 py-12 border rounded-sm flex items-center justify-center text-control-placeholder">
          {t("common.no-data")}
        </div>
      )}
    </div>
  );
}
