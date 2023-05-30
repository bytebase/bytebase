import { DEFAULT_PROJECT_ID, type Project } from "../types";

export function projectName(project: Project) {
  if (project.id === DEFAULT_PROJECT_ID) {
    return "Unassigned";
  }

  let name = project.name;
  if (project.rowStatus == "ARCHIVED") {
    name += " (Archived)";
  }
  return name;
}
