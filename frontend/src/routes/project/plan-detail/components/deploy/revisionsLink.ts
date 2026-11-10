import type { RevisionsLinkTarget } from "@/components/task-run-log/ReleaseSpineView";
import { PROJECT_DATABASE_DETAIL_TAB_REVISION } from "../../../database-detail/tabs";

// "instances/{i}/databases/{d}", optionally under projects/{p}/. The
// instance's full name is the database page's `parent`: a project instance
// is not found under the workspace's instances.
const parseDatabaseTarget = (target: string) => {
  const match =
    /^((?:projects\/[^/]+\/)?instances\/([^/]+))\/databases\/([^/]+)$/.exec(
      target
    );
  return match
    ? { parent: match[1], instance: match[2], database: match[3] }
    : undefined;
};

// The task database's Revisions tab, each route segment encoded.
export const revisionsLinkOf = (
  taskTarget: string,
  projectId: string
): RevisionsLinkTarget | undefined => {
  const target = parseDatabaseTarget(taskTarget);
  if (!target) return undefined;
  return {
    href: `/projects/${encodeURIComponent(projectId)}/instances/${encodeURIComponent(target.instance)}/databases/${encodeURIComponent(target.database)}?parent=${encodeURIComponent(target.parent)}#${PROJECT_DATABASE_DETAIL_TAB_REVISION}`,
    database: target.database,
  };
};
