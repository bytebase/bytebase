<template>
  <NTag
    :class="[clickable && 'cursor-pointer']"
    :type="tagType"
    round
    @click="handleClick"
  >
    <template #icon>
      <template v-if="status === 'RUNNING'">
        <TaskSpinner class="h-4 w-4 text-info" />
      </template>
      <template v-else-if="status === 'SUCCESS'">
        <CheckIcon :size="16" class="text-success" />
      </template>
      <template v-else-if="status === 'WARNING'">
        <TriangleAlertIcon :size="16" />
      </template>
      <template v-else-if="status === 'ERROR'">
        <CircleAlertIcon :size="16" />
      </template>
    </template>
    <span>{{ $t("task.check-type.sql-review.self") }}</span>
  </NTag>

  <SQLCheckPanel
    v-if="showDetailPanel"
    :project="project.name"
    :database="database"
    :advices="advices"
    @close="showDetailPanel = false"
  />
</template>

<script setup lang="ts">
import { CheckIcon, CircleAlertIcon, TriangleAlertIcon } from "lucide-vue-next";
import { NTag } from "naive-ui";
import { computed, ref } from "vue";
import { TaskSpinner } from "@/components/IssueV1/components/common";
import { SQLCheckPanel } from "@/components/SQLCheck";
import type { Advice } from "@/types/proto-es/v1/sql_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { usePlanSQLCheckContext } from "./context";

const props = defineProps<{
  advices: Advice[];
  isRunning?: boolean;
}>();

defineEmits<{
  (event: "click"): void;
}>();

const { database, project } = usePlanSQLCheckContext();
const showDetailPanel = ref(false);

const status = computed(() => {
  const { isRunning, advices } = props;
  if (isRunning) {
    return "RUNNING";
  }
  if (advices.some((adv) => adv.status === Advice_Level.ERROR)) {
    return "ERROR";
  }
  if (advices.some((adv) => adv.status === Advice_Level.WARNING)) {
    return "WARNING";
  }
  return "SUCCESS";
});

const clickable = computed(() => {
  return status.value === "ERROR" || status.value === "WARNING";
});

const tagType = computed(() => {
  switch (status.value) {
    case "SUCCESS":
      return "default";
    case "RUNNING":
      return "info";
    case "WARNING":
      return "warning";
    case "ERROR":
      return "error";
  }
  // Should not reach here.
  return "default";
});

const handleClick = () => {
  showDetailPanel.value = true;
};
</script>
