<template>
  <div class="aside-panel h-full flex flex-col overflow-hidden">
    <div
      v-if="!strictProject"
      class="flex flex-row items-center gap-x-1 px-1 pt-1"
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
    <div class="flex-1 overflow-hidden pt-1">
      <DatabaseTree />
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
import DatabaseTree from "./DatabaseTree.vue";

const editorStore = useSQLEditorStore();
const treeStore = useSQLEditorTreeStore();
const { events } = useSQLEditorContext();
const { project, projectContextReady, strictProject } =
  storeToRefs(editorStore);

const projectUID = computed(() => {
  return treeStore.currentProject?.uid ?? null;
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
  --n-height: 30px !important;
}
</style>
