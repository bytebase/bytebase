<template>
  <div class="w-full flex flex-col justify-start items-start pt-6 space-y-4">
    <h3 class="text-lg font-medium leading-7 text-main">
      {{ $t("settings.sidebar.security-and-policy") }}
    </h3>
    <SQLReviewForResource
      v-if="databaseChangeMode === DatabaseChangeMode.PIPELINE"
      ref="sqlReviewForResourceRef"
      :resource="project.name"
      :allow-edit="allowEdit"
    />
    <RestrictIssueCreationConfigure
      v-if="databaseChangeMode === DatabaseChangeMode.PIPELINE"
      ref="restrictIssueCreationConfigureRef"
      :resource="project.name"
      :allow-edit="allowEdit"
    />
    <AccessControlConfigure
      ref="accessControlConfigureRef"
      :resource="project.name"
      :allow-edit="allowEdit"
    />
    <div v-if="allowEdit" class="w-full flex justify-end">
      <NButton type="primary" :disabled="!isDirty" @click.prevent="onUpdate">
        {{ $t("common.update") }}
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import AccessControlConfigure from "@/components/EnvironmentForm/AccessControlConfigure.vue";
import RestrictIssueCreationConfigure from "@/components/GeneralSetting/RestrictIssueCreationConfigure.vue";
import { SQLReviewForResource } from "@/components/SQLReview";
import { useAppFeature, pushNotification } from "@/store";
import type { ComposedProject } from "@/types";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";

defineProps<{
  project: ComposedProject;
  allowEdit: boolean;
}>();

const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");
const { t } = useI18n();

const restrictIssueCreationConfigureRef =
  ref<InstanceType<typeof RestrictIssueCreationConfigure>>();
const accessControlConfigureRef =
  ref<InstanceType<typeof AccessControlConfigure>>();
const sqlReviewForResourceRef =
  ref<InstanceType<typeof SQLReviewForResource>>();

const isDirty = computed(
  () =>
    restrictIssueCreationConfigureRef.value?.isDirty ||
    accessControlConfigureRef.value?.isDirty ||
    sqlReviewForResourceRef.value?.isDirty
);

const onUpdate = async () => {
  if (restrictIssueCreationConfigureRef.value?.isDirty) {
    await restrictIssueCreationConfigureRef.value.update();
  }
  if (sqlReviewForResourceRef.value?.isDirty) {
    await sqlReviewForResourceRef.value.update();
  }
  if (accessControlConfigureRef.value?.isDirty) {
    await accessControlConfigureRef.value.update();
  }
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
};

defineExpose({
  isDirty,
});
</script>
