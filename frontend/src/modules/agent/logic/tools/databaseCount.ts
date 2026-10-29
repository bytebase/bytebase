import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import {
  databaseNamePrefix,
  getNameParentTokens,
  instanceNamePrefix,
  projectNamePrefix,
  workspaceNamePrefix,
} from "@/stores/modules/v1/common";
import { SearchDatabasesRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { getDefaultPagination } from "@/utils/pagination";
import { celString } from "@/utils/v1/celLiteral";

const isDatabaseCollectionParent = (name: string | undefined): name is string =>
  !!name &&
  (getNameParentTokens(name, [workspaceNamePrefix]).length === 1 ||
    getNameParentTokens(name, [projectNamePrefix]).length === 1);

const getReferencedInstanceName = (
  databaseName: string | undefined,
  projectName: string | undefined
): string | undefined => {
  if (
    !projectName ||
    getNameParentTokens(projectName, [projectNamePrefix]).length !== 1
  ) {
    return undefined;
  }

  const projectTokens = getNameParentTokens(databaseName ?? "", [
    projectNamePrefix,
    instanceNamePrefix,
    databaseNamePrefix,
  ]);
  if (
    projectTokens.length === 3 &&
    `projects/${projectTokens[0]}` === projectName
  ) {
    return `projects/${projectTokens[0]}/instances/${projectTokens[1]}`;
  }

  const workspaceTokens = getNameParentTokens(databaseName ?? "", [
    instanceNamePrefix,
    databaseNamePrefix,
  ]);
  return workspaceTokens.length === 2
    ? `instances/${workspaceTokens[0]}`
    : undefined;
};

export function createDatabaseCountTool({
  scopeName,
  referencedProject,
  referencedDatabase,
}: {
  scopeName?: string;
  referencedProject?: string;
  referencedDatabase?: string;
}) {
  return async (args: Record<string, unknown> = {}): Promise<string> => {
    if (Object.keys(args).some((key) => key !== "scope")) {
      return JSON.stringify({
        error:
          "Only the current chat or referenced database instance can be counted.",
      });
    }
    const scope = args.scope ?? "context";
    const parent =
      scope === "context"
        ? scopeName
        : scope === "referenced-instance"
          ? getReferencedInstanceName(referencedDatabase, referencedProject)
          : undefined;
    if (
      !parent ||
      (scope === "context" && !isDatabaseCollectionParent(parent))
    ) {
      return JSON.stringify({
        error:
          scope === "referenced-instance"
            ? "This turn has no selected database with an accessible instance."
            : "This chat is not bound to a workspace or project.",
      });
    }

    try {
      const options = {
        contextValues: createContextValues().set(silentContextKey, true),
      };
      const filter =
        scope === "referenced-instance"
          ? `instance == ${celString(parent)}`
          : parent.startsWith(projectNamePrefix)
            ? `project == ${celString(parent)}`
            : "";
      const page = await databaseServiceClientConnect.searchDatabases(
        create(SearchDatabasesRequestSchema, {
          pageSize: getDefaultPagination(),
          filter,
        }),
        options
      );
      return JSON.stringify({
        scope: parent,
        status: "available",
        count: page.databases.length,
        partial: Boolean(page.nextPageToken),
      });
    } catch {
      return JSON.stringify({ scope: parent, status: "unavailable" });
    }
  };
}
