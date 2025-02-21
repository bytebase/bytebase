<template>
  <div class="w-full space-y-6 mb-6">
    <div class="divide-y divide-block-border space-y-6">
      <ProjectGeneralSettingPanel
        ref="projectGeneralSettingPanelRef"
        :project="project"
        :allow-edit="allowEdit"
      />
      <ProjectSecuritySettingPanel
        ref="projectSecuritySettingPanelRef"
        :project="project"
        :allow-edit="allowEdit"
      />
      <ProjectIssueRelatedSettingPanel
        v-if="databaseChangeMode === DatabaseChangeMode.PIPELINE"
        ref="projectIssueRelatedSettingPanelRef"
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
import { useEventListener } from "@vueuse/core";
import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { onBeforeRouteLeave } from "vue-router";
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
const { t } = useI18n();

const projectSecuritySettingPanelRef =
  ref<InstanceType<typeof ProjectSecuritySettingPanel>>();
const projectGeneralSettingPanelRef =
  ref<InstanceType<typeof ProjectGeneralSettingPanel>>();
const projectIssueRelatedSettingPanelRef =
  ref<InstanceType<typeof ProjectIssueRelatedSettingPanel>>();

const isDirty = computed(() => {
  return (
    projectSecuritySettingPanelRef.value?.isDirty ||
    projectGeneralSettingPanelRef.value?.isDirty ||
    projectIssueRelatedSettingPanelRef.value?.isDirty
  );
});

useEventListener("beforeunload", (e) => {
  if (!isDirty.value) {
    return;
  }
  e.returnValue = t("common.leave-without-saving");
  return e.returnValue;
});

onBeforeRouteLeave((to, from, next) => {
  if (isDirty.value) {
    if (!window.confirm(t("common.leave-without-saving"))) {
      return;
    }
  }
  next();
});
</script>
