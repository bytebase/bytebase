import { create } from "@bufbuild/protobuf";
import { KeyRound, Plus } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import githubIcon from "@/assets/git/github.svg";
import gitlabIcon from "@/assets/git/gitlab.svg";
import { PermissionGuard } from "@/components/PermissionGuard";
import {
  ProjectPageLayout,
  ProjectPageToolbar,
} from "@/components/ProjectPageLayout";
import { Button } from "@/components/ui/button";
import { EllipsisText } from "@/components/ui/ellipsis-text";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tooltip } from "@/components/ui/tooltip";
import { WorkloadIdentitySheet } from "@/components/WorkloadIdentityFlow/WorkloadIdentitySheet";
import {
  WorkspacePageLayout,
  WorkspacePageToolbar,
} from "@/components/WorkspacePageLayout";
import { PagedTableFooter, usePagedData } from "@/hooks/usePagedData";
import { useProjectByName } from "@/hooks/useProjectByName";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { ensureWorkloadIdentityFullName } from "@/stores/app/workloadIdentity";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import { getWorkloadIdentityNameInBinding } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  type WorkloadIdentity,
  WorkloadIdentitySchema,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";
import {
  AccountCreatedCell,
  AccountEmptyRow,
  AccountListToolbar,
  type AccountMenuItem,
  AccountNameCell,
  AccountPageInfo,
  AccountRoleCell,
  AccountRowMenu,
  AccountSkeletonRows,
  type AccountStateFilter,
  accountListRequest,
  accountMatchesList,
  DeactivateAccountDialog,
  useAccountRoles,
} from "./MachineAccountList";
import {
  type WorkloadIdentityTrust,
  workloadIdentityTrust,
} from "./workloadIdentityTrust";

// ============================================================
// TrustCell
// ============================================================

const capitalize = (text: string) =>
  text.charAt(0).toLocaleUpperCase() + text.slice(1);

function ProviderIcon({
  provider,
}: {
  readonly provider: WorkloadIdentityTrust["provider"];
}) {
  const { t } = useTranslation();
  const title =
    provider === "github"
      ? t("workload-identity.token-source.github-title")
      : provider === "gitlab"
        ? t("workload-identity.token-source.gitlab-title")
        : t("workload-identity.token-source.oidc-title");
  return (
    <Tooltip content={title}>
      <span
        role="img"
        aria-label={title}
        className="flex h-5 w-4 shrink-0 items-center justify-center"
      >
        {provider === "oidc" ? (
          <KeyRound className="size-4 text-control-light" />
        ) : (
          <img
            src={provider === "github" ? githubIcon : gitlabIcon}
            alt=""
            className="size-4"
          />
        )}
      </span>
    </Tooltip>
  );
}

// Restates what the create form asked, in the form's own words.
function TrustCell({ trust }: { readonly trust: WorkloadIdentityTrust }) {
  const { t } = useTranslation();
  const { primary, secondary } = trust;
  const primaryText =
    primary.kind === "any-repository"
      ? capitalize(
          trust.provider === "gitlab"
            ? t("workload-identity.done.target-any-project", {
                owner: primary.owner,
              })
            : t("workload-identity.done.target-any-repository", {
                owner: primary.owner,
              })
        )
      : primary.value;

  let secondaryLine: React.ReactNode = null;
  if (secondary?.kind === "environment") {
    secondaryLine = (
      <EllipsisText
        text={t("account-list.trust-environment-plain", {
          environment: secondary.value,
        })}
        className="text-xs text-control-light"
      >
        <Trans
          i18nKey="account-list.trust-environment"
          values={{ environment: secondary.value }}
          components={{ code: <code className="font-mono" /> }}
        />
      </EllipsisText>
    );
  } else if (secondary?.kind === "any-environment") {
    secondaryLine = (
      <EllipsisText
        text={t("account-list.trust-any-environment")}
        className="text-xs text-control-light"
      />
    );
  } else if (secondary?.kind === "pattern") {
    secondaryLine = (
      <EllipsisText
        text={secondary.value}
        className="font-mono text-xs text-control"
      />
    );
  }

  return (
    <div className="flex max-w-75 items-start gap-x-2">
      <ProviderIcon provider={trust.provider} />
      <div className="flex min-w-0 flex-col">
        <EllipsisText
          text={primaryText}
          className={
            primary.kind === "pattern"
              ? "font-mono text-xs leading-5 text-main"
              : "text-main"
          }
        />
        {secondaryLine}
      </div>
    </div>
  );
}

// ============================================================
// WorkloadIdentityTable
// ============================================================

const COLUMNS = 5;

