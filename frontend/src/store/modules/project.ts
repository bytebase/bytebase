import { defineStore } from "pinia";
import {
  Project,
  ProjectMember,
  ProjectState,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";

function convert(
  project: ResourceObject,
  includedList: ResourceObject[]
): Project {
  const attrs = project.attributes as Omit<Project, "id" | "memberList">;
  // Only able to assign an empty member list, otherwise would cause circular dependency.
  // This should be fine as we shouldn't access member via member.project.memberList
  const projectWithoutMemberList: Project = {
    id: parseInt(project.id),
    resourceId: attrs.resourceId,
    rowStatus: attrs.rowStatus,
    name: attrs.name,
    key: attrs.key,
    memberList: [],
    workflowType: attrs.workflowType,
    visibility: attrs.visibility,
    tenantMode: attrs.tenantMode,
    schemaChangeType: attrs.schemaChangeType,
  };

  const memberList: ProjectMember[] = [];
  for (const item of includedList || []) {
    if (item.type == "projectMember") {
      const projectMemberIdList = project.relationships!.projectMember
        .data as ResourceIdentifier[];
      for (const idItem of projectMemberIdList) {
        if (idItem.id == item.id) {
          const member = convertMember(item, includedList);
          member.project = projectWithoutMemberList;
          memberList.push(member);
        }
      }
    }
  }

  return {
    ...(projectWithoutMemberList as Omit<Project, "memberList">),
    memberList,
  };
}

// For now, this is exclusively used as part of converting the Project.
// Upon calling, the project itself is not constructed yet, so we return
// an unknown project first.
function convertMember(
  projectMember: ResourceObject,
  includedList: ResourceObject[]
): ProjectMember {
  const attrs = projectMember.attributes as Omit<ProjectMember, "project">;

  return {
    id: projectMember.id,
    role: attrs.role,
    // `project` will be overwritten after the value is correctly composed
    project: unknown("PROJECT") as Project,
    principal: getPrincipalFromIncludedList(
      projectMember.relationships!.principal.data,
      includedList
    ),
  };
}

export const useLegacyProjectStore = defineStore("project_legacy", {
  state: (): ProjectState => ({
    projectById: new Map(),
  }),
  getters: {
    projectList: (state) => {
      return [...state.projectById.values()];
    },
  },
  actions: {
    convert(instance: ResourceObject, includedList: ResourceObject[]): Project {
      return convert(instance, includedList || []);
    },
  },
});
