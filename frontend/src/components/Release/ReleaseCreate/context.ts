import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { useRoute } from "vue-router";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { ComposedProject } from "@/types";
import { unknownProject } from "@/types";
import type { ReleaseFileType } from "@/types/proto/v1/release_service";

export interface FileToCreate {
  // id is the temporary id for the file, mainly using in frontend.
  id: string;
  name: string;
  version: string;
  statement: string;
  type: ReleaseFileType;
  /**
   * The sheet that holds the content.
   * Format: projects/{project}/sheets/{sheet}
   */
  sheet?: string;
}

export type ReleaseCreateContext = {
  title: Ref<string>;
  files: Ref<FileToCreate[]>;
  project: Ref<ComposedProject>;
};

export const KEY = Symbol(
  "bb.release.detail"
) as InjectionKey<ReleaseCreateContext>;

export const useReleaseCreateContext = () => {
  return inject(KEY)!;
};

export const provideReleaseCreateContext = () => {
  const route = useRoute();
  const projectV1Store = useProjectV1Store();

  const title = ref("");
  const files = ref<FileToCreate[]>([]);

  const project = computed(() => {
    const projectId = route.params.projectId as string;
    if (!projectId) {
      return unknownProject();
    }

    return projectV1Store.getProjectByName(`${projectNamePrefix}${projectId}`);
  });

  const context: ReleaseCreateContext = {
    title,
    files,
    project,
  };

  provide(KEY, context);

  return context;
};
