<template>
  <div class="aside-panel h-full flex flex-col overflow-hidden">
    <div
      v-if="!strictProject"
      class="flex flex-row items-center gap-x-1 px-1 py-1"
    >
      <div class="flex-1">
        <ProjectSelect
          style="width: 100%"
          :project="treeStore.selectedProject?.uid ?? String(UNKNOWN_ID)"
          :include-all="allowViewALLProjects"
          @update:project="handleSwitchProject"
        />
      </div>
      <GroupingBar class="shrink-0" />
    </div>
    <div class="flex-1 overflow-hidden">
      <DatabaseTree />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { watch } from "vue";
import { ProjectSelect } from "@/components/v2";
import {
  useProjectV1Store,
  useSQLEditorTreeStore,
  useSQLEditorV2Store,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import DatabaseTree from "./DatabaseTree.vue";
import GroupingBar from "./GroupingBar";

const sqlEditorStore = useSQLEditorV2Store();
const treeStore = useSQLEditorTreeStore();

const { project, projectContextReady, strictProject, allowViewALLProjects } =
  storeToRefs(sqlEditorStore);

watch([project, projectContextReady], ([project, ready]) => {
  if (!ready) {
    treeStore.state = "LOADING";
  } else {
    treeStore.buildTree();
    if (project) {
      const proj = useProjectV1Store().getProjectByName(project);
      treeStore.selectedProject = proj;
    } else {
      treeStore.selectedProject = undefined;
    }
    treeStore.state = "READY";
  }
});

const handleSwitchProject = (uid: string | undefined) => {
  if (!uid || uid === String(UNKNOWN_ID)) {
    project.value = "";
  } else {
    const proj = useProjectV1Store().getProjectByUID(uid);
    project.value = proj.name;
  }
};
</script>
