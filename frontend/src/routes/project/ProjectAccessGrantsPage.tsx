import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AdvancedSearch,
  emptySearchParams,
  type ScopeOption,
  type SearchParams,
  type ValueOption,
} from "@/components/AdvancedSearch";
import { ComponentPermissionGuard } from "@/components/ComponentPermissionGuard";
import { EngineIcon } from "@/components/EngineIcon";
import { FeatureAttention } from "@/components/FeatureAttention";
import { HumanizeTs } from "@/components/HumanizeTs";
import {
  ProjectPageContent,
  ProjectPageFooter,
  ProjectPageLayout,
  ProjectPageToolbar,
} from "@/components/ProjectPageLayout";
import { RouterLink } from "@/components/RouterLink";
import { TimeRangePicker } from "@/components/TimeRangePicker";
import {
  TIMESTAMP_COLUMN_MIN_WIDTH,
  TIMESTAMP_COLUMN_WIDTH,
} from "@/components/timestampColumn";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { EllipsisText } from "@/components/ui/ellipsis-text";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { BlockTooltip, Tooltip } from "@/components/ui/tooltip";
import { useCurrentUser } from "@/hooks/useAppState";
import {
  type ColumnWithWidth,
  distributeColumnWidths,
  useColumnWidths,
} from "@/hooks/useColumnWidths";
import { PagedTableFooter, usePagedData } from "@/hooks/usePagedData";
import { useProjectByName } from "@/hooks/useProjectByName";
import { useTimeReading } from "@/hooks/useTimeReading";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import type { AccessGrantFilter as AccessFilter } from "@/stores/app/types";
import {
  extractUserEmail,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import { AccessGrant_Status } from "@/types/proto-es/v1/access_grant_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  type AccessGrantDisplayStatus,
  accessGrantStatusReading,
  getAccessGrantDisplayStatusText,
  getAccessGrantExpireTimeMs,
  getAccessGrantStatusTagType,
  getDefaultPagination,
  hasProjectPermissionV2,
} from "@/utils";
import { extractDatabaseResourceName } from "@/utils/v1/database";

type SortKey = "creator" | "create_time" | "expire_time";
type SortDir = "asc" | "desc";

// Column descriptor for the access-grants table. Position in the array is
// the `<colgroup>` order — keep this in sync with the cell order inside
// `<AccessGrantRow>` (`useColumnWidths` indexes positionally). The width
// fields are `ColumnWithWidth`'s.
//
// - `title`     — header label; omit for a blank header (actions col).
// - `sortKey`   — present iff the column participates in server sort.
// - `resizable` — defaults true; set false for purely action columns where a
//                 too-narrow width clips the button row.
type GrantColumn = ColumnWithWidth & {
  key: string;
  title?: string;
  minWidth: number;
  sortKey?: SortKey;
  resizable?: boolean;
};

/** The table's columns, in `<colgroup>` order; titles follow the language. */
export const grantColumns = (t: (key: string) => string): GrantColumn[] => [
  {
    key: "status",
    title: t("common.status"),
    defaultWidth: 160,
    minWidth: 128,
  },
  {
    key: "creator",
    title: t("common.creator"),
    defaultWidth: 200,
    minWidth: 128,
    sortKey: "creator",
    yieldOrder: 3,
  },
  {
    key: "created",
    title: t("common.created-at"),
    defaultWidth: TIMESTAMP_COLUMN_WIDTH.operational,
    minWidth: TIMESTAMP_COLUMN_MIN_WIDTH,
    grow: false,
    sortKey: "create_time",
    yieldOrder: 1,
  },
  {
    key: "expiration",
    title: t("common.expiration"),
    defaultWidth: TIMESTAMP_COLUMN_WIDTH.operational,
    minWidth: TIMESTAMP_COLUMN_MIN_WIDTH,
    grow: false,
    sortKey: "expire_time",
    yieldOrder: 1,
  },
  {
    key: "statement",
    title: t("common.statement"),
    defaultWidth: 400,
    minWidth: 180,
    yieldOrder: 2,
  },
  {
    key: "databases",
    title: t("common.databases"),
    defaultWidth: 240,
    minWidth: 128,
    yieldOrder: 4,
  },
  // Trailing actions column — no title (blank header), fixed
  // width sized for two ghost buttons + "View issue".
  { key: "actions", defaultWidth: 140, minWidth: 96, resizable: false },
];

