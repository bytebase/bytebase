import { Project } from "../types";

export function projectName(project: Project) {
  let name = project.name;
  if (project.rowStatus == "ARCHIVED") {
    name += " (Archived)";
  }
  return name;
}
