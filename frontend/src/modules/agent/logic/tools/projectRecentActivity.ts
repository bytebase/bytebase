import { getRecentActivity } from "./recentActivity";

function isProjectName(value: string | undefined): value is string {
  return !!value && /^projects\/[^/]+$/.test(value);
}

export function createProjectRecentActivityTool({
  projectName,
}: {
  projectName?: string;
}) {
  return async (): Promise<string> => {
    if (!isProjectName(projectName)) {
      return JSON.stringify({
        error: "This starter is not bound to a project.",
      });
    }

    const result = await getRecentActivity(projectName);
    const payload = JSON.parse(result) as Record<string, unknown>;
    if (typeof payload.error === "string") {
      return result;
    }
    return JSON.stringify({ project: projectName, ...payload });
  };
}
