import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { useAppStore } from "@/stores/app";
import {
  getNameParentTokens,
  projectNamePrefix,
  workspaceNamePrefix,
} from "@/stores/modules/v1/common";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  type Database,
  ListDatabasesRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { celString } from "@/utils/v1/celLiteral";

export async function searchAgentProjects(
  query: string,
  pageToken: string,
  signal: AbortSignal,
  includeDefault = false
): Promise<{ items: Project[]; nextPageToken: string }> {
  const { projects, nextPageToken } = await useAppStore
    .getState()
    .fetchProjectList({
      filter: {
        query: query.trim(),
        state: State.ACTIVE,
        excludeDefault: !includeDefault,
      },
      pageSize: 20,
      pageToken,
      signal,
      silent: true,
      orderBy: "title",
      cache: true,
    });
  return { items: projects, nextPageToken: nextPageToken ?? "" };
}

export async function searchAgentDatabases(
  parent: string,
  query: string,
  pageToken: string,
  signal: AbortSignal
): Promise<{ items: Database[]; nextPageToken: string }> {
  if (
    getNameParentTokens(parent, [projectNamePrefix]).length !== 1 &&
    getNameParentTokens(parent, [workspaceNamePrefix]).length !== 1
  ) {
    throw new Error(
      "A canonical project or workspace is required to search databases."
    );
  }
  signal.throwIfAborted();
  const keyword = query.trim().toLowerCase();
  const response = await databaseServiceClientConnect.listDatabases(
    create(ListDatabasesRequestSchema, {
      parent,
      pageSize: 20,
      pageToken,
      filter: keyword ? `name.contains(${celString(keyword)})` : "",
      orderBy: "name",
      showDeleted: false,
    }),
    {
      signal,
      contextValues: createContextValues().set(silentContextKey, true),
    }
  );
  signal.throwIfAborted();
  return {
    items: response.databases,
    nextPageToken: response.nextPageToken ?? "",
  };
}
