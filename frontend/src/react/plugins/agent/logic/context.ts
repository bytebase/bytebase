import type { RouteLocationNormalizedLoaded } from "vue-router";
import { useAppStore } from "@/react/stores/app";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import { Issue_Type, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";

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
    const user =
      useAppStore.getState().currentUser ??
      (await useAppStore.getState().loadCurrentUser());
    if (user?.name) {
      ctx.user = {
        name: user.name,
        email: user.email,
        title: user.title,
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
      const project = useAppStore
        .getState()
        .getProjectByName(`projects/${projectId}`);
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
      const db = useAppStore
        .getState()
        .getDatabaseByName(`instances/${instanceId}/databases/${databaseName}`);
      if (db?.name) {
        ctx.database = {
          name: db.name,
          engine: db.instanceResource
            ? (Engine[db.instanceResource.engine] ?? "")
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
      const issue = await useAppStore
        .getState()
        .fetchIssueByName(`projects/${projectId}/issues/${issueId}`, true);
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
