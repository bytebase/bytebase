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
import {
  ListDatabasesRequestSchema,
  SearchDatabasesRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { celString } from "@/utils/v1/celLiteral";

const PAGE_SIZE = 1000;

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

    let count = 0;
    let pageToken = "";
    const seenPageTokens = new Set<string>();
    try {
      do {
        const options = {
          contextValues: createContextValues().set(silentContextKey, true),
        };
        const page =
          scope === "context"
            ? await databaseServiceClientConnect.searchDatabases(
                create(SearchDatabasesRequestSchema, {
                  pageSize: PAGE_SIZE,
                  pageToken,
                  filter: parent.startsWith(projectNamePrefix)
                    ? `project == ${celString(parent)}`
                    : "",
                }),
                options
              )
            : await databaseServiceClientConnect.listDatabases(
                create(ListDatabasesRequestSchema, {
                  parent,
                  pageSize: PAGE_SIZE,
                  pageToken,
                  showDeleted: false,
                }),
                options
              );
        count += page.databases.length;
        pageToken = page.nextPageToken;
        if (pageToken && seenPageTokens.has(pageToken)) {
          return JSON.stringify({ scope: parent, status: "unavailable" });
        }
        seenPageTokens.add(pageToken);
      } while (pageToken);
      return JSON.stringify({ scope: parent, status: "available", count });
    } catch {
      return JSON.stringify({ scope: parent, status: "unavailable" });
    }
  };
}
