<template>
  <div
    ref="containerRef"
    class="h-full flex flex-row overflow-hidden"
    :data-width="containerWidth"
  >
    <div class="h-full border-r shrink-0">
      <GutterBar size="medium" />
    </div>
    <div class="h-full flex-1 flex flex-col overflow-hidden">
      <div
        v-if="!strictProject && !hideProjects"
        class="flex flex-row items-center gap-x-1 px-1 py-1 border-b"
      >
        <ProjectSelect
          style="width: 100%"
          class="project-select"
          :project-name="projectName"
          :include-all="false"
          :include-default-project="allowAccessDefaultProject"
          :loading="!projectContextReady"
          @update:project-name="handleSwitchProject"
        />
      </div>

      <div class="flex-1 flex flex-row overflow-hidden">
        <div class="h-full flex-1 flex flex-col pt-1 overflow-hidden">
          <WorksheetPane v-if="asidePanelTab === 'WORKSHEET'" />
          <SchemaPane v-if="asidePanelTab === 'SCHEMA'" />
          <HistoryPane v-if="asidePanelTab === 'HISTORY'" />
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { storeToRefs } from "pinia";
import { computed, ref, watch } from "vue";
import { ProjectSelect } from "@/components/v2";
import {
  useSQLEditorTreeStore,
  useSQLEditorStore,
  useAppFeature,
} from "@/store";
import { defaultProject, isValidProjectName } from "@/types";
import { hasProjectPermissionV2 } from "@/utils";
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
const containerRef = ref<HTMLDivElement>();
const { width: containerWidth } = useElementSize(containerRef);
const hideProjects = useAppFeature("bb.feature.sql-editor.hide-projects");

const projectName = computed(() => {
  return editorStore.currentProject?.name ?? null;
});

const allowAccessDefaultProject = computed(() => {
  return hasProjectPermissionV2(defaultProject(), "bb.projects.get");
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

const handleSwitchProject = (name: string | undefined) => {
  if (!name || !isValidProjectName(name)) {
    project.value = "";
  } else {
    project.value = name;
  }
};
</script>

<style lang="postcss" scoped>
.project-select :deep(.n-base-selection) {
  --n-height: 30px !important;
}
</style>
