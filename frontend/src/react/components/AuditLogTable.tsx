import { create, toJsonString } from "@bufbuild/protobuf";
import { AnySchema } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import { Download, ExternalLink, Maximize2, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { auditLogServiceClientConnect } from "@/connect";
import { ALL_METHODS_WITH_AUDIT } from "@/connect/methods";
import {
  AdvancedSearch,
  type ScopeOption,
  type SearchParams,
  type ValueOption,
} from "@/react/components/AdvancedSearch";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { TimeRangePicker } from "@/react/components/TimeRangePicker";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { usePlanFeature } from "@/react/hooks/useAppState";
import { useColumnWidths } from "@/react/hooks/useColumnWidths";
import { PagedTableFooter } from "@/react/hooks/usePagedData";
import {
  getPageSizeOptions,
  useSessionPageSize,
} from "@/react/hooks/useSessionPageSize";
import { cn } from "@/react/lib/utils";
import { pushNotification, useUserStore } from "@/store";
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
  AuditLog_Severity,
  ExportAuditLogsRequestSchema,
  SearchAuditLogsRequestSchema,
} from "@/types/proto-es/v1/audit_log_service_pb";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { IssueService } from "@/types/proto-es/v1/issue_service_pb";
import { PlanService } from "@/types/proto-es/v1/plan_service_pb";
import { RolloutService } from "@/types/proto-es/v1/rollout_service_pb";
import { SQLService } from "@/types/proto-es/v1/sql_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { protobufJsonRegistry } from "@/types/protobufJsonRegistry";
import {
  formatAbsoluteDateTime,
  getDefaultPagination,
  humanizeDurationV1,
} from "@/utils";

dayjs.extend(utc);

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
            <Maximize2 className="size-3" />
          </button>
        </div>
      </div>
      {showModal && (
        <Dialog
          open
          onOpenChange={(nextOpen) => !nextOpen && setShowModal(false)}
        >
          <DialogContent className="flex h-[calc(100vh-12rem)] w-[calc(100vw-12rem)] max-w-none flex-col p-0">
            <div className="flex items-center justify-between px-4 py-3 border-b">
              <DialogTitle className="text-base font-medium">
                {t("common.view-details")}
              </DialogTitle>
              <Button
                variant="ghost"
                size="icon"
                aria-label={t("common.close")}
                onClick={() => setShowModal(false)}
              >
                <X className="size-4" />
              </Button>
            </div>
            <div className="flex-1 overflow-auto p-4">
              <pre className="text-sm font-mono whitespace-pre-wrap break-all">
                {formatted}
              </pre>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </>
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
        <Download className="size-4 mr-1" />
        {t("common.export")}
      </Button>
      {open && !disabled && (
        <div
          className={cn(
            "absolute right-0 top-[42px] bg-background border border-control-border rounded-sm shadow-lg py-1 min-w-[100px]",
            LAYER_SURFACE_CLASS
          )}
        >
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
                registry: protobufJsonRegistry,
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
                registry: protobufJsonRegistry,
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
              <ExternalLink className="size-4 text-accent" />
            </a>
          );
        },
      },
    ],
    [t]
  );
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
  const hasAuditLogFeature = usePlanFeature(PlanFeature.FEATURE_AUDIT_LOG);
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

  const userStore = useUserStore();
  const searchUsers = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      const { users } = await userStore.fetchUserList({
        pageSize: getDefaultPagination(),
        filter: keyword.trim() ? { query: keyword } : undefined,
      });
      return users.map((u) => ({
        value: u.email,
        keywords: [u.email, u.title],
      }));
    },
    [userStore]
  );

  const scopeOptions = useMemo((): ScopeOption[] => {
    return [
      {
        id: "actor",
        title: t("audit-log.advanced-search.scope.actor.title"),
        description: t("audit-log.advanced-search.scope.actor.description"),
        onSearch: searchUsers,
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
  }, [t, searchUsers]);

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
            <Table
              className="border-t border-block-border table-fixed"
              style={{ minWidth: `${totalWidth}px` }}
            >
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
                      className="text-sm text-main whitespace-nowrap"
                      sortable={col.sortable}
                      sortActive={col.sortable && sortKey === col.key}
                      sortDir={sortOrder}
                      onSort={
                        col.sortable ? () => toggleSort(col.key) : undefined
                      }
                      resizable={col.resizable}
                      onResizeStart={
                        col.resizable
                          ? (e) => onResizeStart(colIdx, e)
                          : undefined
                      }
                    >
                      {col.title}
                    </TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading && auditLogs.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={columns.length}
                      className="text-center py-8 text-control-placeholder"
                    >
                      {t("common.loading")}
                    </TableCell>
                  </TableRow>
                ) : auditLogs.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={columns.length}
                      className="text-center py-8 text-control-placeholder"
                    >
                      {t("common.no-data")}
                    </TableCell>
                  </TableRow>
                ) : (
                  auditLogs.map((log, idx) => (
                    <TableRow key={log.name || idx}>
                      {columns.map((col) => (
                        <TableCell
                          key={col.key}
                          className="align-top overflow-hidden"
                        >
                          {col.render(log)}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          {/* Pagination footer */}
          <div className="mt-4 mx-2">
            <PagedTableFooter
              pageSize={pageSize}
              pageSizeOptions={pageSizeOptions}
              onPageSizeChange={setPageSize}
              hasMore={hasMore}
              isFetchingMore={isFetchingMore || loading}
              onLoadMore={loadMore}
            />
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
