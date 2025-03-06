import { orderBy, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watchEffect } from "vue";
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
import { State } from "@/types/proto/v1/common";
import type {
  Project,
  ListProjectsResponse,
} from "@/types/proto/v1/project_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import { useListCache } from "./cache";
import { useProjectIamPolicyStore } from "./projectIamPolicy";

export const useProjectV1Store = defineStore("project_v1", () => {
  const projectMapByName = reactive(new Map<ResourceId, ComposedProject>());

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
    showDeleted: boolean;
    pageSize: number;
    pageToken?: string;
    query?: string;
  }): Promise<{
    projects: ComposedProject[];
    nextPageToken?: string;
  }> => {
    const request =
      hasWorkspacePermissionV2("bb.projects.list") && !params.query
        ? projectServiceClient.listProjects
        : projectServiceClient.searchProjects;
    const response = await request(params);
    const composedProjects = await upsertProjectMap(response.projects);

    return {
      projects: composedProjects,
      nextPageToken: (response as ListProjectsResponse).nextPageToken,
    };
  };

  const searchProjects = async ({
    query,
    silent = true,
    showDeleted = false,
  }: {
    query: string;
    silent?: boolean;
    showDeleted?: boolean;
  }) => {
    const { projects } = await projectServiceClient.searchProjects(
      {
        query,
        showDeleted,
      },
      { silent }
    );
    return await upsertProjectMap(projects);
  };

  const getOrFetchProjectByName = async (name: string, silent = false) => {
    const cachedData = getProjectByName(name);
    if (cachedData && cachedData.name !== UNKNOWN_PROJECT_NAME) {
      return cachedData;
    }
    if (!isValidProjectName(name)) {
      return unknownProject();
    }
    return fetchProjectByName(name, silent);
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
    restoreProject,
    updateProjectCache,
    searchProjects,
    fetchProjectList,
  };
});

export const useProjectV1List = (showDeleted: boolean = false) => {
  const listCache = useListCache("project");
  const store = useProjectV1Store();
  const cacheKey = listCache.getCacheKey(showDeleted ? "" : "active");

  const cache = computed(() => listCache.getCache(cacheKey));

  watchEffect(async () => {
    // Skip if request is already in progress or cache is available.
    if (cache.value?.isFetching || cache.value) {
      return;
    }

    listCache.cacheMap.set(cacheKey, {
      timestamp: Date.now(),
      isFetching: true,
    });
    const { projects } = await store.fetchProjectList({
      showDeleted,
      pageSize: 100,
    });
    await store.upsertProjectMap(projects);
    listCache.cacheMap.set(cacheKey, {
      timestamp: Date.now(),
      isFetching: false,
    });
  });

  const projectList = computed(() => {
    return store.getProjectList(showDeleted);
  });

  return {
    projectList,
    ready: computed(() => cache.value && !cache.value.isFetching),
  };
};

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

const projectRequestCache = new Map<string, Promise<ComposedProject>>();

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
      const cached = projectRequestCache.get(projectName);
      if (cached) return cached;
      const request = store.getOrFetchProjectByName(
        projectName,
        true /* silent */
      );
      projectRequestCache.set(projectName, request);
      return request;
    })
  );
};
