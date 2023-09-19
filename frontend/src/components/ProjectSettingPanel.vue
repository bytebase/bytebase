<template>
  <div class="max-w-3xl mx-auto space-y-4">
    <div class="divide-y divide-block-border space-y-6">
      <ProjectGeneralSettingPanel :project="project" :allow-edit="allowEdit" />
    </div>
    <ProjectArchiveRestoreButton :project="project" />
  </div>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { ComposedProject } from "@/types";
import ProjectArchiveRestoreButton from "./Project/ProjectArchiveRestoreButton.vue";
import ProjectGeneralSettingPanel from "./Project/ProjectGeneralSettingPanel.vue";

defineProps({
  project: {
    required: true,
    type: Object as PropType<ComposedProject>,
  },
  allowEdit: {
    default: true,
    type: Boolean,
  },
});
</script>
