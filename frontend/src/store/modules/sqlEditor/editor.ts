import { useLocalStorage } from "@vueuse/core";
import { defineStore } from "pinia";
import { computed, ref, watch } from "vue";
import { ComposedDatabase } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";
import { useCurrentUserV1 } from "../auth";
import { useProjectV1Store } from "../v1";

// set the limit to 1000 temporarily to avoid the query timeout and page crash
export const RESULT_ROWS_LIMIT = 1000;

export const useSQLEditorStore = defineStore("sqlEditor", () => {
  // empty to "ALL" projects for high-privileged users
  const project = ref<string>("");
  // if `true`, won't show project selector and not allowed to switch to other projects
  const strictProject = ref<boolean>(false);
  // `false` if we are preparing project-scoped resources
  // we should render a skeleton layout with spinner placeholders
  const projectContextReady = ref<boolean>(false);
  const storedLastViewedProject = useLocalStorage<string>(
    "bb.sql-editor.last-viewed-project",
    ""
  );

  const currentProject = computed(() => {
    if (project.value) {
      return useProjectV1Store().getProjectByName(project.value);
    }
    return undefined;
  });
  const allowViewALLProjects = computed(() => {
    const me = useCurrentUserV1();
    return hasWorkspacePermissionV2(me.value, "bb.projects.list");
  });

  // `databaseList` is query-able databases scoped by `project`
  const databaseList = ref<ComposedDatabase[]>([]);

  watch(project, (project) => {
    storedLastViewedProject.value = project;
  });

  const isShowExecutingHint = ref(false);

  return {
    project,
    strictProject,
    projectContextReady,
    storedLastViewedProject,
    currentProject,
    allowViewALLProjects,
    databaseList,
    isShowExecutingHint,
  };
});
