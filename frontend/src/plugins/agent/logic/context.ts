import type { RouteLocationNormalizedLoaded } from "vue-router";
import { useCurrentUserV1 } from "@/store";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import {
  IssueStatus,
  Issue_Type,
} from "@/types/proto-es/v1/issue_service_pb";

interface PageContext {
  user?: { name: string; email: string; title: string };
  project?: { name: string; title: string; state: string };
  database?: { name: string; engine: string; environment: string };
  issue?: { name: string; title: string; status: string; type: string };
  [key: string]: unknown;
}

export async function extractRouteContext(
  route: RouteLocationNormalizedLoaded
): Promise<PageContext> {
  const ctx: PageContext = {};

  // Current user — always available
  try {
    const user = useCurrentUserV1();
    if (user.value?.name) {
      ctx.user = {
        name: user.value.name,
        email: user.value.email,
        title: user.value.title,
      };
    }
  } catch {
    // Store not initialized
  }

  const { projectId, databaseName, instanceId, issueId } =
    route.params as Record<string, string>;

  // Project context
  if (projectId) {
    try {
      const { useProjectV1Store } = await import("@/store");
      const store = useProjectV1Store();
      const project = store.getProjectByName(`projects/${projectId}`);
      if (project?.name) {
        ctx.project = {
          name: project.name,
          title: project.title,
          state: State[project.state] ?? "",
        };
      }
    } catch {
      // Store not available
    }
  }

  // Database context
  if (instanceId && databaseName) {
    try {
      const { useDatabaseV1Store } = await import("@/store");
      const store = useDatabaseV1Store();
      const db = store.getDatabaseByName(
        `instances/${instanceId}/databases/${databaseName}`
      );
      if (db?.name) {
        ctx.database = {
          name: db.name,
          engine: db.instanceResource
            ? Engine[db.instanceResource.engine] ?? ""
            : "",
          environment: db.effectiveEnvironment ?? "",
        };
      }
    } catch {
      // Store not available
    }
  }

  // Issue context
  if (projectId && issueId) {
    try {
      const { useIssueV1Store } = await import("@/store");
      const store = useIssueV1Store();
      const issue = await store.fetchIssueByName(
        `projects/${projectId}/issues/${issueId}`,
        true
      );
      if (issue?.name) {
        ctx.issue = {
          name: issue.name,
          title: issue.title,
          status: IssueStatus[issue.status] ?? "",
          type: Issue_Type[issue.type] ?? "",
        };
      }
    } catch {
      // Store not available or fetch failed
    }
  }

  return ctx;
}
