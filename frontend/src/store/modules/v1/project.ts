import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watchEffect } from "vue";

import {
  ComposedProject,
  emptyProject,
  EMPTY_ID,
  IdType,
  MaybeRef,
  ResourceId,
  unknownProject,
  UNKNOWN_ID,
} from "@/types";
import { projectServiceClient } from "@/grpcweb";
import { Project } from "@/types/proto/v1/project_service";
import { useProjectIamPolicyStore } from "./projectIamPolicy";
import { State } from "@/types/proto/v1/common";
import { User } from "@/types/proto/v1/auth_service";
import { useCurrentUserV1 } from "../auth";

export const useProjectV1Store = defineStore("project_v1", () => {
  const projectMapByName = reactive(new Map<ResourceId, ComposedProject>());

  // Getters
  const projectList = computed(() => {
    return Array.from(projectMapByName.values());
  });

  // Actions
  const upsertProjectMap = async (projectList: Project[]) => {
    const composedProjectList = await batchComposeProjectIamPolicy(projectList);
    composedProjectList.forEach((project) => {
      projectMapByName.set(project.name, project);
    });
  };
  const fetchProjectList = async (showDeleted = false) => {
    const { projects } = await projectServiceClient.listProjects({
      showDeleted,
    });
    await upsertProjectMap(projects);
    return projects;
  };
  const getProjectList = (showDeleted = false) => {
    if (unref(showDeleted)) {
      return projectList.value;
    }
    return projectList.value.filter(
      (project) => project.state === State.ACTIVE
    );
  };
  const getProjectListByUser = (user: User, showDeleted = false) => {
    return getProjectList(showDeleted).filter((project) => {
      return project.iamPolicy.bindings.some((binding) => {
        return binding.members.some((email) => {
          return email === `user:${unref(user).email}`;
        });
      });
    });
  };
  const getProjectByUID = (uid: string) => {
    if (uid === String(EMPTY_ID)) {
      return emptyProject();
    }
    return (
      projectList.value.find((project) => project.uid === uid) ??
      unknownProject()
    );
  };
  const fetchProjectByName = async (name: string) => {
    const project = await projectServiceClient.getProject({
      name,
    });
    await upsertProjectMap([project]);
    return project as ComposedProject;
  };
  const fetchProjectByUID = async (uid: string) => {
    return fetchProjectByName(`projects/${uid}`);
  };
  const getOrFetchProjectByName = async (name: string) => {
    const cachedData = projectMapByName.get(name);
    if (cachedData) {
      return cachedData;
    }
    return fetchProjectByName(name);
  };
  const getOrFetchProjectByUID = async (uid: string) => {
    if (uid === String(EMPTY_ID)) return emptyProject();
    if (uid === String(UNKNOWN_ID)) return unknownProject();

    const cachedData = projectList.value.find((project) => project.uid === uid);
    if (cachedData) {
      return cachedData;
    }
    await fetchProjectByUID(uid);
    return getProjectByUID(uid);
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
  const archiveProject = async (project: Project) => {
    await projectServiceClient.deleteProject({
      name: project.name,
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
    projectMapByName,
    projectList,
    getProjectList,
    getProjectListByUser,
    upsertProjectMap,
    getProjectByUID,
    fetchProjectList,
    fetchProjectByName,
    fetchProjectByUID,
    getOrFetchProjectByName,
    getOrFetchProjectByUID,
    createProject,
    updateProject,
    archiveProject,
    restoreProject,
  };
});

export const useProjectV1List = (showDeleted: MaybeRef<boolean> = false) => {
  const store = useProjectV1Store();
  const ready = ref(false);
  watchEffect(() => {
    ready.value = false;
    store.fetchProjectList(unref(showDeleted)).then(() => {
      ready.value = true;
    });
  });
  const projectList = computed(() => {
    return store.getProjectList(unref(showDeleted));
  });
  return { projectList, ready };
};

export const useProjectV1ListByUser = (
  user: MaybeRef<User>,
  showDeleted: MaybeRef<boolean> = false
) => {
  const store = useProjectV1Store();
  const { ready } = useProjectV1List(showDeleted);
  const projectList = computed(() => {
    return store.getProjectListByUser(unref(user), unref(showDeleted));
  });
  return { projectList, ready };
};

export const useProjectV1ListByCurrentUser = (
  showDeleted: MaybeRef<boolean> = false
) => useProjectV1ListByUser(useCurrentUserV1(), showDeleted);

export const useProjectV1ByUID = (uid: MaybeRef<string>) => {
  const store = useProjectV1Store();
  const ready = ref(false);
  watchEffect(() => {
    ready.value = false;
    store.getOrFetchProjectByUID(unref(uid)).then(() => {
      ready.value = true;
    });
  });
  const project = computed(() => {
    return store.getProjectByUID(unref(uid));
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
