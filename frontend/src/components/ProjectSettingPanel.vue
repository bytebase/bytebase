<template>
  <div class="w-full space-y-6 mb-6">
    <div class="divide-y divide-block-border space-y-6">
      <ProjectGeneralSettingPanel :project="project" :allow-edit="allowEdit" />
      <ProjectSecuritySettingPanel :project="project" :allow-edit="allowEdit" />
      <ProjectBranchProtectionRulesSettingPanel
        :project="project"
        :allow-edit="allowEdit"
      />
      <ProjectIssueRelatedSettingPanel
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
import type { ComposedProject } from "@/types";
import ProjectArchiveRestoreButton from "./Project/ProjectArchiveRestoreButton.vue";
import {
  ProjectBranchProtectionRulesSettingPanel,
  ProjectGeneralSettingPanel,
  ProjectSecuritySettingPanel,
  ProjectIssueRelatedSettingPanel,
} from "./Project/Settings/";

defineProps<{
  project: ComposedProject;
  allowEdit: boolean;
}>();
</script>