function hashCode(str: string): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = (hash << 5) - hash + str.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash);
}

function getDatabaseName(target: string) {
  const match = target.match(/databases\/(.+)$/);
  return match ? match[1] : target;
}

function statusTagVariant(
  status: AccessGrantDisplayStatus
): "success" | "warning" | "destructive" | "default" {
  const tagType = getAccessGrantStatusTagType(status);
  if (tagType === "error") return "destructive";
  return tagType;
}

function getValuesFromScopes(params: SearchParams, id: string): string[] {
  return params.scopes.filter((s) => s.id === id).map((s) => s.value);
}

function getValueFromScopes(params: SearchParams, id: string): string {
  return params.scopes.find((s) => s.id === id)?.value ?? "";
}

function mapDatabase(db: Database) {
  const { databaseName: dbName } = extractDatabaseResourceName(db.name);
  const inst = db.instanceResource;
  const envId = (db.effectiveEnvironment ?? db.environment ?? "")
    .split("/")
    .pop();
  return {
    value: db.name,
    dbName,
    instanceTitle: inst?.title ?? "",
    envId: envId ?? "",
    engine: inst?.engine,
  };
}

export function ProjectAccessGrantsPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  void projectsByName;
  const listUsers = useAppStore((state) => state.listUsers);
  const listAccessGrants = useAppStore((state) => state.listAccessGrants);
  const fetchIssueByName = useAppStore((state) => state.fetchIssueByName);
  const activateAccessGrant = useAppStore((state) => state.activateAccessGrant);
  const revokeAccessGrant = useAppStore((state) => state.revokeAccessGrant);
  const currentUser = useCurrentUser();

  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useProjectByName(projectName);

  const hasJITFeature = useAppStore((s) =>
    s.hasInstanceFeature(PlanFeature.FEATURE_JIT)
  );
  const canList = useMemo(
    () =>
      project ? hasProjectPermissionV2(project, "bb.accessGrants.list") : false,
    [project]
  );
  const canActivate = useMemo(
    () =>
      project
        ? hasProjectPermissionV2(project, "bb.accessGrants.activate")
        : false,
    [project]
  );
  const canRevoke = useMemo(
    () =>
      project
        ? hasProjectPermissionV2(project, "bb.accessGrants.revoke")
        : false,
    [project]
  );

  // --- Search state ---
  const [searchParams, setSearchParams] =
    useState<SearchParams>(emptySearchParams);

  // --- Sort state ---
  const [sortKey, setSortKey] = useState<SortKey | "">("");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  const [confirmAction, setConfirmAction] = useState<{
    type: "activate" | "revoke";
    grant: AccessGrant;
  } | null>(null);
  const [issueByGrantName, setIssueByGrantName] = useState<Map<string, Issue>>(
    new Map()
  );

  // Server-side search for database filter options
  const searchDatabases = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      if (!project || !hasProjectPermissionV2(project, "bb.databases.list")) {
        return [];
      }
      const result = await useAppStore.getState().fetchDatabases({
        parent: projectName,
        pageSize: getDefaultPagination(),
        filter: keyword ? { query: keyword } : undefined,
      });
      return result.databases.map((db) => {
        const mapped = mapDatabase(db);
        return {
          value: mapped.value,
          keywords: [
            mapped.dbName,
            mapped.instanceTitle,
            mapped.envId,
            mapped.value,
          ],
          custom: true,
          render: () => (
            <span className="inline-flex items-center gap-x-1">
              {mapped.engine && (
                <EngineIcon engine={mapped.engine} className="h-4 w-4" />
              )}
              <span>{mapped.instanceTitle}</span>
              <span className="text-control-placeholder">&gt;</span>
              <span>{mapped.envId}</span>
              <span className="text-control-placeholder">&gt;</span>
              <span>{mapped.dbName}</span>
            </span>
          ),
        };
      });
    },
    [projectName, project]
  );

  // Server-side search for creator filter options
  const searchUsers = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      const result = await listUsers({
        pageSize: getDefaultPagination(),
        filter: keyword ? { query: keyword } : undefined,
      });
      return result.users.map((u) => ({
        value: u.email,
        keywords: [u.email, u.title],
        custom: true,
        render: () => (
          <span className="inline-flex items-center gap-x-1.5">
            <span
              className="w-5 h-5 rounded-full text-accent-text text-xs flex items-center justify-center shrink-0"
              style={{
                backgroundColor: `hsl(${hashCode(u.title) % 360}, 55%, 55%)`,
              }}
            >
              {u.title.charAt(0).toUpperCase()}
            </span>
            <span>{u.title}</span>
            {currentUser && u.name === currentUser.name && (
              <span className="text-xs bg-success/10 text-success rounded-full px-1.5">
                {t("common.you")}
              </span>
            )}
            <span className="text-control-light">{u.email}</span>
          </span>
        ),
      }));
    },
    [listUsers, currentUser, t]
  );

  const scopeOptions: ScopeOption[] = useMemo(
    () => [
      {
        id: "status",
        title: t("common.status"),
        description: t(
          "issue.access-grant.advanced-search.scope.status.description"
        ),
        allowMultiple: true,
        options: [
          {
            value: AccessGrant_Status[AccessGrant_Status.ACTIVE],
            keywords: ["active"],
            render: () => <span>{t("common.active")}</span>,
          },
          {
            value: AccessGrant_Status[AccessGrant_Status.PENDING],
            keywords: ["pending"],
            render: () => <span>{t("common.pending")}</span>,
          },
          {
            value: "EXPIRED",
            keywords: ["expired"],
            render: () => <span>{t("sql-editor.expired")}</span>,
          },
          {
            value: AccessGrant_Status[AccessGrant_Status.REVOKED],
            keywords: ["revoked"],
            render: () => <span>{t("common.revoked")}</span>,
          },
          {
            value: "REJECTED",
            keywords: ["rejected"],
            render: () => <span>{t("common.rejected")}</span>,
          },
          {
            value: "CANCELED",
            keywords: ["canceled", "cancelled"],
            render: () => <span>{t("common.canceled")}</span>,
          },
        ],
      },
      {
        id: "database",
        title: t("common.database"),
        description: t("issue.advanced-search.scope.database.description"),
        onSearch: searchDatabases,
      },
      {
        id: "creator",
        title: t("common.creator"),
        description: t("issue.advanced-search.scope.creator.description"),
        onSearch: searchUsers,
      },
      {
        id: "unmask",
        title: t("sql-editor.grant-type-unmask"),
        description: t(
          "issue.access-grant.advanced-search.scope.unmask.description"
        ),
        options: [
          {
            value: "true",
            keywords: ["yes", "true"],
            render: () => <span>{t("common.yes")}</span>,
          },
          {
            value: "false",
            keywords: ["no", "false"],
            render: () => <span>{t("common.no")}</span>,
          },
        ],
      },
      {
        id: "export",
        title: t("sql-editor.grant-type-export"),
        description: t(
          "issue.access-grant.advanced-search.scope.export.description"
        ),
        options: [
          {
            value: "true",
            keywords: ["yes", "true"],
            render: () => <span>{t("common.yes")}</span>,
          },
          {
            value: "false",
            keywords: ["no", "false"],
            render: () => <span>{t("common.no")}</span>,
          },
        ],
      },
    ],
    [t, searchDatabases, searchUsers]
  );

  const orderBy = useMemo(() => {
    if (!sortKey) return "";
    return `${sortKey} ${sortDir}`;
  }, [sortKey, sortDir]);

  const handleSort = useCallback(
    (key: SortKey) => {
      if (sortKey === key) {
        if (sortDir === "desc") {
          setSortDir("asc");
        } else {
          setSortKey("");
          setSortDir("desc");
        }
      } else {
        setSortKey(key);
        setSortDir("desc");
      }
    },
    [sortKey, sortDir]
  );

  const fetchList = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const filter: AccessFilter = {};
      const statuses = getValuesFromScopes(searchParams, "status") as Exclude<
        AccessGrantDisplayStatus,
        "UNKNOWN"
      >[];
      if (statuses.length > 0) {
        filter.status = statuses;
      }
      const creator = getValueFromScopes(searchParams, "creator");
      if (creator) {
        filter.creator = `users/${creator}`;
      }
      const database = getValueFromScopes(searchParams, "database");
      if (database) {
        filter.target = database;
      }
      const unmask = getValueFromScopes(searchParams, "unmask");
      if (unmask === "true" || unmask === "false") {
        filter.unmask = unmask === "true";
      }
      const exportScope = getValueFromScopes(searchParams, "export");
      if (exportScope === "true" || exportScope === "false") {
        filter.export = exportScope === "true";
      }
      const query = searchParams.query.trim();
      if (query) {
        filter.statement = query;
      }
      const createdScope = searchParams.scopes.find((s) => s.id === "created");
      if (createdScope) {
        const parts = createdScope.value.split(",");
        if (parts.length === 2) {
          filter.createdTsAfter = parseInt(parts[0], 10);
          filter.createdTsBefore = parseInt(parts[1], 10);
        }
      }
      const response = await listAccessGrants({
        parent: projectName,
        filter,
        pageSize: params.pageSize,
        pageToken: params.pageToken || undefined,
        orderBy,
      });
      return {
        list: response.accessGrants,
        nextPageToken: response.nextPageToken,
      };
    },
    [projectName, listAccessGrants, searchParams, orderBy]
  );

  const paged = usePagedData<AccessGrant>({
    sessionKey: `project-${projectName}-access-grants`,
    fetchList,
    enabled: canList,
  });

  useEffect(() => {
    const pendingWithMissingIssue = paged.dataList.filter(
      (g) =>
        g.status === AccessGrant_Status.PENDING &&
        g.issue &&
        !issueByGrantName.has(g.name)
    );
    if (pendingWithMissingIssue.length === 0) {
      return;
    }

    void (async () => {
      const results = await Promise.all(
        pendingWithMissingIssue.map(async (g) => {
          try {
            const issue = await fetchIssueByName(g.issue, true);
            return { grantName: g.name, issue };
          } catch {
            return undefined;
          }
        })
      );
      setIssueByGrantName((prev) => {
        const next = new Map(prev);
        let changed = false;
        for (const r of results) {
          if (r) {
            next.set(r.grantName, r.issue);
            changed = true;
          }
        }
        return changed ? next : prev;
      });
    })();
  }, [fetchIssueByName, issueByGrantName, paged.dataList]);

  // Memoized on `t` so the array identity is stable within a language:
  // `useColumnWidths` reads the initial widths from it on first render.
  const columns = useMemo(() => grantColumns(t), [t]);

  // User-controlled column widths so long statements / databases /
  // expiration values aren't permanently truncated by the table's
  // fixed-width layout. The hook owns the per-column width state and
  // the mousedown→mousemove→mouseup drag pipeline.
  const { widths, totalWidth, onResizeStart, setWidths } =
    useColumnWidths(columns);

  const didFitColumnsRef = useRef(false);
  const fitTableContainer = useCallback(
    (node: HTMLDivElement | null) => {
      if (!node || didFitColumnsRef.current) return;
      const width = node.clientWidth;
      if (width <= 0) return;
      didFitColumnsRef.current = true;
      setWidths(distributeColumnWidths(columns, width));
    },
    [columns, setWidths]
  );

  const handleConfirm = useCallback(async () => {
    if (!confirmAction) return;
    const { type, grant } = confirmAction;
    if (type === "activate") {
      await activateAccessGrant(grant.name);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.activated"),
      });
    } else {
      await revokeAccessGrant(grant.name);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.revoked"),
      });
    }
    setConfirmAction(null);
    paged.refresh();
  }, [confirmAction, activateAccessGrant, revokeAccessGrant, t, paged]);

  return (
    <ProjectPageLayout>
      <FeatureAttention feature={PlanFeature.FEATURE_JIT} />

      <ComponentPermissionGuard
        permissions={["bb.accessGrants.list"]}
        project={project}
      >
        <>
          <ProjectPageToolbar align="start">
            <AdvancedSearch
              params={searchParams}
              onParamsChange={setSearchParams}
              scopeOptions={scopeOptions}
              placeholder={t("issue.advanced-search.filter")}
            />
            <TimeRangePicker
              params={searchParams}
              onParamsChange={setSearchParams}
            />
          </ProjectPageToolbar>

          {!hasJITFeature ? (
            <div className="py-12 border rounded-sm flex items-center justify-center text-control-light">
              {t("common.no-data")}
            </div>
          ) : paged.isLoading ? (
            <div className="flex justify-center py-8 text-control-light">
              {t("common.loading")}
            </div>
          ) : paged.dataList.length === 0 ? (
            <div className="flex justify-center py-8 text-control-light">
              {t("common.no-data")}
            </div>
          ) : (
            <ProjectPageContent ref={fitTableContainer}>
              <div className="border rounded-sm overflow-x-auto">
                <Table
                  className="w-auto table-fixed"
                  style={{ width: `${totalWidth}px` }}
                >
                  {/*
                    `<colgroup>` order mirrors `columns`, which in turn
                    mirrors the cell order inside `<AccessGrantRow>` —
                    `useColumnWidths` indexes positionally, not by key.
                  */}
                  <colgroup>
                    {widths.map((w, i) => (
                      <col key={columns[i].key} style={{ width: `${w}px` }} />
                    ))}
                  </colgroup>
                  <TableHeader>
                    <TableRow className="bg-control-bg">
                      {columns.map((col, colIdx) => {
                        const isSortable = col.sortKey !== undefined;
                        const isResizable = col.resizable !== false;
                        return (
                          <TableHead
                            key={col.key}
                            sortable={isSortable}
                            sortActive={isSortable && sortKey === col.sortKey}
                            sortDir={sortDir}
                            onSort={
                              isSortable
                                ? () => handleSort(col.sortKey!)
                                : undefined
                            }
                            resizable={isResizable}
                            onResizeStart={
                              isResizable
                                ? (e) => onResizeStart(colIdx, e)
                                : undefined
                            }
                          >
                            {col.title ?? null}
                          </TableHead>
                        );
                      })}
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {paged.dataList.map((grant) => (
                      <AccessGrantRow
                        key={grant.name}
                        grant={grant}
                        issue={issueByGrantName.get(grant.name)}
                        canActivate={canActivate}
                        canRevoke={canRevoke}
                        onActivate={() =>
                          setConfirmAction({ type: "activate", grant })
                        }
                        onRevoke={() =>
                          setConfirmAction({ type: "revoke", grant })
                        }
                      />
                    ))}
                  </TableBody>
                </Table>
              </div>

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
            </ProjectPageContent>
          )}
        </>
      </ComponentPermissionGuard>

      {/* Activate / Revoke confirmation dialog */}
      <Dialog
        open={confirmAction !== null}
        onOpenChange={(open) => {
          if (!open) setConfirmAction(null);
        }}
      >
        <DialogContent className="p-6">
          <DialogTitle>
            {confirmAction?.type === "activate"
              ? t("sql-editor.activate-access")
              : t("sql-editor.revoke-access")}
          </DialogTitle>
          <p className="text-sm text-control-light mt-2">
            {confirmAction?.type === "activate"
              ? t("sql-editor.activate-confirm")
              : t("sql-editor.revoke-confirm")}
          </p>
          <div className="flex justify-end gap-x-2 mt-4">
            <Button appearance="outline" onClick={() => setConfirmAction(null)}>
              {t("common.cancel")}
            </Button>
            <Button
              variant={
                confirmAction?.type === "revoke" ? "destructive" : "default"
              }
              onClick={handleConfirm}
            >
              {confirmAction?.type === "activate"
                ? t("sql-editor.activate-access")
                : t("sql-editor.revoke-access")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </ProjectPageLayout>
  );
}

