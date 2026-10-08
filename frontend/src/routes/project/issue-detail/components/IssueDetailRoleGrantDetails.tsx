import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AccountTypeBadge } from "@/components/AccountTypeBadge";
import { HumanizeTs } from "@/components/HumanizeTs";
import { DirectExecutionCallout } from "@/components/role-grant/DirectExecutionCallout";
import { RoleDescription } from "@/components/role-grant/RoleDescription";
import { UserCell } from "@/components/UserCell";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useEnvironmentList, usePlanFeature } from "@/hooks/useAppState";
import { directExecutionOf } from "@/lib/project-member/directExecution";
import { displayRoleTitleFromList } from "@/lib/role";
import { useAppStore } from "@/stores/app";
import type { DatabaseResource } from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { unknownDatabase } from "@/types/v1/database";
import {
  type ConditionExpression,
  readableConditionFromCELString,
} from "@/utils/issue/cel";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { useIssueDetailContext } from "../context/IssueDetailContext";
import { usePrincipal } from "./usePrincipal";

export function IssueDetailRoleGrantDetails() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const issue = page.issue;
  const requestRoleName = issue?.roleGrant?.role ?? "";
  const granteeName = issue?.roleGrant?.user;
  const creatorName = issue?.creator;
  const roleList = useAppStore((state) => state.roleList);
  const requestRole = useAppStore((state) =>
    state.getRoleByName(requestRoleName)
  );
  const grantee = usePrincipal(granteeName);
  const showRequestedBy =
    creatorName !== undefined && creatorName !== granteeName;
  const creator = usePrincipal(showRequestedBy ? creatorName : undefined);
  const environmentList = useEnvironmentList();
  const hasEnvTierFeature = usePlanFeature(
    PlanFeature.FEATURE_ENVIRONMENT_TIERS
  );
  const [condition, setCondition] = useState<ConditionExpression | undefined>();

  useEffect(() => {
    // Clear synchronously so a prop change doesn't briefly show the
    // previous issue's environments while the new CEL expression parses.
    setCondition(undefined);

    const expression = issue?.roleGrant?.condition?.expression ?? "";
    if (!expression) return;

    let canceled = false;
    void (async () => {
      try {
        const parsed = await readableConditionFromCELString(expression);
        if (!canceled) setCondition(parsed);
      } catch (error) {
        // Unreadable reads as the widest grant, so the execution row never
        // disappears.
        console.error("Failed to parse CEL expression:", error);
        if (!canceled) setCondition({ databaseResources: [] });
      }
    })();
    return () => {
      canceled = true;
    };
  }, [issue?.roleGrant?.condition?.expression]);

  useEffect(() => {
    const resources = condition?.databaseResources ?? [];
    if (resources.length > 0) {
      void useAppStore
        .getState()
        .batchGetOrFetchDatabases(
          resources.map((resource) => resource.databaseFullName)
        );
    }
  }, [condition?.databaseResources]);

  const expression = issue?.roleGrant?.condition?.expression ?? "";
  // Hidden while the expression parses, so a request that resolves to an
  // explicit list is never shown as unscoped in between.
  const isParsing = expression !== "" && condition === undefined;
  const directExecution = isParsing
    ? undefined
    : directExecutionOf(requestRoleName, condition);
  const granteeTitle = grantee?.title || grantee?.email || granteeName || "";

  return (
    <div className="flex flex-col gap-y-4">
      <h3 className="text-base font-medium">{t("issue.role-grant.details")}</h3>

      <div className="flex flex-col gap-y-4 rounded-sm border p-4">
        {requestRoleName && (
          <div className="flex flex-col gap-y-2">
            <span className="text-sm text-control-light">{t("role.self")}</span>
            <div className="text-base">
              {displayRoleTitleFromList(requestRoleName, roleList)}
            </div>
            <RoleDescription role={requestRoleName} roleList={roleList} />
          </div>
        )}

        {granteeName && (
          <div
            className="flex flex-col gap-y-2"
            data-testid="role-grant-grantee"
          >
            <span className="text-sm text-control-light">
              {t("issue.role-grant.grantee")}
            </span>
            <UserCell
              title={granteeTitle}
              subtitle={grantee?.title ? grantee.email : undefined}
              showAvatar={false}
              badges={<AccountTypeBadge email={grantee?.email} />}
            />
            {showRequestedBy && (
              <p className="text-xs leading-4 text-control-light">
                {t("issue.role-grant.requested-by", {
                  creator: creator?.title || creator?.email || creatorName,
                })}
              </p>
            )}
          </div>
        )}

        {directExecution && (
          <div
            className="flex flex-col gap-y-2"
            data-testid="role-grant-direct-execution"
          >
            <span className="text-sm text-control-light">
              {t("project.members.direct-execution.title", {
                kind: directExecution.kind,
              })}
            </span>
            <DirectExecutionCallout
              kind={directExecution.kind}
              lead="approver"
              scope={directExecution.scope}
              grantee={granteeTitle}
              environmentList={environmentList}
              hasEnvTierFeature={hasEnvTierFeature}
            />
          </div>
        )}

        {condition?.databaseResources && (
          <div className="flex flex-col gap-y-2">
            <span className="text-sm text-control-light">
              {t("common.database")}
            </span>
            <div>
              {condition.databaseResources.length === 0 ? (
                <span className="text-base">
                  {t("issue.role-grant.all-databases")}
                </span>
              ) : (
                <IssueDetailDatabaseResourceTable
                  databaseResourceList={condition.databaseResources}
                />
              )}
            </div>
          </div>
        )}

        <div className="flex flex-col gap-y-2">
          <span className="text-sm text-control-light">
            {t("issue.role-grant.expired-at")}
          </span>
          <div className="text-base">
            {condition?.expiredTime ? (
              <HumanizeTs
                mode="operational"
                tsMs={new Date(condition.expiredTime).getTime()}
              />
            ) : (
              t("project.members.never-expires")
            )}
          </div>
        </div>

        {requestRole && (
          <div className="flex flex-col gap-y-2">
            <span className="text-sm text-control-light">
              {t("common.permissions")} ({requestRole.permissions.length})
            </span>
            <div className="max-h-[10em] overflow-auto rounded-sm border p-2">
              {requestRole.permissions.map((permission) => (
                <p key={permission} className="text-sm leading-5">
                  {permission}
                </p>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function IssueDetailDatabaseResourceTable({
  databaseResourceList,
}: {
  databaseResourceList: DatabaseResource[];
}) {
  const { t } = useTranslation();
  // Subscribe to the instance cache so rows reactively pick up titles once
  // instances hydrate; a bare getState() read would not re-render here.
  const instancesByName = useAppStore((s) => s.instancesByName);
  const databasesByName = useAppStore((s) => s.databasesByName);
  // Subscribe so titles refresh if the environment list changes.
  const environmentList = useAppStore((s) => s.environmentList);
  const rows = useMemo(
    () =>
      databaseResourceList.map((resource) => {
        const database =
          databasesByName[resource.databaseFullName] ?? unknownDatabase();
        const { databaseName, instance } = extractDatabaseResourceName(
          resource.databaseFullName
        );
        const instanceResource =
          instancesByName[instance] ?? database.instanceResource;
        const environmentName =
          database.effectiveEnvironment ??
          database.instanceResource?.environment ??
          "";
        const environment = useAppStore
          .getState()
          .getEnvironmentByName(environmentName);
        return {
          databaseName,
          environmentTitle: environment.title,
          instanceTitle: instanceResource?.title ?? "",
          resource,
        };
      }),
    [databaseResourceList, databasesByName, instancesByName, environmentList]
  );

  return (
    <div className="overflow-auto rounded-sm border">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead className="bg-control-bg/50">
              {t("common.database")}
            </TableHead>
            <TableHead className="bg-control-bg/50">
              {t("common.table")}
            </TableHead>
            <TableHead className="bg-control-bg/50">
              {t("common.environment")}
            </TableHead>
            <TableHead className="bg-control-bg/50">
              {t("common.instance")}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row, index) => (
            <TableRow key={`${row.resource.databaseFullName}-${index}`}>
              <TableCell>{row.databaseName}</TableCell>
              <TableCell>
                <span className="line-clamp-1">
                  {extractTableName(row.resource)}
                </span>
              </TableCell>
              <TableCell>{row.environmentTitle}</TableCell>
              <TableCell>{row.instanceTitle}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function extractTableName(databaseResource: DatabaseResource) {
  if (!databaseResource.schema && !databaseResource.table) {
    return "*";
  }
  const names = [];
  if (databaseResource.schema) {
    names.push(databaseResource.schema);
  }
  names.push(databaseResource.table || "*");
  return names.join(".");
}
