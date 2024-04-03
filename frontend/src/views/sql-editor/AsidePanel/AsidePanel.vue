<template>
  <div class="h-full flex flex-col overflow-hidden">
    <div
      v-if="!strictProject"
      class="flex flex-row items-center gap-x-1 px-1 py-1 border-b"
    >
      <ProjectSelect
        style="width: 100%"
        size="small"
        class="project-select"
        :project="projectUID"
        :include-all="false"
        :loading="!projectContextReady"
        @update:project="handleSwitchProject"
      />
    </div>

    <div class="flex-1 flex flex-row overflow-hidden">
      <div class="h-full border-r shrink-0">
        <GutterBar />
      </div>
      <div class="h-full flex-1 flex flex-col pt-1 overflow-hidden">
        <WorksheetPane v-if="asidePanelTab === 'WORKSHEET'" />
        <SchemaPane v-if="asidePanelTab === 'SCHEMA'" />
        <HistoryPane v-if="asidePanelTab === 'HISTORY'" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed, watch } from "vue";
import { ProjectSelect } from "@/components/v2";
import {
  useProjectV1Store,
  useSQLEditorTreeStore,
  useSQLEditorStore,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { useSQLEditorContext } from "../context";
import GutterBar from "./GutterBar";
import HistoryPane from "./HistoryPane";
import SchemaPane from "./SchemaPane";
import WorksheetPane from "./WorksheetPane";

const editorStore = useSQLEditorStore();
const treeStore = useSQLEditorTreeStore();
const { events, asidePanelTab } = useSQLEditorContext();
const { project, projectContextReady, strictProject } =
  storeToRefs(editorStore);

const projectUID = computed(() => {
  return editorStore.currentProject?.uid ?? null;
});

watch([project, projectContextReady], ([project, ready]) => {
  if (!ready) {
    treeStore.state = "LOADING";
  } else {
    treeStore.buildTree();
    treeStore.state = "READY";
    events.emit("tree-ready");
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

<style lang="postcss" scoped>
.project-select :deep(.n-base-selection) {
  --n-height: 25px !important;
}
</style>