// ---------------------------------------------------------------------------
// AccessGrantRow
// ---------------------------------------------------------------------------

export function AccessGrantRow({
  grant,
  issue,
  canActivate,
  canRevoke,
  onActivate,
  onRevoke,
}: {
  grant: AccessGrant;
  issue?: Issue;
  canActivate: boolean;
  canRevoke: boolean;
  onActivate: () => void;
  onRevoke: () => void;
}) {
  const { t } = useTranslation();
  const status = useTimeReading(accessGrantStatusReading, { grant, issue });
  const createdTimeMs = getTimeForPbTimestampProtoEs(grant.createTime);
  const expireTimeMs = getAccessGrantExpireTimeMs(grant);

  return (
    <TableRow>
      <TableCell>
        <Badge variant={statusTagVariant(status)}>
          {getAccessGrantDisplayStatusText(status)}
        </Badge>
      </TableCell>
      <TableCell>
        <EllipsisText text={extractUserEmail(grant.creator)} />
      </TableCell>
      <TableCell>
        {createdTimeMs !== undefined ? (
          <HumanizeTs
            className="block truncate"
            mode="operational"
            tsMs={createdTimeMs}
          />
        ) : (
          "-"
        )}
      </TableCell>
      <TableCell>
        {expireTimeMs !== undefined ? (
          <HumanizeTs
            className="block truncate"
            mode="operational"
            tsMs={expireTimeMs}
          />
        ) : (
          "-"
        )}
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-x-1 overflow-hidden">
          <TruncatedQuery query={grant.query} />
          {grant.unmask && (
            <Badge variant="warning" className="shrink-0">
              {t("sql-editor.grant-type-unmask")}
            </Badge>
          )}
          {grant.export && (
            <Badge variant="default" className="shrink-0">
              {t("sql-editor.grant-type-export")}
            </Badge>
          )}
        </div>
      </TableCell>
      <TableCell>
        <DatabaseTargets targets={grant.targets} />
      </TableCell>
      <TableCell>
        <div className="flex items-center justify-end gap-x-1">
          {status === "REVOKED" && canActivate && (
            <Button appearance="secondary" size="sm" onClick={onActivate}>
              {t("sql-editor.activate-access")}
            </Button>
          )}
          {status === "ACTIVE" && canRevoke && (
            <Button
              appearance="secondary"
              size="sm"
              className="text-error"
              onClick={onRevoke}
            >
              {t("sql-editor.revoke-access")}
            </Button>
          )}
          {grant.issue && (
            <RouterLink
              to={grant.issue.startsWith("/") ? grant.issue : `/${grant.issue}`}
              target="_blank"
              rel="noreferrer"
              onClick={(e) => e.stopPropagation()}
            >
              <Button appearance="secondary" size="sm">
                {t("sql-editor.view-issue")}
              </Button>
            </RouterLink>
          )}
        </div>
      </TableCell>
    </TableRow>
  );
}

