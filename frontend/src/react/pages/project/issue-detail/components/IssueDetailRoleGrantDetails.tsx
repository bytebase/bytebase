import dayjs from "dayjs";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getRoleEnvironmentLimitationKind } from "@/components/ProjectMember/utils";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { DDLWarningCallout } from "@/react/components/role-grant/DDLWarningCallout";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { useEnvironmentList } from "@/react/hooks/useAppState";
import { useVueState } from "@/react/hooks/useVueState";
import {
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useRoleStore,
} from "@/store";
import type { DatabaseResource } from "@/types";
import { displayRoleTitle } from "@/utils";
import {
  type ConditionExpression,
  convertFromCELString,
} from "@/utils/issue/cel";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { useIssueDetailContext } from "../context/IssueDetailContext";

export function IssueDetailRoleGrantDetails() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const roleStore = useRoleStore();
  const databaseStore = useDatabaseV1Store();
  const issue = page.issue;
  const requestRoleName = issue?.roleGrant?.role ?? "";
  const requestRole = useVueState(() =>
    roleStore.getRoleByName(requestRoleName)
  );
  const [condition, setCondition] = useState<ConditionExpression | undefined>();

  useEffect(() => {
    let canceled = false;

    const run = async () => {
      const expression = issue?.roleGrant?.condition?.expression ?? "";
      if (!expression) {
        setCondition(undefined);
        return;
      }
      try {
        const parsed = await convertFromCELString(expression);
        if (!canceled) {
          setCondition(parsed);
        }
      } catch (error) {
        console.error("Failed to parse CEL expression:", error);
        if (!canceled) {
          setCondition(undefined);
        }
      }
    };

    void run();
    return () => {
      canceled = true;
    };
  }, [issue?.roleGrant?.condition?.expression]);

  useEffect(() => {
    const resources = condition?.databaseResources ?? [];
    if (resources.length > 0) {
      void databaseStore.batchGetOrFetchDatabases(
        resources.map((resource) => resource.databaseFullName)
      );
    }
  }, [condition?.databaseResources, databaseStore]);

  const envKind = getRoleEnvironmentLimitationKind(requestRoleName);
  const envNames = condition?.environments ?? [];
  const envList = useEnvironmentList();
  const envTitles = useMemo(() => {
    const names = condition?.environments ?? [];
    const byName = new Map(envList.map((e) => [e.name, e.title]));
    // Falls back to the raw env resource name (e.g. environments/prod-old) if the
    // env isn't in the store, which can happen if the env was renamed or deleted
    // between request submission and approver review.
    return names.map((n) => byName.get(n) ?? n);
  }, [condition?.environments, envList]);

  return (
    <div className="flex flex-col gap-y-4">
      <h3 className="text-base font-medium">{t("issue.role-grant.details")}</h3>

      <div className="flex flex-col gap-y-4 rounded-sm border p-4">
        {envKind && envNames.length > 0 && (
          <DDLWarningCallout
            type="issue"
            kind={envKind}
            environments={envTitles}
          />
        )}

        {requestRoleName && (
          <div className="flex flex-col gap-y-2">
            <span className="text-sm text-control-light">{t("role.self")}</span>
            <div className="text-base">{displayRoleTitle(requestRoleName)}</div>
          </div>
        )}

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

        {envNames.length > 0 && (
          <div className="flex flex-col gap-y-2">
            <span className="text-sm text-control-light">
              {t("common.environments")}
            </span>
            <div className="flex flex-wrap gap-1">
              {envNames.map((env) => (
                <EnvironmentLabel
                  key={env}
                  environmentName={env}
                  className="text-xs"
                />
              ))}
            </div>
          </div>
        )}

        {condition?.databaseResources && (
          <div className="flex flex-col gap-y-2">
            <span className="text-sm text-control-light">
              {t("common.database")}
            </span>
            <div>
              {condition.databaseResources.length === 0 ? (
                <span>{t("issue.role-grant.all-databases")}</span>
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
            {condition?.expiredTime
              ? dayjs(new Date(condition.expiredTime)).format("LLL")
              : t("project.members.never-expires")}
          </div>
        </div>
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
  const databaseStore = useDatabaseV1Store();
  const environmentStore = useEnvironmentV1Store();
  const instanceStore = useInstanceV1Store();
  const rows = useVueState(() =>
    databaseResourceList.map((resource) => {
      const database = databaseStore.getDatabaseByName(
        resource.databaseFullName
      );
      const { databaseName, instanceName } = extractDatabaseResourceName(
        resource.databaseFullName
      );
      const instance = instanceName
        ? instanceStore.getInstanceByName(`instances/${instanceName}`)
        : database.instanceResource;
      const environmentName =
        database.effectiveEnvironment ??
        database.instanceResource?.environment ??
        "";
      const environment =
        environmentStore.getEnvironmentByName(environmentName);
      return {
        databaseName,
        environmentTitle: environment.title,
        instanceTitle: instance?.title ?? "",
        resource,
      };
    })
  );

  return (
    <div className="overflow-auto rounded-sm border">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead className="bg-gray-50">{t("common.database")}</TableHead>
            <TableHead className="bg-gray-50">{t("common.table")}</TableHead>
            <TableHead className="bg-gray-50">
              {t("common.environment")}
            </TableHead>
            <TableHead className="bg-gray-50">{t("common.instance")}</TableHead>
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
