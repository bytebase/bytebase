import { Tooltip as BaseTooltip } from "@base-ui/react/tooltip";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AdvancedSearch,
  emptySearchParams,
  type ScopeOption,
  type SearchParams,
  type ValueOption,
} from "@/react/components/AdvancedSearch";
import { ComponentPermissionGuard } from "@/react/components/ComponentPermissionGuard";
import { EngineIcon } from "@/react/components/EngineIcon";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { RouterLink } from "@/react/components/RouterLink";
import { TimeRangePicker } from "@/react/components/TimeRangePicker";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { useColumnWidths } from "@/react/hooks/useColumnWidths";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { useAppStore } from "@/react/stores/app";
import type { AccessGrantFilter as AccessFilter } from "@/react/stores/app/types";
import { pushNotification } from "@/store";
import { extractUserEmail, projectNamePrefix } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import { AccessGrant_Status } from "@/types/proto-es/v1/access_grant_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  type AccessGrantDisplayStatus,
  type AccessGrantFilterStatus,
  formatAbsoluteDateTime,
  getAccessGrantDisplayStatus,
  getAccessGrantDisplayStatusText,
  getAccessGrantExpirationText,
  getAccessGrantStatusTagType,
  getDefaultPagination,
  hasProjectPermissionV2,
} from "@/utils";
import { extractDatabaseResourceName } from "@/utils/v1/database";

type SortKey = "creator" | "create_time" | "expire_time";
type SortDir = "asc" | "desc";

