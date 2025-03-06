import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import { useProjectV1Store, useCurrentUserV1 } from "@/store";
import { isValidProjectName } from "@/types";
import { hasProjectPermissionV2, useDynamicLocalStorage } from "@/utils";

const MAX_RECENT_PROJECT = 5;

export const useRecentProjects = () => {
  const projectV1Store = useProjectV1Store();
  const currentUser = useCurrentUserV1();

  const recentViewProjectNames = useDynamicLocalStorage<string[]>(
    computed(() => `bb.project.recent-view.${currentUser.value.name}`),
    []
  );

  const setRecentProject = (name: string) => {
    if (!name) {
      return;
    }
    const index = recentViewProjectNames.value.findIndex(
      (proj) => proj === name
    );
    if (index >= 0) {
      recentViewProjectNames.value.splice(index, 1);
    }

    recentViewProjectNames.value.unshift(name);
    if (recentViewProjectNames.value.length > MAX_RECENT_PROJECT) {
      recentViewProjectNames.value.pop();
    }
  };

  const recentViewProjects = computedAsync(async () => {
    const projects = [];
    for (const projectName of recentViewProjectNames.value) {
      const project = await projectV1Store.getOrFetchProjectByName(
        projectName,
        true /* silent */
      );
      if (
        isValidProjectName(project.name) &&
        hasProjectPermissionV2(project, "bb.projects.get")
      ) {
        projects.push(project);
      }
    }
    return projects;
  }, []);

  return {
    setRecentProject,
    recentViewProjects,
  };
};
