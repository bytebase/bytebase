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
    for (const project of projectList) {
      const composed = await composeProjectIamPolicy(project);
      projectMapByName.set(composed.name, composed);
    }
  };
  const fetchProjectList = async (showDeleted = false) => {
    const { projects } = await projectServiceClient.listProjects({
      showDeleted,
    });
    await upsertProjectMap(projects);
    return projects;
  };
  const getProjectByUID = (uid: IdType) => {
    if (typeof uid === "string") {
      uid = parseInt(uid, 10);
    }
    if (uid === EMPTY_ID) {
      return emptyProject();
    }
    return (
      projectList.value.find((project) => parseInt(project.uid, 10) === uid) ??
      unknownProject()
    );
  };
  const fetchProjectByName = async (name: string) => {
    const project = await projectServiceClient.getProject({
      name,
    });
    await upsertProjectMap([project]);
    return project;
  };
  const fetchProjectByUID = async (uid: IdType) => {
    return fetchProjectByName(`projects/${uid}`);
  };
  const getOrFetchProjectByName = async (name: string) => {
    const cachedData = projectMapByName.get(name);
    if (cachedData) {
      return cachedData;
    }
    return fetchProjectByName(name);
  };
  const getOrFetchProjectByUID = async (uid: IdType) => {
    const cachedData = projectList.value.find(
      (project) => parseInt(project.uid, 10) === uid
    );
    if (cachedData) {
      return cachedData;
    }
    return fetchProjectByUID(uid);
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
    if (unref(showDeleted)) {
      return store.projectList;
    }
    return store.projectList.filter(
      (project) => project.state === State.ACTIVE
    );
  });
  return { projectList, ready };
};

export const useProjectV1ListByUser = (
  user: MaybeRef<User>,
  showDeleted: MaybeRef<boolean> = false
) => {
  const { projectList: rawProjectList, ready } = useProjectV1List(showDeleted);
  const projectList = computed(() => {
    return rawProjectList.value.filter((project) => {
      return project.iamPolicy.bindings.some((binding) => {
        return binding.members.some((email) => {
          return email === `user:${unref(user).email}`;
        });
      });
    });
  });
  return { projectList, ready };
};

export const useProjectV1ListByCurrentUser = (
  showDeleted: MaybeRef<boolean> = false
) => useProjectV1ListByUser(useCurrentUserV1(), showDeleted);

export const useProjectV1ByUID = (uid: MaybeRef<IdType>) => {
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

const composeProjectIamPolicy = async (project: Project) => {
  const policy = await useProjectIamPolicyStore().getOrFetchProjectIamPolicy(
    project.name
  );
  const composedProject = project as ComposedProject;
  composedProject.iamPolicy = policy;
  return composedProject;
};
