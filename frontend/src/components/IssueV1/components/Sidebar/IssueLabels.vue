<template>
  <div class="flex flex-col gap-y-1">
    <div class="flex items-center gap-x-1 textlabel">
      <span>{{ $t("issue.labels") }}</span>
      <RequiredStar v-if="project.forceIssueLabels" />
    </div>
    <IssueLabelSelector
      :disabled="disabled"
      :selected="value"
      :project="project"
      :size="'medium'"
      @update:selected="onLablesUpdate"
    />
  </div>
</template>

<script setup lang="ts">
import IssueLabelSelector from "@/components/IssueV1/components/IssueLabelSelector.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

withDefaults(
  defineProps<{
    value: string[];
    project: Project;
    disabled?: boolean;
  }>(),
  {
    disabled: false,
  }
);

const emit = defineEmits<{
  (e: "update:value", labels: string[]): void;
}>();

const onLablesUpdate = async (labels: string[]) => {
  emit("update:value", labels);
};
</script>
