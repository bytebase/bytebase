import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import {
  PagedTableFooter,
  usePagedData,
} from "@/react/pages/settings/shared/usePagedData";
import { router } from "@/router";
import {
  featureToRef,
  pushNotification,
  useAccessGrantStore,
  useProjectV1Store,
} from "@/store";
import type { AccessFilter } from "@/store/modules/accessGrant";
import { extractUserEmail, projectNamePrefix } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  type AccessGrantDisplayStatus,
  type AccessGrantFilterStatus,
  formatAbsoluteDateTime,
  getAccessGrantDisplayStatus,
  getAccessGrantDisplayStatusText,
  getAccessGrantExpirationText,
  getAccessGrantStatusTagType,
  hasProjectPermissionV2,
} from "@/utils";

const STATUS_OPTIONS: AccessGrantFilterStatus[] = [
  "ACTIVE",
  "PENDING",
  "REVOKED",
  "EXPIRED",
];

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

export function ProjectAccessGrantsPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const accessGrantStore = useAccessGrantStore();
  const projectStore = useProjectV1Store();

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

  const [searchText, setSearchText] = useState("");
  const [statusFilter, setStatusFilter] = useState<
    AccessGrantFilterStatus | ""
  >("");
  const [confirmAction, setConfirmAction] = useState<{
    type: "activate" | "revoke";
    grant: AccessGrant;
  } | null>(null);

  const fetchList = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const filter: AccessFilter = {};
      if (statusFilter) {
        filter.status = [statusFilter];
      }
      const query = searchText.trim();
      if (query) {
        filter.statement = query;
      }
      const response = await accessGrantStore.listAccessGrants({
        parent: projectName,
        filter,
        pageSize: params.pageSize,
        pageToken: params.pageToken || undefined,
      });
      return {
        list: response.accessGrants,
        nextPageToken: response.nextPageToken,
      };
    },
    [projectName, accessGrantStore, searchText, statusFilter]
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

      {canList ? (
        <>
          <div className="px-4 pb-2 flex items-center gap-x-2">
            <Input
              className="flex-1"
              placeholder={t("issue.advanced-search.filter")}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
            />
            <select
              value={statusFilter}
              onChange={(e) =>
                setStatusFilter(e.target.value as AccessGrantFilterStatus | "")
              }
              className="border border-control-border rounded-sm text-sm pl-2 pr-6 py-1.5 min-w-[8rem]"
            >
              <option value="">{t("common.all")}</option>
              {STATUS_OPTIONS.map((s) => (
                <option key={s} value={s}>
                  {s === "ACTIVE"
                    ? t("common.active")
                    : s === "PENDING"
                      ? t("common.pending")
                      : s === "REVOKED"
                        ? t("common.revoked")
                        : t("sql-editor.expired")}
                </option>
              ))}
            </select>
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
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-control-light">
                    <th className="py-2 pr-4 font-medium w-24">
                      {t("common.status")}
                    </th>
                    <th className="py-2 pr-4 font-medium w-44">
                      {t("common.creator")}
                    </th>
                    <th className="py-2 pr-4 font-medium w-44 hidden xl:table-cell">
                      {t("common.created-at")}
                    </th>
                    <th className="py-2 pr-4 font-medium w-44 hidden xl:table-cell">
                      {t("common.expiration")}
                    </th>
                    <th className="py-2 pr-4 font-medium">
                      {t("common.statement")}
                    </th>
                    <th className="py-2 pr-4 font-medium w-48">
                      {t("common.databases")}
                    </th>
                    <th className="py-2 font-medium w-40" />
                  </tr>
                </thead>
                <tbody>
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
                </tbody>
              </table>

              <div className="mt-3">
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
      ) : (
        <div className="mx-4 mt-2 flex items-start gap-3 rounded-xs border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-700">
          <span>
            {t("sql-editor.access-grants-redirect-hint")
              .split("{link}")
              .map((part, i) =>
                i === 0 ? (
                  <span key={i}>{part}</span>
                ) : (
                  <span key={i}>
                    <a
                      className="normal-link"
                      href="#"
                      onClick={(e) => {
                        e.preventDefault();
                        router.push({
                          name: "sql-editor.project",
                          params: { project: projectId },
                          query: { panel: "access" },
                        });
                      }}
                    >
                      {t("sql-editor.self")}
                    </a>
                    {part}
                  </span>
                )
              )}
          </span>
        </div>
      )}

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
    <tr className="border-b">
      <td className="py-2 pr-4">
        <Badge variant={statusTagVariant(status)}>
          {getAccessGrantDisplayStatusText(grant)}
        </Badge>
      </td>
      <td className="py-2 pr-4 truncate max-w-44">
        {extractUserEmail(grant.creator)}
      </td>
      <td className="py-2 pr-4 text-sm hidden xl:table-cell">{createdAt}</td>
      <td className="py-2 pr-4 text-sm hidden xl:table-cell">{expiration}</td>
      <td className="py-2 pr-4">
        <Tooltip
          content={
            <pre className="max-w-lg whitespace-pre-wrap">{grant.query}</pre>
          }
        >
          <div className="flex items-center gap-x-1 overflow-hidden">
            <span className="font-mono text-xs truncate shrink">
              {grant.query}
            </span>
            {grant.unmask && (
              <Badge variant="warning" className="shrink-0">
                {t("sql-editor.grant-type-unmask")}
              </Badge>
            )}
          </div>
        </Tooltip>
      </td>
      <td className="py-2 pr-4">
        <DatabaseTargets targets={grant.targets} />
      </td>
      <td className="py-2">
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
      </td>
    </tr>
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
