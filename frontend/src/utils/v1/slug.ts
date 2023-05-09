import slug from "slug";

import type { Project } from "@/types/proto/v1/project_service";

export function projectV1Slug(project: Project): string {
  return [slug(project.title), project.uid].join("-");
}
