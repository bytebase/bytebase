import { useLocalStorage } from "@vueuse/core";
import { computed } from "vue";
import { useProjectV1Store, useCurrentUserIamPolicy } from "@/store";

const MAX_RECENT_PROJECT = 5;

export const useRecentProjects = () => {
  const projectV1Store = useProjectV1Store();
  const currentUserIamPolicy = useCurrentUserIamPolicy();

  const recentViewProjectNames = useLocalStorage<string[]>(
    "bb.project.recent-view",
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

  const recentViewProjects = computed(() => {
    return recentViewProjectNames.value
      .filter((project) => currentUserIamPolicy.isMemberOfProject(project))
      .map((project) => {
        return projectV1Store.getProjectByName(project);
      });
  });

  return {
    setRecentProject,
    recentViewProjects,
  };
};