// Column descriptor for the access-grants table. The column array is
// built inside the component (so titles resolve via `t()` and react
// to language switches); only the shape lives at module scope.
//
// Position in the array is the `<colgroup>` order — keep this in sync
// with the cell order inside `<AccessGrantRow>` (`useColumnWidths`
// indexes positionally).
//
// - `title`        — header label; omit for a blank header (actions col).
// - `defaultWidth` — initial render width; user-resizable from there.
// - `minWidth`     — drag floor so a column can't collapse to a sliver.
// - `sortKey`      — present iff the column participates in server sort.
// - `resizable`    — defaults true; set false for purely action columns
//                    where a too-narrow width clips the button row.
type GrantColumn = {
  key: string;
  title?: string;
  defaultWidth: number;
  minWidth: number;
  sortKey?: SortKey;
  resizable?: boolean;
};

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
  const { database: dbName } = extractDatabaseResourceName(db.name);
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
              className="w-5 h-5 rounded-full text-white text-xs flex items-center justify-center shrink-0"
              style={{
                backgroundColor: `hsl(${hashCode(u.title) % 360}, 55%, 55%)`,
              }}
            >
              {u.title.charAt(0).toUpperCase()}
            </span>
            <span>{u.title}</span>
            {currentUser && u.name === currentUser.name && (
              <span className="text-xs bg-green-100 text-green-700 rounded-full px-1.5">
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
      const statuses = getValuesFromScopes(
        searchParams,
        "status"
      ) as AccessGrantFilterStatus[];
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

  // Translated column descriptors. Built inside the component (not at
  // module scope) so `title` strings resolve via `t()` and update
  // automatically on language switches. Memoized on `t` so the
  // descriptor array identity is stable between renders within the
  // same language — which matters because `useColumnWidths` reads the
  // initial widths from this array on first render.
  const columns = useMemo<GrantColumn[]>(
    () => [
      {
        key: "status",
        title: t("common.status"),
        defaultWidth: 112,
        minWidth: 80,
      },
      {
        key: "creator",
        title: t("common.creator"),
        defaultWidth: 200,
        minWidth: 140,
        sortKey: "creator",
      },
      {
        key: "created",
        title: t("common.created-at"),
        defaultWidth: 200,
        minWidth: 160,
        sortKey: "create_time",
      },
      {
        key: "expiration",
        title: t("common.expiration"),
        defaultWidth: 200,
        minWidth: 160,
        sortKey: "expire_time",
      },
      {
        key: "statement",
        title: t("common.statement"),
        defaultWidth: 400,
        minWidth: 200,
      },
      {
        key: "databases",
        title: t("common.databases"),
        defaultWidth: 240,
        minWidth: 160,
      },
      // Trailing actions column — no title (blank header), fixed
      // width sized for two ghost buttons + "View issue".
      { key: "actions", defaultWidth: 180, minWidth: 120, resizable: false },
    ],
    [t]
  );

  // User-controlled column widths so long statements / databases /
  // expiration values aren't permanently truncated by the table's
  // fixed-width layout. The hook owns the per-column width state and
  // the mousedown→mousemove→mouseup drag pipeline.
  const { widths, totalWidth, onResizeStart } = useColumnWidths(columns);

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
    <div className="py-4 w-full flex flex-col">
      <div className="mx-4 mb-2">
        <FeatureAttention feature={PlanFeature.FEATURE_JIT} />
      </div>

      <ComponentPermissionGuard
        permissions={["bb.accessGrants.list"]}
        project={project}
        className="mx-4"
      >
        <>
          <div className="px-4 pb-2 flex items-center gap-x-2">
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
          </div>

          {!hasJITFeature ? (
            <div className="mx-4 py-12 border rounded-sm flex items-center justify-center text-control-light">
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
            <div className="px-4">
              {/*
                `overflow-x-auto` lets users drag a column past the
                container width without clipping the trailing columns.
                The previous `overflow-hidden` would have swallowed
                the resized overflow.
              */}
              <div className="border rounded-sm overflow-x-auto">
                <Table
                  className="table-fixed"
                  style={{ minWidth: `${totalWidth}px` }}
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

              <div className="mt-4">
                <PagedTableFooter
                  pageSize={paged.pageSize}
                  pageSizeOptions={paged.pageSizeOptions}
                  onPageSizeChange={paged.onPageSizeChange}
                  hasMore={paged.hasMore}
                  isFetchingMore={paged.isFetchingMore}
                  onLoadMore={paged.loadMore}
                />
              </div>
            </div>
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
            <Button variant="outline" onClick={() => setConfirmAction(null)}>
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
    </div>
  );
}

// ---------------------------------------------------------------------------
// AccessGrantRow
// ---------------------------------------------------------------------------

function AccessGrantRow({
  grant,
  canActivate,
  canRevoke,
  onActivate,
  onRevoke,
}: {
  grant: AccessGrant;
  canActivate: boolean;
  canRevoke: boolean;
  onActivate: () => void;
  onRevoke: () => void;
}) {
  const { t } = useTranslation();
  const status = getAccessGrantDisplayStatus(grant);

  const createdAt = grant.createTime
    ? formatAbsoluteDateTime(getTimeForPbTimestampProtoEs(grant.createTime))
    : "-";

  const expirationInfo = getAccessGrantExpirationText(grant);
  const expiration =
    expirationInfo.type === "datetime" ? expirationInfo.value : "-";

  return (
    <TableRow>
      <TableCell>
        <Badge variant={statusTagVariant(status)}>
          {getAccessGrantDisplayStatusText(grant)}
        </Badge>
      </TableCell>
      <TableCell>
        <EllipsisText text={extractUserEmail(grant.creator)} />
      </TableCell>
      <TableCell>
        <EllipsisText text={createdAt} />
      </TableCell>
      <TableCell>
        <EllipsisText text={expiration} />
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
            <Button variant="ghost" size="sm" onClick={onActivate}>
              {t("sql-editor.activate-access")}
            </Button>
          )}
          {status === "ACTIVE" && canRevoke && (
            <Button
              variant="ghost"
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
              <Button variant="ghost" size="sm">
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
    <BaseTooltip.Provider delay={300}>
      <BaseTooltip.Root open={isTruncated && open} onOpenChange={setOpen}>
        <BaseTooltip.Trigger
          render={
            <span
              ref={ref}
              className="block font-mono text-xs truncate min-w-0 flex-1"
            />
          }
        >
          {query}
        </BaseTooltip.Trigger>
        <BaseTooltip.Portal container={getLayerRoot("overlay")}>
          <BaseTooltip.Positioner
            side="top"
            sideOffset={4}
            className={LAYER_SURFACE_CLASS}
          >
            <BaseTooltip.Popup className="rounded-sm bg-main px-2.5 py-1.5 text-xs text-main-text shadow-md">
              <pre className="max-w-lg whitespace-pre-wrap font-mono">
                {query}
              </pre>
              <BaseTooltip.Arrow className="fill-main" />
            </BaseTooltip.Popup>
          </BaseTooltip.Positioner>
        </BaseTooltip.Portal>
      </BaseTooltip.Root>
    </BaseTooltip.Provider>
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
    <div className="flex items-center truncate gap-x-0.5">
      {visible.map((target, i) => (
        <span key={target} className="flex items-center gap-x-0.5">
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
