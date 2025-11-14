import { computedAsync } from "@vueuse/core";
import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, watchEffect } from "vue";
import { useRoute } from "vue-router";
import { useProjectV1Store, useReleaseStore } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { unknownProject, unknownRelease } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Release } from "@/types/proto-es/v1/release_service_pb";
import { hasPermissionToCreateChangeDatabaseIssueInProject } from "@/utils";

export type ReleaseDetailContext = {
  release: Ref<Release>;
  project: Ref<Project>;
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

  const project = computedAsync(async () => {
    const projectId = route.params.projectId as string;
    if (!projectId) {
      return unknownProject();
    }

    return await projectV1Store.getOrFetchProjectByName(
      `${projectNamePrefix}${projectId}`
    );
  }, unknownProject());

  const name = computed(() => {
    return `${projectNamePrefix}${route.params.projectId}/releases/${route.params.releaseId}`;
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
