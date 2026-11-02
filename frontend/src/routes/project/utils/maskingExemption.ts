import type { TFunction } from "i18next";
import type { ExemptionGrant } from "@/lib/sensitive-data/types";
import type { DatabaseResource } from "@/types";
import { extractDatabaseResourceName } from "@/utils/v1/database";

export const resourceCoversDatabase = (
  resource: DatabaseResource,
  databaseFullName: string
): boolean => {
  const { instanceName, databaseName } =
    extractDatabaseResourceName(databaseFullName);
  if (resource.instanceResourceId || resource.databaseResourceId) {
    return (
      (!resource.instanceResourceId ||
        resource.instanceResourceId === instanceName) &&
      (!resource.databaseResourceId ||
        resource.databaseResourceId === databaseName)
    );
  }
  return (
    !resource.databaseFullName || resource.databaseFullName === databaseFullName
  );
};

export const generateMemberScopeSummary = (
  t: TFunction,
  grants: ExemptionGrant[],
  databaseNames: string[]
): string => {
  const grantCount = grants.length;
  const exemptionCount = t("project.masking-exemption.n-exemptions", {
    count: grantCount,
    n: grantCount,
  });
  const uniqueDatabases = [...new Set(databaseNames)];
  if (grants.some((grant) => grant.databaseResources === undefined)) {
    return `${exemptionCount} · ${t("cel.condition.self")}`;
  }
  if (uniqueDatabases.includes("")) {
    return `${exemptionCount} · ${t("database.all")}`;
  }
  if (uniqueDatabases.length === 1) {
    return `${exemptionCount} · ${uniqueDatabases[0]}`;
  }
  if (uniqueDatabases.length > 1) {
    const databaseCount = uniqueDatabases.length;
    return `${exemptionCount} · ${t("project.masking-exemption.n-databases", {
      count: databaseCount,
      n: databaseCount,
    })}`;
  }
  return exemptionCount;
};
