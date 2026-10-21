import { getRecentActivity } from "./recentActivity";

function isWorkspaceName(value: string | undefined): value is string {
  return !!value && /^workspaces\/[^/]+$/.test(value);
}

export function createWorkspaceRecentActivityTool({
  workspaceName,
}: {
  workspaceName?: string;
}) {
  return async (): Promise<string> => {
    if (!isWorkspaceName(workspaceName)) {
      return JSON.stringify({
        error: "This starter is not bound to a workspace.",
      });
    }

    const result = await getRecentActivity("projects/-");
    const payload = JSON.parse(result) as Record<string, unknown>;
    if (typeof payload.error === "string") {
      return result;
    }
    return JSON.stringify({ workspace: workspaceName, ...payload });
  };
}
