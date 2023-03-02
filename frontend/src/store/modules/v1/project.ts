import { defineStore } from "pinia";
import { projectServiceClient } from "@/grpcweb";
import { ResourceId } from "@/types";
import { Project } from "@/types/proto/v1/project_service";

interface ProjectState {
  projectMapByName: Map<ResourceId, Project>;
}

export const useProjectV1Store = defineStore("project_v1", {
  state: (): ProjectState => ({
    projectMapByName: new Map(),
  }),
  getters: {
    projectList(state) {
      return Array.from(state.projectMapByName.values());
    },
  },
  actions: {
    async fetchProjects(showDeleted = false) {
      const { projects } = await projectServiceClient.listProjects({
        showDeleted,
      });
      for (const project of projects) {
        this.projectMapByName.set(project.name, project);
      }
      return projects;
    },
    async getOrFetchProjectByName(name: string) {
      const cachedData = this.projectMapByName.get(name);
      if (cachedData) {
        return cachedData;
      }
      const project = await projectServiceClient.getProject({
        name,
      });
      this.projectMapByName.set(project.name, project);
      return project;
    },
  },
});
