<template>
  <div class="w-full space-y-0 pt-4">
    <div class="divide-y divide-block-border">
      <!-- General Settings Section -->
      <div class="pb-6 lg:flex">
        <div class="text-left lg:w-1/4">
          <h1 class="text-2xl font-bold">
            {{ $t("common.general") }}
          </h1>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0">
          <ProjectGeneralSettingPanel
            ref="projectGeneralSettingPanelRef"
            :project="project"
            :allow-edit="allowEdit"
          />
        </div>
      </div>

      <!-- Security Settings Section -->
      <div class="py-6 lg:flex">
        <div class="text-left lg:w-1/4">
          <h1 class="text-2xl font-bold">
            {{ $t("settings.sidebar.security-and-policy") }}
          </h1>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0">
          <ProjectSecuritySettingPanel
            ref="projectSecuritySettingPanelRef"
            :project="project"
            :allow-edit="allowEdit"
          />
        </div>
      </div>

      <div id="issue-related" class="py-6 lg:flex">
        <div class="text-left lg:w-1/4">
          <h1 class="text-2xl font-bold">
            {{ $t("project.settings.issue-related.self") }}
          </h1>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0">
          <ProjectIssueRelatedSettingPanel
            ref="projectIssueRelatedSettingPanelRef"
            :project="project"
            :allow-edit="allowEdit"
          />
        </div>
      </div>

      <div class="py-6 lg:flex">
        <ProjectArchiveRestoreButton :project="project" />
      </div>

      <!-- Save/Cancel buttons -->
      <div v-if="allowEdit && isDirty" class="sticky bottom-0 z-10">
        <div
          class="flex justify-between w-full py-4 border-block-border bg-white"
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
import { pushNotification } from "@/store";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import ProjectArchiveRestoreButton from "./Project/ProjectArchiveRestoreButton.vue";
import {
  ProjectGeneralSettingPanel,
  ProjectSecuritySettingPanel,
  ProjectIssueRelatedSettingPanel,
} from "./Project/Settings/";

defineProps<{
  project: Project;
  allowEdit: boolean;
}>();

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
