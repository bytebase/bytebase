<template>
  <div class="w-full flex flex-col justify-start items-start gap-y-6">
    <SQLReviewForResource
      ref="sqlReviewForResourceRef"
      :resource="project.name"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { SQLReviewForResource } from "@/components/SQLReview";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

defineProps<{
  project: Project;
}>();

const sqlReviewForResourceRef =
  ref<InstanceType<typeof SQLReviewForResource>>();

const isDirty = computed(() => sqlReviewForResourceRef.value?.isDirty);

const onUpdate = async () => {
  if (sqlReviewForResourceRef.value?.isDirty) {
    await sqlReviewForResourceRef.value.update();
  }
};

const resetState = () => {
  sqlReviewForResourceRef.value?.revert();
};

defineExpose({
  isDirty,
  update: onUpdate,
  revert: resetState,
});
</script>