// ---------------------------------------------------------------------------
// TruncatedQuery
// ---------------------------------------------------------------------------

function TruncatedQuery({ query }: { query: string }) {
  const ref = useRef<HTMLSpanElement>(null);
  const [isTruncated, setIsTruncated] = useState(false);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const check = () => {
      setIsTruncated(el.scrollWidth > el.clientWidth);
    };
    check();
    const ro = new ResizeObserver(check);
    ro.observe(el);
    return () => ro.disconnect();
  }, [query]);

  return (
    <BlockTooltip
      content={
        <pre className="max-w-lg whitespace-pre-wrap font-mono">{query}</pre>
      }
      delayDuration={300}
      open={isTruncated && open}
      onOpenChange={setOpen}
      popupClassName="max-w-lg"
      render={
        <span
          ref={ref}
          className="block font-mono text-xs truncate min-w-0 flex-1"
        />
      }
    >
      {query}
    </BlockTooltip>
  );
}

// ---------------------------------------------------------------------------
// DatabaseTargets
// ---------------------------------------------------------------------------

function DatabaseTargets({ targets }: { targets: string[] }) {
  if (targets.length === 0) return <span>-</span>;

  const visible = targets.slice(0, 2);
  const rest = targets.length - visible.length;

  const renderLink = (target: string) => (
    <RouterLink
      key={target}
      to={{ path: `/${target}` }}
      className="normal-link hover:underline cursor-pointer text-sm"
    >
      {getDatabaseName(target)}
    </RouterLink>
  );

  const inline = (
    <div className="flex items-center truncate gap-x-1">
      {visible.map((target, i) => (
        <span key={target} className="flex items-center gap-x-1">
          {i > 0 && <span className="text-sm">, </span>}
          {renderLink(target)}
        </span>
      ))}
      {rest > 0 && (
        <span className="text-sm text-control-placeholder"> +{rest}</span>
      )}
    </div>
  );

  if (rest <= 0) return inline;

  return (
    <Tooltip
      content={
        <div className="flex flex-col gap-y-1">
          {targets.map((target) => renderLink(target))}
        </div>
      }
    >
      {inline}
    </Tooltip>
  );
}
