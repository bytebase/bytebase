import { useLocalStorage } from "@vueuse/core";
import { defineStore } from "pinia";
import { computed, ref, watch } from "vue";
import {
  SQLEditorState,
  QueryInfo,
  QueryHistory,
  ActivitySQLEditorQueryPayload,
  ComposedDatabase,
} from "@/types";
import { UNKNOWN_ID } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";
import { useCurrentUserV1 } from "./auth";
import { RESULT_ROWS_LIMIT } from "./sqlEditor";
import { useInstanceV1Store, useSQLStore, useActivityV1Store } from "./v1";

export const useSQLEditorV2Store = defineStore("sqlEditorV2", () => {
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

  const allowViewALLProjects = computed(() => {
    const me = useCurrentUserV1();
    return hasWorkspacePermissionV2(me.value, "bb.projects.list");
  });

  // `databaseList` is query-able databases scoped by `project`
  const databaseList = ref<ComposedDatabase[]>([]);

  watch(project, (project) => {
    storedLastViewedProject.value = project;
  });

  return {
    project,
    strictProject,
    projectContextReady,
    storedLastViewedProject,
    allowViewALLProjects,
    databaseList,
  };
});
