import { useAppStore } from "@/stores/app";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type { AgentEvidence, AgentEvidenceReference } from "../types";
import { createProjectOpenIssuesTool } from "./projectOpenIssues";

const DATABASE_PAGE_SIZE = 3;
const DATABASE_CHANGE_PAGE_SIZE = 5;

type EvidenceFacet<Key extends string, Value> =
  | ({
      status: "available";
      partial: boolean;
    } & Record<Key, Value>)
  | {
      status: "unavailable";
    };

type ProjectIssue = {
  name: string;
  title: string;
  relation: "review" | "created";
  approvalStatus: string;
};

type DatabaseChange = {
  name: string;
  title: string;
};

export type ProjectOverview = {
  project: string;
  personalIssues: EvidenceFacet<"issues", ProjectIssue[]>;
  databases: EvidenceFacet<"databases", { name: string }[]>;
  openDatabaseChanges: EvidenceFacet<"issues", DatabaseChange[]>;
  evidence: AgentEvidence;
};

function isProjectName(value: string | undefined): value is string {
  return !!value && /^projects\/[^/]+$/.test(value);
}

function unavailableFacet<Key extends string, Value>(): EvidenceFacet<
  Key,
  Value
> {
  return { status: "unavailable" };
}

function collectEvidenceReferences(
  project: string,
  personalIssues: EvidenceFacet<"issues", ProjectIssue[]>,
  databases: EvidenceFacet<"databases", { name: string }[]>,
  openDatabaseChanges: EvidenceFacet<"issues", DatabaseChange[]>
): AgentEvidence {
  const references: AgentEvidenceReference[] = [];
  const seen = new Set<string>();
  const add = (reference: AgentEvidenceReference) => {
    const key = `${reference.kind}:${reference.name}`;
    if (seen.has(key)) return;
    seen.add(key);
    references.push(reference);
  };

  if (openDatabaseChanges.status === "available") {
    for (const issue of openDatabaseChanges.issues) {
      add({ kind: "issue", name: issue.name, title: issue.title });
    }
  }
  if (personalIssues.status === "available") {
    for (const issue of personalIssues.issues) {
      add({ kind: "issue", name: issue.name, title: issue.title });
    }
  }
  if (databases.status === "available") {
    for (const database of databases.databases) {
      add({ kind: "database", name: database.name });
    }
  }

  return { project, references };
}

function parsePersonalIssues(
  value: string
): EvidenceFacet<"issues", ProjectIssue[]> {
  try {
    const payload = JSON.parse(value) as Record<string, unknown>;
    if (typeof payload.error === "string" || !Array.isArray(payload.issues)) {
      return unavailableFacet();
    }
    const issues = payload.issues.flatMap((issue) => {
      if (!issue || typeof issue !== "object") return [];
      const candidate = issue as Record<string, unknown>;
      if (
        typeof candidate.name !== "string" ||
        typeof candidate.title !== "string" ||
        (candidate.relation !== "review" && candidate.relation !== "created") ||
        typeof candidate.approvalStatus !== "string"
      ) {
        return [];
      }
      const relation: ProjectIssue["relation"] =
        candidate.relation === "review" ? "review" : "created";
      return [
        {
          name: candidate.name,
          title: candidate.title,
          relation,
          approvalStatus: candidate.approvalStatus,
        },
      ];
    });
    return {
      status: "available",
      partial: payload.partial === true,
      issues,
    };
  } catch {
    return unavailableFacet();
  }
}

export async function collectProjectOverview(
  projectName: string | undefined
): Promise<ProjectOverview | null> {
  if (!isProjectName(projectName)) {
    return null;
  }

  const { fetchDatabases, listIssues } = useAppStore.getState();
  const results = await Promise.allSettled([
    createProjectOpenIssuesTool({ projectName })(),
    fetchDatabases({
      parent: projectName,
      pageSize: DATABASE_PAGE_SIZE,
      silent: true,
    }),
    listIssues({
      find: {
        project: projectName,
        query: "",
        typeList: [Issue_Type.DATABASE_CHANGE],
        statusList: [IssueStatus.OPEN],
        orderBy: "update_time desc",
      },
      pageSize: DATABASE_CHANGE_PAGE_SIZE,
    }),
  ]);

  const personalIssues =
    results[0].status === "fulfilled"
      ? parsePersonalIssues(results[0].value)
      : unavailableFacet<"issues", ProjectIssue[]>();
  const databases =
    results[1].status === "fulfilled"
      ? {
          status: "available" as const,
          partial: Boolean(results[1].value.nextPageToken),
          databases: results[1].value.databases.map((database) => ({
            name: database.name,
          })),
        }
      : unavailableFacet<"databases", { name: string }[]>();
  const openDatabaseChanges =
    results[2].status === "fulfilled"
      ? {
          status: "available" as const,
          partial: Boolean(results[2].value.nextPageToken),
          issues: results[2].value.issues.map((issue) => ({
            name: issue.name,
            title: issue.title,
          })),
        }
      : unavailableFacet<"issues", DatabaseChange[]>();

  return {
    project: projectName,
    personalIssues,
    databases,
    openDatabaseChanges,
    evidence: collectEvidenceReferences(
      projectName,
      personalIssues,
      databases,
      openDatabaseChanges
    ),
  };
}

export function extractProjectOverviewEvidence(
  result: string
): AgentEvidence | undefined {
  try {
    const payload = JSON.parse(result) as { evidence?: unknown };
    const evidence = payload.evidence;
    if (!evidence || typeof evidence !== "object") return undefined;

    const candidate = evidence as {
      project?: unknown;
      references?: unknown;
    };
    const project = candidate.project;
    const rawReferences = candidate.references;
    if (
      typeof project !== "string" ||
      !isProjectName(project) ||
      !Array.isArray(rawReferences)
    ) {
      return undefined;
    }

    const references: AgentEvidenceReference[] = [];
    for (const reference of rawReferences) {
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
        continue;
      }
      if (value.kind === "database" && typeof value.name === "string") {
        references.push({ kind: "database", name: value.name });
      }
    }

    return { project, references };
  } catch {
    return undefined;
  }
}

export function createProjectOverviewTool({
  projectName,
}: {
  projectName?: string;
}) {
  return async (): Promise<string> => {
    const overview = await collectProjectOverview(projectName);
    return JSON.stringify(
      overview ?? { error: "This chat is not bound to a project." }
    );
  };
}
