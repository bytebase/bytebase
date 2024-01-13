<template>
  <div class="w-full space-y-6 mb-6">
    <div class="divide-y divide-block-border space-y-6">
      <ProjectGeneralSettingPanel :project="project" :allow-edit="allowEdit" />
      <ProjectBranchProtectionRulesSettingPanel
        :project="project"
        :allow-edit="allowEdit"
      />
      <div class="pt-4">
        <ProjectArchiveRestoreButton :project="project" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ComposedProject } from "@/types";
import ProjectArchiveRestoreButton from "./Project/ProjectArchiveRestoreButton.vue";
import ProjectGeneralSettingPanel from "./Project/ProjectGeneralSettingPanel.vue";

defineProps<{
  project: ComposedProject;
  allowEdit: boolean;
}>();
</script>
