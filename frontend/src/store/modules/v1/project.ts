import { orderBy, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watchEffect } from "vue";
import { useRoute } from "vue-router";
import { projectServiceClient } from "@/grpcweb";
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
import { State, stateToJSON } from "@/types/proto/v1/common";
import type {
  Project,
  ListProjectsResponse,
} from "@/types/proto/v1/project_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import { projectNamePrefix } from "./common";
import { useProjectIamPolicyStore } from "./projectIamPolicy";

export interface ProjectFilter {
  query?: string;
  excludeDefault?: boolean;
  state?: State;
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
  if (params.state === State.DELETED) {
    list.push(`state == "${stateToJSON(params.state)}"`);
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
    const project = await projectServiceClient.getProject({ name }, { silent });
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
    const request = hasWorkspacePermissionV2("bb.projects.list")
      ? projectServiceClient.listProjects
      : projectServiceClient.searchProjects;

    let response: ListProjectsResponse | undefined = undefined;
    let pageToken = params.pageToken;
    while (true) {
      const resp = await request(
        {
          ...params,
          pageToken,
          filter: getListProjectFilter(params.filter ?? {}),
          showDeleted: params.filter?.state === State.DELETED ? true : false,
        },
        { silent: params.silent ?? true }
      );
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
    const created = await projectServiceClient.createProject({
      project,
      projectId: resourceId,
    });
    const composed = await upsertProjectMap([created]);
    return composed[0];
  };
  const updateProject = async (project: Project, updateMask: string[]) => {
    const updated = await projectServiceClient.updateProject({
      project,
      updateMask,
    });
    const composed = await upsertProjectMap([updated]);
    return composed[0];
  };
  const archiveProject = async (project: Project, force = false) => {
    await projectServiceClient.deleteProject({
      name: project.name,
      force,
    });
    project.state = State.DELETED;
    await upsertProjectMap([project]);
  };
  const batchDeleteProjects = async (projectNames: string[], force = false) => {
    await projectServiceClient.batchDeleteProjects({
      names: projectNames,
      force,
    });
    // Update local cache - mark all projects as deleted
    const projects = projectNames
      .map((name) => {
        const project = getProjectByName(name);
        if (project && project.name !== UNKNOWN_PROJECT_NAME) {
          // Extract Project properties (excluding iamPolicy)
          const { iamPolicy: _iamPolicy, ...projectData } = project;
          return { ...projectData, state: State.DELETED };
        }
        return null;
      })
      .filter((p): p is Project => p !== null);

    if (projects.length > 0) {
      await upsertProjectMap(projects);
    }
  };
  const restoreProject = async (project: Project) => {
    await projectServiceClient.undeleteProject({
      name: project.name,
    });
    project.state = State.ACTIVE;
    await upsertProjectMap([project]);
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
