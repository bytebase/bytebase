import type { InjectionKey, Ref } from "vue";
import { inject, provide, ref } from "vue";
import { useCurrentProjectV1 } from "@/store";
import type { ComposedProject } from "@/types";

export interface FileToCreate {
  // id is the temporary id for the file, mainly using in frontend.
  id: string;
  path: string;
  version: string;
  statement: string;
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
  const { project } = useCurrentProjectV1();

  const title = ref("");
  const files = ref<FileToCreate[]>([]);

  const context: ReleaseCreateContext = {
    title,
    files,
    project,
  };

  provide(KEY, context);

  return context;
};
