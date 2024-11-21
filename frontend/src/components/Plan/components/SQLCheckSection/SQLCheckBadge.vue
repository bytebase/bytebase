<template>
  <button
    class="inline-flex items-center px-3 py-0.5 rounded-full text-sm border border-transparent"
    :class="buttonClasses"
    @click="handleClick"
  >
    <template v-if="status === 'RUNNING'">
      <TaskSpinner class="-ml-1 mr-1.5 h-4 w-4 text-info" />
    </template>
    <template v-else-if="status === 'SUCCESS'">
      <heroicons-outline:check
        class="-ml-1 mr-1.5 mt-0.5 h-4 w-4 text-success"
      />
    </template>
    <template v-else-if="status === 'WARNING'">
      <heroicons-outline:exclamation
        class="-ml-1 mr-1.5 mt-0.5 h-4 w-4 text-warning"
      />
    </template>
    <template v-else-if="status === 'ERROR'">
      <span class="mr-1.5 font-medium text-error" aria-hidden="true"> ! </span>
    </template>

    <span>{{ $t("task.check-type.sql-review") }}</span>

    <SQLCheckPanel
      v-if="showDetailPanel"
      :database="database"
      :advices="advices"
      @close="showDetailPanel = false"
    >
    </SQLCheckPanel>
  </button>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { TaskSpinner } from "@/components/IssueV1/components/common";
import { SQLCheckPanel } from "@/components/SQLCheck";
import type { Advice } from "@/types/proto/v1/sql_service";
import { Advice_Status } from "@/types/proto/v1/sql_service";
import { databaseForSpec, usePlanContext } from "../../logic";

const props = defineProps<{
  isRunning: boolean;
  advices: Advice[];
}>();

defineEmits<{
  (event: "click"): void;
}>();

const { plan, selectedSpec } = usePlanContext();
const showDetailPanel = ref(false);

const database = computed(() => {
  return databaseForSpec(plan.value, selectedSpec.value);
});

const status = computed(() => {
  const { isRunning, advices } = props;
  if (isRunning) {
    return "RUNNING";
  }
  if (advices.some((adv) => adv.status === Advice_Status.ERROR)) {
    return "ERROR";
  }
  if (advices.some((adv) => adv.status === Advice_Status.WARNING)) {
    return "WARNING";
  }
  return "SUCCESS";
});

const clickable = computed(() => {
  return status.value === "ERROR" || status.value === "WARNING";
});

const buttonClasses = computed(() => {
  let bgColor = "";
  let bgHoverColor = "";
  let textColor = "";
  let borderColor = "";
  switch (status.value) {
    case "RUNNING":
      bgColor = "bg-blue-100";
      bgHoverColor = "bg-blue-300";
      textColor = "text-blue-800";
      borderColor = "border-blue-800";
      break;
    case "SUCCESS":
      bgColor = "bg-gray-100";
      bgHoverColor = "bg-gray-300";
      textColor = "text-gray-800";
      borderColor = "border-gray-800";
      break;
    case "WARNING":
      bgColor = "bg-yellow-100";
      bgHoverColor = "bg-yellow-300";
      textColor = "text-yellow-800";
      borderColor = "border-yellow-800";
      break;
    case "ERROR":
      bgColor = "bg-red-100";
      bgHoverColor = "bg-red-300";
      textColor = "text-red-800";
      borderColor = "border-red-800";
      break;
  }

  const styleList: string[] = [textColor, bgColor];
  if (clickable.value) {
    styleList.push(
      "cursor-pointer",
      `hover:${bgHoverColor}`,
      `hover:${borderColor}`
    );
  } else {
    styleList.push("cursor-default");
  }

  return styleList.join(" ");
});

const handleClick = () => {
  showDetailPanel.value = true;
};
</script>
