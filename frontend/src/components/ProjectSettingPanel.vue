<template>
  <div class="w-full space-y-6">
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

      <div v-if="allowEdit && isDirty" class="sticky bottom-0 z-10">
        <div
          class="flex justify-between w-full pt-4 border-block-border bg-white"
        >
          <NButton @click.prevent="onRevert">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton type="primary" @click.prevent="onUpdate">
            {{ $t("common.confirm-and-update") }}
          </NButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useEventListener } from "@vueuse/core";
import { NButton } from "naive-ui";
import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { onBeforeRouteLeave } from "vue-router";
import { useAppFeature, pushNotification } from "@/store";
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

const settingRefList = computed(() => {
  return [
    projectGeneralSettingPanelRef,
    projectSecuritySettingPanelRef,
    projectIssueRelatedSettingPanelRef,
  ];
});

const isDirty = computed(() => {
  return settingRefList.value.some((settingRef) => settingRef.value?.isDirty);
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

const onUpdate = async () => {
  for (const settingRef of settingRefList.value) {
    if (!settingRef.value?.isDirty) {
      continue;
    }
    try {
      await settingRef.value.update();
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("project.settings.update-failed"),
      });
      return;
    }
  }
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("project.settings.success-updated"),
  });
};

const onRevert = () => {
  for (const settingRef of settingRefList.value) {
    settingRef.value?.revert();
  }
};
</script>
