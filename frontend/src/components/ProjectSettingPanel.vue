<template>
  <div class="w-full space-y-6 mb-6">
    <div class="divide-y divide-block-border space-y-6">
      <ProjectGeneralSettingPanel :project="project" :allow-edit="allowEdit" />
      <ProjectSecuritySettingPanel :project="project" :allow-edit="allowEdit" />
      <ProjectIssueRelatedSettingPanel
        v-if="databaseChangeMode === DatabaseChangeMode.PIPELINE"
        :project="project"
        :allow-edit="allowEdit"
      />
      <div
        v-if="databaseChangeMode === DatabaseChangeMode.PIPELINE"
        class="pt-4"
      >
        <ProjectArchiveRestoreButton :project="project" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useAppFeature } from "@/store";
import type { ComposedProject } from "@/types";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import ProjectArchiveRestoreButton from "./Project/ProjectArchiveRestoreButton.vue";
import {
  ProjectGeneralSettingPanel,
  ProjectSecuritySettingPanel,
  ProjectIssueRelatedSettingPanel,
} from "./Project/Settings/";

defineProps<{
  project: ComposedProject;
  allowEdit: boolean;
}>();

const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");
</script>
