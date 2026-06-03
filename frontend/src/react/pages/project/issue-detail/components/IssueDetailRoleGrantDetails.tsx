import dayjs from "dayjs";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
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
import { getRoleEnvironmentLimitationKind } from "@/react/lib/project-member/utils";
import { displayRoleTitleFromList } from "@/react/lib/role";
import { useAppStore } from "@/react/stores/app";
import type { DatabaseResource } from "@/types";
import { unknownDatabase } from "@/types/v1/database";
import {
  type ConditionExpression,
  convertFromCELString,
} from "@/utils/issue/cel";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { useIssueDetailContext } from "../context/IssueDetailContext";

export function IssueDetailRoleGrantDetails() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const issue = page.issue;
  const requestRoleName = issue?.roleGrant?.role ?? "";
  const roleList = useAppStore((state) => state.roleList);
  const requestRole = useAppStore((state) =>
    state.getRoleByName(requestRoleName)
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
        const parsed = await convertFromCELString(expression);
        if (!canceled) setCondition(parsed);
      } catch (error) {
        console.error("Failed to parse CEL expression:", error);
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

  const envKind = getRoleEnvironmentLimitationKind(requestRoleName);
  const envNames = condition?.environments ?? [];
  const envList = useEnvironmentList();
  // Falls back to the raw env resource name (e.g. environments/prod-old) when
  // the env isn't in the store — happens when an env is renamed or deleted
  // between request submission and approver review.
  const envTitles = envNames.map(
    (n) => envList.find((e) => e.name === n)?.title ?? n
  );

  // Three-way env scope:
  //   environments === undefined  → no env clause in CEL → unrestricted (binding-all)
  //   environments === []         → restricted to empty list (binding-none)
  //   environments === [list]     → restricted to listed envs (binding-some)
  // Hide during async parse so we don't briefly show binding-all for an
  // expression that's about to resolve to binding-some/binding-none.
  const expression = issue?.roleGrant?.condition?.expression ?? "";
  const isParsing = expression !== "" && condition === undefined;
  const envScope = computeEnvScope(envKind, isParsing, condition?.environments);

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

        {envScope === "binding-all" && envKind && (
          <DDLWarningCallout type="binding-all" kind={envKind} />
        )}
        {/*
         * binding-none on the issue page = the request specified an empty
         * env list (degenerate: the binding would grant no env access at
         * all). Showing an info box would suggest there's something to
         * approve here when really the binding grants nothing. Hide.
         */}
        {envScope === "binding-some" && envKind && (
          <div className="flex flex-col gap-y-2">
            <span className="text-sm text-control-light">
              {t("common.environments")}
            </span>
            <DDLWarningCallout type="binding-some" kind={envKind} />
            <div className="text-base">{envTitles.join(", ")}</div>
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
  // Subscribe to the instance cache so rows reactively pick up titles once
  // instances hydrate; a bare getState() read would not re-render here because
  // useVueState only tracks Vue dependencies.
  const instancesByName = useAppStore((s) => s.instancesByName);
  const databasesByName = useAppStore((s) => s.databasesByName);
  // Subscribe so titles refresh if the environment list changes.
  void useAppStore((s) => s.environmentList);
  const rows = useVueState(() =>
    databaseResourceList.map((resource) => {
      const database =
        databasesByName[resource.databaseFullName] ?? unknownDatabase();
      const { databaseName, instanceName } = extractDatabaseResourceName(
        resource.databaseFullName
      );
      const instance = instanceName
        ? instancesByName[`instances/${instanceName}`]
        : database.instanceResource;
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

function computeEnvScope(
  envKind: ReturnType<typeof getRoleEnvironmentLimitationKind>,
  isParsing: boolean,
  environments: string[] | undefined
): "binding-all" | "binding-some" | "binding-none" | undefined {
  if (!envKind || isParsing) return undefined;
  if (environments === undefined) return "binding-all";
  if (environments.length === 0) return "binding-none";
  return "binding-some";
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
