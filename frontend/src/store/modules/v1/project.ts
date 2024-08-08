import { orderBy } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watchEffect } from "vue";
import { projectServiceClient } from "@/grpcweb";
import type { ComposedProject, MaybeRef, ResourceId } from "@/types";
import {
  emptyProject,
  EMPTY_ID,
  EMPTY_PROJECT_NAME,
  unknownProject,
  defaultProject,
  UNKNOWN_PROJECT_NAME,
  DEFAULT_PROJECT_NAME,
  UNKNOWN_ID,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import type { Project } from "@/types/proto/v1/project_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import { useCurrentUserV1 } from "../auth";
import { getResourceStoreCacheKey, type StoreCache } from "./cache";
import { useProjectIamPolicyStore } from "./projectIamPolicy";

export const useProjectV1Store = defineStore("project_v1", () => {
  const listCache = reactive(new Map<string, StoreCache>());
  const currentUser = useCurrentUserV1();
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
  const listProjects = async (showDeleted = false) => {
    const cacheKey = getResourceStoreCacheKey(
      "project",
      showDeleted ? "" : "active"
    );
    if (!listCache.has(cacheKey)) {
      listCache.set(cacheKey, {
        timestamp: Date.now(),
        isFetching: true,
      });
    }
    const request = hasWorkspacePermissionV2(
      currentUser.value,
      "bb.projects.list"
    )
      ? projectServiceClient.listProjects
      : projectServiceClient.searchProjects;
    const { projects } = await request({ showDeleted });
    const composedProjects = await upsertProjectMap(projects);
    listCache.set(cacheKey, {
      timestamp: Date.now(),
      isFetching: false,
    });
    return composedProjects;
  };
  const getProjectList = (showDeleted = false) => {
    if (unref(showDeleted)) {
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
  const findProjectByUID = (uid: string) => {
    if (uid === String(EMPTY_ID)) {
      return emptyProject();
    }
    if (uid === String(UNKNOWN_ID)) {
      return unknownProject();
    }
    return (
      projectList.value.find((project) => project.uid === uid) ??
      unknownProject()
    );
  };
  const fetchProjectByName = async (name: string, silent = false) => {
    const project = await projectServiceClient.getProject({ name }, { silent });
    await upsertProjectMap([project]);
    return project as ComposedProject;
  };
  const getOrFetchProjectByName = async (name: string, silent = false) => {
    const cachedData = projectMapByName.get(name);
    if (cachedData) {
      return cachedData;
    }
    return fetchProjectByName(name, silent);
  };
  const createProject = async (project: Project, resourceId: string) => {
    const created = await projectServiceClient.createProject({
      project,
      projectId: resourceId,
    });
    await upsertProjectMap([created]);
    return created;
  };
  const updateProject = async (project: Project, updateMask: string[]) => {
    const updated = await projectServiceClient.updateProject({
      project,
      updateMask,
    });
    await upsertProjectMap([updated]);
    return updated;
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
    listCache,
    projectMapByName,
    projectList,
    getProjectList,
    findProjectByUID,
    getProjectByName,
    listProjects,
    getOrFetchProjectByName,
    createProject,
    updateProject,
    archiveProject,
    restoreProject,
    updateProjectCache,
  };
});

export const useProjectV1List = (showDeleted: MaybeRef<boolean> = false) => {
  const store = useProjectV1Store();
  const cacheKey = getResourceStoreCacheKey(
    "project",
    showDeleted ? "" : "active"
  );
  const cache = computed(() => store.listCache.get(cacheKey));
  const requestPromise = ref<Promise<ComposedProject[]> | null>(null);

  watchEffect(() => {
    // If request is already in progress, do not send another request.
    if (cache.value?.isFetching) {
      return;
    }
    // If cache is available, do not send another request.
    if (cache.value) {
      return;
    }
    requestPromise.value = store.listProjects(unref(showDeleted));
  });

  const projectList = computed(() => {
    return store.getProjectList(unref(showDeleted));
  });
  return {
    projectList,
    ready: computed(() => cache.value && !cache.value.isFetching),
    requestPromise,
  };
};

export const useProjectByName = (name: MaybeRef<string>) => {
  const store = useProjectV1Store();
  const ready = ref(false);
  watchEffect(() => {
    ready.value = false;
    store.getOrFetchProjectByName(unref(name)).then(() => {
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