function WorkloadIdentityTable({
  workloadIdentities,
  loading,
  project,
  projectName,
  filtered,
  onClearFilters,
  onUpdated,
  onSelected,
}: {
  workloadIdentities: WorkloadIdentity[];
  loading: boolean;
  project?: Project;
  projectName?: string;
  filtered: boolean;
  onClearFilters: () => void;
  onUpdated: (wi: WorkloadIdentity) => void;
  onSelected: (wi: WorkloadIdentity) => void;
}) {
  const { t } = useTranslation();
  const deleteWorkloadIdentity = useAppStore(
    (state) => state.deleteWorkloadIdentity
  );
  const undeleteWorkloadIdentity = useAppStore(
    (state) => state.undeleteWorkloadIdentity
  );
  const rolesOf = useAccountRoles(projectName);

  const can = (permission: Parameters<typeof hasWorkspacePermissionV2>[0]) =>
    project
      ? hasProjectPermissionV2(project, permission)
      : hasWorkspacePermissionV2(permission);
  const canOpenDetail = can("bb.workloadIdentities.get");
  const canDelete = can("bb.workloadIdentities.delete");
  const canRestore = can("bb.workloadIdentities.undelete");

  const [deactivating, setDeactivating] = useState<WorkloadIdentity>();
  const [isDeactivating, setIsDeactivating] = useState(false);
  const handleDeactivate = async () => {
    if (!deactivating) return;
    setIsDeactivating(true);
    try {
      await deleteWorkloadIdentity(
        ensureWorkloadIdentityFullName(deactivating.email)
      );
      onUpdated(
        create(WorkloadIdentitySchema, {
          ...deactivating,
          state: State.DELETED,
        })
      );
      setDeactivating(undefined);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // error already shown by store
    } finally {
      setIsDeactivating(false);
    }
  };

  const handleRestore = async (wi: WorkloadIdentity) => {
    try {
      await undeleteWorkloadIdentity(ensureWorkloadIdentityFullName(wi.email));
      onUpdated(create(WorkloadIdentitySchema, { ...wi, state: State.ACTIVE }));
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // error already shown by store
    }
  };

  const menuItems = (wi: WorkloadIdentity): AccountMenuItem[] => {
    if (wi.state === State.DELETED) {
      return canRestore
        ? [
            {
              key: "restore",
              label: t("common.restore"),
              onSelect: () => void handleRestore(wi),
            },
          ]
        : [];
    }
    const items: AccountMenuItem[] = [];
    if (canOpenDetail) {
      items.push({
        key: "edit",
        label: t("common.edit"),
        onSelect: () => onSelected(wi),
      });
    }
    if (canDelete) {
      items.push({
        key: "deactivate",
        label: t("common.deactivate"),
        destructive: true,
        onSelect: () => setDeactivating(wi),
      });
    }
    return items;
  };

  return (
    <div className="overflow-x-auto rounded-sm border">
      <Table>
        <TableHeader>
          <TableRow className="bg-control-bg">
            <TableHead className="min-w-50 whitespace-nowrap">
              {t("common.name")}
            </TableHead>
            <TableHead className="whitespace-nowrap">
              {t("account-list.trusts")}
            </TableHead>
            <TableHead className="whitespace-nowrap">
              {t("settings.members.table.roles")}
            </TableHead>
            <TableHead className="whitespace-nowrap">
              {t("common.created")}
            </TableHead>
            <TableHead>
              <span className="sr-only">{t("common.actions")}</span>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {loading ? (
            <AccountSkeletonRows columns={COLUMNS} />
          ) : workloadIdentities.length === 0 ? (
            <AccountEmptyRow
              colSpan={COLUMNS}
              filtered={filtered}
              emptyText={t("account-list.no-workload-identities")}
              onClearFilters={onClearFilters}
            />
          ) : (
            workloadIdentities.map((wi) => (
              <TableRow key={wi.name}>
                <TableCell className="w-full max-w-0">
                  <AccountNameCell
                    title={wi.title}
                    email={wi.email}
                    inactive={wi.state === State.DELETED}
                    onOpen={
                      canOpenDetail && wi.state !== State.DELETED
                        ? () => onSelected(wi)
                        : undefined
                    }
                  />
                </TableCell>
                <TableCell>
                  <TrustCell
                    trust={workloadIdentityTrust(wi.workloadIdentityConfig)}
                  />
                </TableCell>
                <TableCell>
                  <AccountRoleCell
                    roles={rolesOf(getWorkloadIdentityNameInBinding(wi.email))}
                  />
                </TableCell>
                <TableCell>
                  <AccountCreatedCell createTime={wi.createTime} />
                </TableCell>
                <TableCell>
                  <div className="flex justify-end">
                    <AccountRowMenu
                      label={wi.title || wi.email}
                      items={menuItems(wi)}
                    />
                  </div>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      <DeactivateAccountDialog
        account={deactivating}
        pending={isDeactivating}
        onCancel={() => setDeactivating(undefined)}
        onConfirm={() => void handleDeactivate()}
      />
    </div>
  );
}

// ============================================================
// WorkloadIdentitiesPage (main)
// ============================================================

export function WorkloadIdentitiesPage({ projectId }: { projectId?: string }) {
  const { t } = useTranslation();
  const workspaceResourceName = useAppStore((s) => s.workspaceResourceName());
  const projectsByName = useAppStore((s) => s.projectsByName);
  const listWorkloadIdentities = useAppStore(
    (state) => state.listWorkloadIdentities
  );
  const fetchWorkloadIdentity = useAppStore(
    (state) => state.fetchWorkloadIdentity
  );

  const projectName = projectId
    ? `${projectNamePrefix}${projectId}`
    : undefined;
  // subscribe to re-render on project cache change
  void projectsByName;
  const projectFromName = useProjectByName(projectName ?? "");
  const project = projectName ? projectFromName : undefined;

  const parent = projectName ?? workspaceResourceName;

  const [query, setQuery] = useState("");
  const [stateFilter, setStateFilter] = useState<AccountStateFilter>("active");
  const [showDrawer, setShowDrawer] = useState(false);
  const [editingWI, setEditingWI] = useState<WorkloadIdentity | undefined>(
    undefined
  );

  const fetchList = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const response = await listWorkloadIdentities({
        parent,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        ...accountListRequest(stateFilter, query),
      });
      return {
        list: response.workloadIdentities,
        nextPageToken: response.nextPageToken,
      };
    },
    [listWorkloadIdentities, parent, stateFilter, query]
  );

  const data = usePagedData<WorkloadIdentity>({
    sessionKey: `bb.workload-identities${projectName ? `.${projectName}` : ""}.page-size`,
    fetchList,
  });

  // Keeps a row only while it matches the filters. Removing the last loaded
  // row refetches, so remaining pages stay reachable.
  const handleUpdated = (wi: WorkloadIdentity) => {
    if (accountMatchesList(wi, stateFilter, query)) {
      data.updateCache([wi]);
      return;
    }
    data.removeCache(wi);
    if (data.hasMore && data.dataList.every((row) => row.name === wi.name)) {
      data.refresh();
    }
  };

  // Only the latest click may open the sheet, so a slow fetch for an earlier
  // row cannot replace a later one.
  const selectSeqRef = useRef(0);
  const handleSelected = async (wi: WorkloadIdentity) => {
    const seq = ++selectSeqRef.current;
    const full = await fetchWorkloadIdentity(wi.email, true);
    if (seq !== selectSeqRef.current) return;
    setEditingWI(full);
    setShowDrawer(true);
  };

  const canCreate = project
    ? project.state === State.ACTIVE &&
      hasProjectPermissionV2(project, "bb.workloadIdentities.create")
    : hasWorkspacePermissionV2("bb.workloadIdentities.create");
  const openCreate = () => {
    setEditingWI(undefined);
    setShowDrawer(true);
  };
  const filtered = query.trim() !== "" || stateFilter !== "active";

  const PageLayout = projectName ? ProjectPageLayout : WorkspacePageLayout;
  const PageToolbar = projectName ? ProjectPageToolbar : WorkspacePageToolbar;

  return (
    <PageLayout>
      <AccountPageInfo
        isProject={!!projectName}
        description={t("account-list.workload-identities-description")}
      />

      <PageToolbar>
        <AccountListToolbar
          query={query}
          onQueryChange={setQuery}
          state={stateFilter}
          onStateChange={setStateFilter}
          action={
            <PermissionGuard
              permissions={["bb.workloadIdentities.create"]}
              project={project}
            >
              <Button disabled={!canCreate} onClick={openCreate}>
                <Plus className="size-4" />
                {t("common.create")}
              </Button>
            </PermissionGuard>
          }
        />
      </PageToolbar>

      <div className="flex flex-col gap-y-4">
        <WorkloadIdentityTable
          workloadIdentities={data.dataList}
          loading={data.isLoading && data.dataList.length === 0}
          project={project}
          projectName={projectName}
          filtered={filtered}
          onClearFilters={() => {
            setQuery("");
            setStateFilter("active");
          }}
          onUpdated={handleUpdated}
          onSelected={(wi) => void handleSelected(wi)}
        />
        {(data.dataList.length > 0 || data.hasMore) && (
          <PagedTableFooter
            pageSize={data.pageSize}
            pageSizeOptions={data.pageSizeOptions}
            onPageSizeChange={data.onPageSizeChange}
            hasMore={data.hasMore}
            isFetchingMore={data.isFetchingMore}
            onLoadMore={data.loadMore}
          />
        )}
      </div>

      <WorkloadIdentitySheet
        open={showDrawer}
        workloadIdentity={editingWI}
        project={projectName}
        onClose={() => {
          setShowDrawer(false);
          setEditingWI(undefined);
        }}
        onCreated={handleUpdated}
        onUpdated={handleUpdated}
      />
    </PageLayout>
  );
}
