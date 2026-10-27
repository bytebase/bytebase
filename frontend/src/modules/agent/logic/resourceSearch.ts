import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseServiceClientConnect, issueServiceClientConnect } from "@/api";
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
import {
  type Issue,
  SearchIssuesRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
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

export async function searchAgentIssues(
  parent: string,
  query: string,
  pageToken: string,
  signal: AbortSignal
): Promise<{ items: Issue[]; nextPageToken: string }> {
  const tokens = getNameParentTokens(parent, [projectNamePrefix]);
  if (tokens.length !== 1) {
    throw new Error("A canonical project is required to search issues.");
  }
  signal.throwIfAborted();
  const response = await issueServiceClientConnect.searchIssues(
    create(SearchIssuesRequestSchema, {
      parent,
      pageSize: 20,
      pageToken,
      query: query.trim(),
      orderBy: "update_time desc",
    }),
    {
      signal,
      contextValues: createContextValues().set(silentContextKey, true),
    }
  );
  signal.throwIfAborted();
  const creators = [
    ...new Set(response.issues.map((issue) => issue.creator).filter(Boolean)),
  ];
  if (creators.length > 0) {
    try {
      await useAppStore.getState().batchGetOrFetchUsers(creators);
    } catch {
      // Missing creator details must not hide otherwise accessible issues.
    }
  }
  signal.throwIfAborted();
  return {
    items: response.issues,
    nextPageToken: response.nextPageToken ?? "",
  };
}
