import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { orderBy, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watchEffect } from "vue";
import { useRoute } from "vue-router";
import { projectServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import type { MaybeRef, ResourceId } from "@/types";
import {
  DEFAULT_PROJECT_NAME,
  defaultProject,
  EMPTY_PROJECT_NAME,
  emptyProject,
  isValidProjectName,
  UNKNOWN_PROJECT_NAME,
  unknownProject,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  BatchDeleteProjectsRequestSchema,
  BatchGetProjectsRequestSchema,
  CreateProjectRequestSchema,
  DeleteProjectRequestSchema,
  GetProjectRequestSchema,
  ListProjectsRequestSchema,
  type Project,
  SearchProjectsRequestSchema,
  UndeleteProjectRequestSchema,
  UpdateProjectRequestSchema,
} from "@/types/proto-es/v1/project_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { projectNamePrefix } from "./common";
import { getLabelFilter } from "./database";

export interface ProjectFilter {
  query?: string;
  excludeDefault?: boolean;
  state?: State;
  // label should be "{label key}:{label value}" format
  labels?: string[];
}

const getListProjectFilter = (params: ProjectFilter) => {
  const list = [];
  const search = params.query?.trim();

  if (search) {
    // It's a regular name/resource_id search
    const searchLower = search.toLowerCase();
    list.push(
      `(name.matches("${searchLower}") || resource_id.matches("${searchLower}"))`
    );
  }

  if (params.labels) {
    list.push(...getLabelFilter(params.labels));
  }

  if (params.excludeDefault) {
    list.push("exclude_default == true");
  }
  if (params.state === State.DELETED) {
    list.push(`state == "${State[params.state]}"`);
  }
  return list.join(" && ");
};

export const useProjectV1Store = defineStore("project_v1", () => {
  const projectMapByName = reactive(new Map<ResourceId, Project>());
  const projectRequestCache = new Map<string, Promise<Project>>();

  const reset = () => {
    projectMapByName.clear();
  };

  // Getters
  const projectList = computed(() => {
    return orderBy(
      Array.from(projectMapByName.values()),
      (project) => project.name,
      "asc"
    );
  });

  // Actions
  const updateProjectCache = (project: Project) => {
    projectMapByName.set(project.name, project);
  };
  const upsertProjectsCache = (projectList: Project[]) => {
    projectList.forEach((project) => {
      updateProjectCache(project);
    });
  };
  const getProjectList = (showDeleted = false) => {
    if (showDeleted) {
      return projectList.value;
    }
    return projectList.value.filter(
      (project) => project.state === State.ACTIVE
    );
  };
  const getProjectByName = (name: string) => {
    if (name === EMPTY_PROJECT_NAME) return emptyProject();
    if (name === UNKNOWN_PROJECT_NAME) return unknownProject();
    if (name === DEFAULT_PROJECT_NAME) return defaultProject();
    return projectMapByName.get(name) ?? unknownProject();
  };
  const fetchProjectByName = async (name: string, silent = false) => {
    const request = create(GetProjectRequestSchema, { name });
    const response = await projectServiceClientConnect.getProject(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });

    upsertProjectsCache([response]);
    return response;
  };

  const fetchProjectList = async (params: {
    pageSize?: number;
    pageToken?: string;
    silent?: boolean;
    filter?: ProjectFilter;
    orderBy?: string;
  }): Promise<{
    projects: Project[];
    nextPageToken?: string;
  }> => {
    const contextValues = createContextValues().set(
      silentContextKey,
      params.silent ?? true
    );

    let response: { projects: Project[]; nextPageToken: string } | undefined =
      undefined;
    let pageToken = params.pageToken;
    while (true) {
      let resp;
      if (hasWorkspacePermissionV2("bb.projects.list")) {
        const request = create(ListProjectsRequestSchema, {
          ...params,
          pageToken,
          filter: getListProjectFilter(params.filter ?? {}),
          orderBy: params.orderBy,
          showDeleted: params.filter?.state === State.DELETED ? true : false,
        });
        const connectResponse = await projectServiceClientConnect.listProjects(
          request,
          { contextValues }
        );
        resp = {
          projects: connectResponse.projects,
          nextPageToken: connectResponse.nextPageToken,
        };
      } else {
        const request = create(SearchProjectsRequestSchema, {
          ...params,
          pageToken,
          filter: getListProjectFilter(params.filter ?? {}),
          orderBy: params.orderBy,
          showDeleted: params.filter?.state === State.DELETED ? true : false,
        });
        const connectResponse =
          await projectServiceClientConnect.searchProjects(request, {
            contextValues,
          });
        resp = {
          projects: connectResponse.projects,
          nextPageToken: connectResponse.nextPageToken,
        };
      }
      if (resp.nextPageToken !== "" && resp.projects.length === 0) {
        pageToken = resp.nextPageToken;
        continue;
      }
      response = resp;
      break;
    }

    upsertProjectsCache(response.projects);

    return {
      projects: response.projects,
      nextPageToken: response.nextPageToken,
    };
  };

  const batchGetProjects = async (names: string[], silent = true) => {
    const validNames = names.filter(isValidProjectName);
    if (validNames.length === 0) {
      return [];
    }
    const request = create(BatchGetProjectsRequestSchema, {
      names: validNames,
    });
    const response = await projectServiceClientConnect.batchGetProjects(
      request,
      {
        contextValues: createContextValues().set(silentContextKey, silent),
      }
    );
    upsertProjectsCache(response.projects);
    return response.projects;
  };

  const batchGetOrFetchProjects = async (projectNames: string[]) => {
    const validProjectList = uniq(projectNames).filter((projectName) => {
      if (
        !projectName ||
        !isValidProjectName(projectName) ||
        projectName === DEFAULT_PROJECT_NAME
      ) {
        return false;
      }
      return true;
    });
    const pendingFetch = validProjectList.filter((projectName) => {
      const project = getProjectByName(projectName);
      if (isValidProjectName(project.name)) {
        return false;
      }
      return true;
    });
    await batchGetProjects(pendingFetch, true /* silent */);
    return validProjectList.map(getProjectByName);
  };

  const getOrFetchProjectByName = async (name: string, silent = true) => {
    const cachedData = getProjectByName(name);
    if (cachedData && cachedData.name !== UNKNOWN_PROJECT_NAME) {
      return cachedData;
    }
    if (!isValidProjectName(name)) {
      return unknownProject();
    }

    const cached = projectRequestCache.get(name);
    if (cached) return cached;
    const request = fetchProjectByName(name, silent);
    projectRequestCache.set(name, request);
    return request;
  };
  const createProject = async (project: Project, resourceId: string) => {
    const request = create(CreateProjectRequestSchema, {
      project,
      projectId: resourceId,
    });
    const response = await projectServiceClientConnect.createProject(request);
    upsertProjectsCache([response]);
    return response;
  };
  const updateProject = async (project: Project, updateMask: string[]) => {
    const request = create(UpdateProjectRequestSchema, {
      project,
      updateMask: { paths: updateMask },
    });
    const response = await projectServiceClientConnect.updateProject(request);
    upsertProjectsCache([project]);
    return response;
  };
  const archiveProject = async (project: Project, force = false) => {
    const request = create(DeleteProjectRequestSchema, {
      name: project.name,
      force,
    });
    await projectServiceClientConnect.deleteProject(request);
    project.state = State.DELETED;
    upsertProjectsCache([project]);
  };
  const deleteProject = async (project: string) => {
    const request = create(DeleteProjectRequestSchema, {
      name: project,
      purge: true,
    });
    await projectServiceClientConnect.deleteProject(request);
    projectMapByName.delete(project);
  };
  const batchDeleteProjects = async (projectNames: string[], force = false) => {
    const request = create(BatchDeleteProjectsRequestSchema, {
      names: projectNames,
      force,
    });
    await projectServiceClientConnect.batchDeleteProjects(request);
    // Update local cache - mark all projects as deleted
    const projects = projectNames
      .map((name) => {
        const project = getProjectByName(name);
        if (project && project.name !== UNKNOWN_PROJECT_NAME) {
          return { ...project, state: State.DELETED };
        }
        return null;
      })
      .filter((p): p is Project => p !== null);

    upsertProjectsCache(projects);
  };
  const restoreProject = async (project: Project) => {
    const request = create(UndeleteProjectRequestSchema, {
      name: project.name,
    });
    const response = await projectServiceClientConnect.undeleteProject(request);
    upsertProjectsCache([response]);
  };

  return {
    reset,
    getProjectList,
    getProjectByName,
    getOrFetchProjectByName,
    createProject,
    updateProject,
    archiveProject,
    deleteProject,
    batchDeleteProjects,
    restoreProject,
    updateProjectCache,
    fetchProjectList,
    batchGetOrFetchProjects,
  };
});

export const useProjectByName = (name: MaybeRef<string>) => {
  const store = useProjectV1Store();
  const ready = ref(false);
  watchEffect(() => {
    ready.value = false;
    store.getOrFetchProjectByName(unref(name), /* silent */ true).then(() => {
      ready.value = true;
    });
  });
  const project = computed(() => {
    return store.getProjectByName(unref(name));
  });
  return { project, ready };
};

export const useCurrentProjectV1 = () => {
  const route = useRoute();
  const projectName = computed(() =>
    route?.params.projectId
      ? `${projectNamePrefix}${route.params.projectId}`
      : unknownProject().name
  );
  return useProjectByName(projectName);
};
