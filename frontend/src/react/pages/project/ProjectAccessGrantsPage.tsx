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
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  featureToRef,
  pushNotification,
  useAccessGrantStore,
  useCurrentUserV1,
  useDatabaseV1Store,
  useProjectV1Store,
  useUserStore,
} from "@/store";
import type { AccessFilter } from "@/store/modules/accessGrant";
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
  const accessGrantStore = useAccessGrantStore();
  const projectStore = useProjectV1Store();
  const databaseStore = useDatabaseV1Store();
  const userStore = useUserStore();
  const currentUser = useVueState(() => useCurrentUserV1().value);

  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const hasJITFeature = useVueState(
    () => featureToRef(PlanFeature.FEATURE_JIT).value
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
      const result = await databaseStore.fetchDatabases({
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
    [databaseStore, projectName, project]
  );

  // Server-side search for creator filter options
  const searchUsers = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      const result = await userStore.fetchUserList({
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
    [userStore, currentUser, t]
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
      const response = await accessGrantStore.listAccessGrants({
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
    [projectName, accessGrantStore, searchParams, orderBy]
  );

  const paged = usePagedData<AccessGrant>({
    sessionKey: `project-${projectName}-access-grants`,
    fetchList,
    enabled: canList,
  });

  const handleConfirm = useCallback(async () => {
    if (!confirmAction) return;
    const { type, grant } = confirmAction;
    if (type === "activate") {
      await accessGrantStore.activateAccessGrant(grant.name);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.activated"),
      });
    } else {
      await accessGrantStore.revokeAccessGrant(grant.name);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.revoked"),
      });
    }
    setConfirmAction(null);
    paged.refresh();
  }, [confirmAction, accessGrantStore, t, paged]);

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
              <div className="border rounded-sm overflow-hidden">
                <Table className="table-fixed">
                  <TableHeader>
                    <TableRow className="bg-control-bg">
                      <TableHead className="w-28">
                        {t("common.status")}
                      </TableHead>
                      <TableHead
                        sortable
                        sortActive={sortKey === "creator"}
                        sortDir={sortDir}
                        onSort={() => handleSort("creator")}
                        className="w-44"
                      >
                        {t("common.creator")}
                      </TableHead>
                      <TableHead
                        sortable
                        sortActive={sortKey === "create_time"}
                        sortDir={sortDir}
                        onSort={() => handleSort("create_time")}
                        className="w-44 hidden xl:table-cell"
                      >
                        {t("common.created-at")}
                      </TableHead>
                      <TableHead
                        sortable
                        sortActive={sortKey === "expire_time"}
                        sortDir={sortDir}
                        onSort={() => handleSort("expire_time")}
                        className="w-44 hidden xl:table-cell"
                      >
                        {t("common.expiration")}
                      </TableHead>
                      <TableHead>{t("common.statement")}</TableHead>
                      <TableHead className="w-48">
                        {t("common.databases")}
                      </TableHead>
                      <TableHead className="w-40" />
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
      <TableCell className="hidden xl:table-cell">
        <EllipsisText text={createdAt} />
      </TableCell>
      <TableCell className="hidden xl:table-cell">
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
            <a
              href={
                grant.issue.startsWith("/") ? grant.issue : `/${grant.issue}`
              }
              target="_blank"
              rel="noreferrer"
              onClick={(e) => e.stopPropagation()}
            >
              <Button variant="ghost" size="sm">
                {t("sql-editor.view-issue")}
              </Button>
            </a>
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
    <span
      key={target}
      className="normal-link hover:underline cursor-pointer text-sm"
      onClick={() => router.push({ path: `/${target}` })}
    >
      {getDatabaseName(target)}
    </span>
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
