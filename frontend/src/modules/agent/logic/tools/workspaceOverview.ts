import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import {
  databaseServiceClientConnect,
  instanceServiceClientConnect,
  projectServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import { SearchDatabasesRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { ListInstancesRequestSchema } from "@/types/proto-es/v1/instance_service_pb";
import { SearchProjectsRequestSchema } from "@/types/proto-es/v1/project_service_pb";
import type { AgentEvidence, AgentEvidenceReference } from "../types";
import { createWorkspaceAttentionTool } from "./workspaceAttention";

const PAGE_SIZE = 3;

type SetupFacet =
  | {
      status: "available";
      partial: boolean;
      count: number;
    }
  | { status: "unavailable" };

type WorkspaceAttentionIssue = {
  name: string;
  title: string;
  relation: "review" | "created";
};

type AttentionFacet =
  | {
      status: "available";
      partial: boolean;
      issues: WorkspaceAttentionIssue[];
    }
  | { status: "unavailable" };

export type WorkspaceOverview = {
  workspace: string;
  setup: {
    projects: SetupFacet;
    instances: SetupFacet;
    databases: SetupFacet;
  };
  attention: AttentionFacet;
  evidence: AgentEvidence;
};

function isWorkspaceName(value: string | undefined): value is string {
  return !!value && /^workspaces\/[^/]+$/.test(value);
}

function unavailableSetupFacet(): SetupFacet {
  return { status: "unavailable" };
}

function parseAttention(result: string): AttentionFacet {
  try {
    const payload = JSON.parse(result) as Record<string, unknown>;
    if (typeof payload.error === "string" || !Array.isArray(payload.issues)) {
      return { status: "unavailable" };
    }

    const issues = payload.issues.flatMap((issue) => {
      if (!issue || typeof issue !== "object") return [];
      const candidate = issue as Record<string, unknown>;
      if (
        typeof candidate.name !== "string" ||
        typeof candidate.title !== "string" ||
        (candidate.relation !== "review" && candidate.relation !== "created")
      ) {
        return [];
      }
      return [
        {
          name: candidate.name,
          title: candidate.title,
          relation: candidate.relation,
        } as WorkspaceAttentionIssue,
      ];
    });

    return {
      status: "available",
      partial: payload.partial === true,
      issues,
    };
  } catch {
    return { status: "unavailable" };
  }
}

function collectEvidence(
  workspace: string,
  attention: AttentionFacet
): AgentEvidence {
  const references: AgentEvidenceReference[] = [];
  if (attention.status === "available") {
    for (const issue of attention.issues) {
      references.push({
        kind: "issue",
        name: issue.name,
        title: issue.title,
      });
    }
  }
  return { workspace, references };
}

export async function collectWorkspaceOverview(
  workspaceName: string | undefined
): Promise<WorkspaceOverview | null> {
  if (!isWorkspaceName(workspaceName)) {
    return null;
  }

  const silentOptions = {
    contextValues: createContextValues().set(silentContextKey, true),
  };
  const results = await Promise.allSettled([
    createWorkspaceAttentionTool({ workspaceName })(),
    projectServiceClientConnect.searchProjects(
      create(SearchProjectsRequestSchema, {
        filter: 'exclude_default == true && state == "ACTIVE"',
        pageSize: PAGE_SIZE,
        orderBy: "title",
        showDeleted: false,
      }),
      silentOptions
    ),
    instanceServiceClientConnect.listInstances(
      create(ListInstancesRequestSchema, {
        pageSize: PAGE_SIZE,
        showDeleted: false,
      }),
      silentOptions
    ),
    databaseServiceClientConnect.searchDatabases(
      create(SearchDatabasesRequestSchema, { pageSize: PAGE_SIZE }),
      silentOptions
    ),
  ]);

  const projectInstanceResults =
    results[1].status === "fulfilled"
      ? await Promise.allSettled(
          results[1].value.projects.map((project) =>
            instanceServiceClientConnect.listInstances(
              create(ListInstancesRequestSchema, {
                parent: project.name,
                pageSize: PAGE_SIZE,
                showDeleted: false,
              }),
              silentOptions
            )
          )
        )
      : [];
  const instanceResults = [results[2], ...projectInstanceResults];
  const availableInstanceResults = instanceResults.filter(
    (result) => result.status === "fulfilled"
  );

  const attention =
    results[0].status === "fulfilled"
      ? parseAttention(results[0].value)
      : { status: "unavailable" as const };
  const setup = {
    projects:
      results[1].status === "fulfilled"
        ? {
            status: "available" as const,
            partial: Boolean(results[1].value.nextPageToken),
            count: results[1].value.projects.length,
          }
        : unavailableSetupFacet(),
    instances:
      availableInstanceResults.length > 0
        ? {
            status: "available" as const,
            partial:
              results[1].status !== "fulfilled" ||
              Boolean(results[1].value.nextPageToken) ||
              instanceResults.some(
                (result) =>
                  result.status === "rejected" ||
                  Boolean(result.value.nextPageToken)
              ),
            count: availableInstanceResults.reduce(
              (count, result) => count + result.value.instances.length,
              0
            ),
          }
        : unavailableSetupFacet(),
    databases:
      results[3].status === "fulfilled"
        ? {
            status: "available" as const,
            partial: Boolean(results[3].value.nextPageToken),
            count: results[3].value.databases.length,
          }
        : unavailableSetupFacet(),
  };

  return {
    workspace: workspaceName,
    setup,
    attention,
    evidence: collectEvidence(workspaceName, attention),
  };
}

export function extractWorkspaceOverviewEvidence(
  result: string
): AgentEvidence | undefined {
  try {
    const payload = JSON.parse(result) as { evidence?: unknown };
    const evidence = payload.evidence;
    if (!evidence || typeof evidence !== "object") return undefined;

    const candidate = evidence as {
      workspace?: unknown;
      references?: unknown;
    };
    if (
      typeof candidate.workspace !== "string" ||
      !isWorkspaceName(candidate.workspace) ||
      !Array.isArray(candidate.references)
    ) {
      return undefined;
    }

    const references: AgentEvidenceReference[] = [];
    for (const reference of candidate.references) {
      if (!reference || typeof reference !== "object") continue;
      const value = reference as Record<string, unknown>;
      if (
        value.kind === "issue" &&
        typeof value.name === "string" &&
        typeof value.title === "string"
      ) {
        references.push({
          kind: "issue",
          name: value.name,
          title: value.title,
        });
      }
    }
    return { workspace: candidate.workspace, references };
  } catch {
    return undefined;
  }
}

export function createWorkspaceOverviewTool({
  workspaceName,
}: {
  workspaceName?: string;
}) {
  return async (): Promise<string> => {
    const overview = await collectWorkspaceOverview(workspaceName);
    return JSON.stringify(
      overview ?? { error: "This chat is not bound to a workspace." }
    );
  };
}
