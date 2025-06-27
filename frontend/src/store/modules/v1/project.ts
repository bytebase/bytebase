import { orderBy, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watchEffect } from "vue";
import { useRoute } from "vue-router";
import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { projectServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import type { ComposedProject, MaybeRef, ResourceId } from "@/types";
import {
  emptyProject,
  EMPTY_PROJECT_NAME,
  unknownProject,
  defaultProject,
  UNKNOWN_PROJECT_NAME,
  DEFAULT_PROJECT_NAME,
  isValidProjectName,
} from "@/types";
import { State as NewState } from "@/types/proto-es/v1/common_pb";
import { convertStateToOld } from "@/utils/v1/common-conversions";
import type {
  Project,
  ListProjectsResponse,
} from "@/types/proto/v1/project_service";
import {
  GetProjectRequestSchema,
  ListProjectsRequestSchema,
  SearchProjectsRequestSchema,
  CreateProjectRequestSchema,
  UpdateProjectRequestSchema,
  DeleteProjectRequestSchema,
  BatchDeleteProjectsRequestSchema,
  UndeleteProjectRequestSchema,
} from "@/types/proto-es/v1/project_service_pb";
import { convertNewProjectToOld, convertOldProjectToNew } from "@/utils/v1/project-conversions";
import { hasWorkspacePermissionV2 } from "@/utils";
import { projectNamePrefix } from "./common";
import { useProjectIamPolicyStore } from "./projectIamPolicy";

export interface ProjectFilter {
  query?: string;
  excludeDefault?: boolean;
  state?: NewState;
}

const getListProjectFilter = (params: ProjectFilter) => {
  const list = [];
  const search = params.query?.trim().toLowerCase();
  if (search) {
    list.push(
      `(name.matches("${search}") || resource_id.matches("${search}"))`
    );
  }
  if (params.excludeDefault) {
    list.push("exclude_default == true");
  }
  if (params.state === NewState.DELETED) {
    list.push(`state == "${convertStateToOld(params.state)}"`);
  }
  return list.join(" && ");
};

export const useProjectV1Store = defineStore("project_v1", () => {
  const projectMapByName = reactive(new Map<ResourceId, ComposedProject>());
  const projectRequestCache = new Map<string, Promise<ComposedProject>>();

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
  const updateProjectCache = (project: ComposedProject) => {
    projectMapByName.set(project.name, project);
  };
  const upsertProjectMap = async (projectList: Project[]) => {
    const composedProjectList = await batchComposeProjectIamPolicy(projectList);
    composedProjectList.forEach((project) => {
      updateProjectCache(project);
    });
    return composedProjectList;
  };
  const getProjectList = (showDeleted = false) => {
    if (showDeleted) {
      return projectList.value;
    }
    return projectList.value.filter(
      (project) => project.state === convertStateToOld(NewState.ACTIVE)
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
    const project = convertNewProjectToOld(response);
    await upsertProjectMap([project]);
    return project as ComposedProject;
  };

  const fetchProjectList = async (params: {
    pageSize?: number;
    pageToken?: string;
    silent?: boolean;
    filter?: ProjectFilter;
  }): Promise<{
    projects: ComposedProject[];
    nextPageToken?: string;
  }> => {
    const contextValues = createContextValues().set(silentContextKey, params.silent ?? true);

    let response: ListProjectsResponse | undefined = undefined;
    let pageToken = params.pageToken;
    while (true) {
      let resp;
      if (hasWorkspacePermissionV2("bb.projects.list")) {
        const request = create(ListProjectsRequestSchema, {
          ...params,
          pageToken,
          filter: getListProjectFilter(params.filter ?? {}),
          showDeleted: params.filter?.state === NewState.DELETED ? true : false,
        });
        const connectResponse = await projectServiceClientConnect.listProjects(request, { contextValues });
        resp = {
          projects: connectResponse.projects.map(convertNewProjectToOld),
          nextPageToken: connectResponse.nextPageToken,
        };
      } else {
        const request = create(SearchProjectsRequestSchema, {
          ...params,
          pageToken,
          filter: getListProjectFilter(params.filter ?? {}),
          showDeleted: params.filter?.state === NewState.DELETED ? true : false,
        });
        const connectResponse = await projectServiceClientConnect.searchProjects(request, { contextValues });
        resp = {
          projects: connectResponse.projects.map(convertNewProjectToOld),
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

    const composedProjects = await upsertProjectMap(response.projects);

    return {
      projects: composedProjects,
      nextPageToken: (response as ListProjectsResponse).nextPageToken,
    };
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
      project: convertOldProjectToNew(project),
      projectId: resourceId,
    });
    const response = await projectServiceClientConnect.createProject(request);
    const created = convertNewProjectToOld(response);
    const composed = await upsertProjectMap([created]);
    return composed[0];
  };
  const updateProject = async (project: Project, updateMask: string[]) => {
    const request = create(UpdateProjectRequestSchema, {
      project: convertOldProjectToNew(project),
      updateMask: { paths: updateMask },
    });
    const response = await projectServiceClientConnect.updateProject(request);
    const updated = convertNewProjectToOld(response);
    const composed = await upsertProjectMap([updated]);
    return composed[0];
  };
  const archiveProject = async (project: Project, force = false) => {
    const request = create(DeleteProjectRequestSchema, {
      name: project.name,
      force,
    });
    await projectServiceClientConnect.deleteProject(request);
    project.state = convertStateToOld(NewState.DELETED);
    await upsertProjectMap([project]);
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
          // Extract Project properties (excluding iamPolicy)
          const { iamPolicy: _iamPolicy, ...projectData } = project;
          return { ...projectData, state: convertStateToOld(NewState.DELETED) };
        }
        return null;
      })
      .filter((p): p is Project => p !== null);

    if (projects.length > 0) {
      await upsertProjectMap(projects);
    }
  };
  const restoreProject = async (project: Project) => {
    const request = create(UndeleteProjectRequestSchema, {
      name: project.name,
    });
    const response = await projectServiceClientConnect.undeleteProject(request);
    const restored = convertNewProjectToOld(response);
    await upsertProjectMap([restored]);
  };

  return {
    reset,
    upsertProjectMap,
    getProjectList,
    getProjectByName,
    getOrFetchProjectByName,
    createProject,
    updateProject,
    archiveProject,
    batchDeleteProjects,
    restoreProject,
    updateProjectCache,
    fetchProjectList,
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
    route.params.projectId
      ? `${projectNamePrefix}${route.params.projectId}`
      : unknownProject().name
  );
  return useProjectByName(projectName);
};

const batchComposeProjectIamPolicy = async (projectList: Project[]) => {
  const projectIamPolicyStore = useProjectIamPolicyStore();
  await projectIamPolicyStore.batchGetOrFetchProjectIamPolicy(
    projectList.map((project) => project.name)
  );
  return projectList.map((project) => {
    const policy = projectIamPolicyStore.getProjectIamPolicy(project.name);
    const composedProject = project as ComposedProject;
    composedProject.iamPolicy = policy;
    return composedProject;
  });
};

export const batchGetOrFetchProjects = async (projectNames: string[]) => {
  const store = useProjectV1Store();

  const distinctProjectList = uniq(projectNames);
  await Promise.all(
    distinctProjectList.map((projectName) => {
      if (
        !projectName ||
        !isValidProjectName(projectName) ||
        projectName === DEFAULT_PROJECT_NAME
      ) {
        return;
      }
      return store.getOrFetchProjectByName(projectName, true /* silent */);
    })
  );
};
