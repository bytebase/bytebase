<template>
  <div class="w-full flex flex-col justify-start items-start gap-y-6">
    <SQLReviewForResource
      ref="sqlReviewForResourceRef"
      :resource="project.name"
    />
    <AccessControlConfigure
      ref="accessControlConfigureRef"
      :resource="project.name"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import AccessControlConfigure from "@/components/EnvironmentForm/AccessControlConfigure.vue";
import { SQLReviewForResource } from "@/components/SQLReview";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

defineProps<{
  project: Project;
}>();

const accessControlConfigureRef =
  ref<InstanceType<typeof AccessControlConfigure>>();
const sqlReviewForResourceRef =
  ref<InstanceType<typeof SQLReviewForResource>>();

const isDirty = computed(
  () =>
    accessControlConfigureRef.value?.isDirty ||
    sqlReviewForResourceRef.value?.isDirty
);

const onUpdate = async () => {
  if (sqlReviewForResourceRef.value?.isDirty) {
    await sqlReviewForResourceRef.value.update();
  }
  if (accessControlConfigureRef.value?.isDirty) {
    await accessControlConfigureRef.value.update();
  }
};

const resetState = () => {
  sqlReviewForResourceRef.value?.revert();
  accessControlConfigureRef.value?.revert();
};

defineExpose({
  isDirty,
  update: onUpdate,
  revert: resetState,
});
</script>
