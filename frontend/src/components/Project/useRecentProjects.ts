import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
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
    const invalidProjects: string[] = [];

    await projectV1Store.batchGetProjects(recentViewProjectNames.value);

    for (const projectName of recentViewProjectNames.value) {
      try {
        const project = projectV1Store.getProjectByName(projectName);
        if (
          isValidProjectName(project.name) &&
          hasProjectPermissionV2(project, "bb.projects.get")
        ) {
          projects.push(project);
        } else {
          // Project exists but user lost access or it's invalid
          invalidProjects.push(projectName);
        }
      } catch {
        // Project was deleted or fetch failed - mark for removal
        invalidProjects.push(projectName);
      }
    }

    // Clean up invalid/deleted projects from localStorage
    if (invalidProjects.length > 0) {
      recentViewProjectNames.value = recentViewProjectNames.value.filter(
        (name) => !invalidProjects.includes(name)
      );
    }

    return projects;
  }, []);

  return {
    setRecentProject,
    recentViewProjects,
  };
};
