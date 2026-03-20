import { isDefaultProject } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

export const extractProjectResourceName = (name: string) => {
  const pattern = /(?:^|\/)projects\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export function projectV1Name(project: Project, workspaceResourceName: string) {
  if (isDefaultProject(project.name, workspaceResourceName)) {
    return "Unassigned";
  }

  return project.title;
}

export function filterProjectV1ListByKeyword<T extends Project>(
  projectList: T[],
  keyword: string
) {
  keyword = keyword.trim().toLowerCase();
  if (!keyword) return projectList;
  return projectList.filter((project) => {
    return (
      project.title.toLowerCase().includes(keyword) ||
      project.name.toLowerCase().includes(keyword)
    );
  });
}
