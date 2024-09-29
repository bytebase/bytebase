import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, watchEffect } from "vue";
import { useRoute } from "vue-router";
import { useProjectV1Store, useReleaseStore } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { ComposedProject, ComposedRelease } from "@/types";
import { unknownProject, unknownRelease } from "@/types";
import { hasPermissionToCreateChangeDatabaseIssueInProject } from "@/utils";

export type ReleaseDetailContext = {
  release: Ref<ComposedRelease>;
  project: Ref<ComposedProject>;
  allowApply: Ref<boolean>;
};

export const KEY = Symbol(
  "bb.release.detail"
) as InjectionKey<ReleaseDetailContext>;

export const useReleaseDetailContext = () => {
  return inject(KEY)!;
};

export const provideReleaseDetailContext = () => {
  const route = useRoute();
  const projectV1Store = useProjectV1Store();
  const releaseStore = useReleaseStore();

  const project = computed(() => {
    const projectId = route.params.projectId as string;
    if (!projectId) {
      return unknownProject();
    }

    return projectV1Store.getProjectByName(`${projectNamePrefix}${projectId}`);
  });

  const name = computed(() => {
    return `${project.value.name}/releases/${route.params.releaseId}`;
  });

  watchEffect(async () => {
    await releaseStore.fetchReleaseByName(name.value);
  });

  const release = computed(() => {
    return releaseStore.getReleaseByName(name.value) ?? unknownRelease();
  });

  const allowApply = computed(() => {
    return hasPermissionToCreateChangeDatabaseIssueInProject(project.value);
  });

  const context: ReleaseDetailContext = {
    release,
    project,
    allowApply,
  };

  provide(KEY, context);

  return context;
};
