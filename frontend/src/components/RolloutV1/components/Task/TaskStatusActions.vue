<template>
  <div>
    <div
      v-if="!readonly && hasActions"
      class="flex flex-row justify-end items-center gap-x-2"
    >
      <NButton v-if="primaryAction" :size="size" @click="handlePrimaryAction">
        {{ primaryActionLabel }}
      </NButton>
      <NDropdown
        v-if="dropdownOptions.length > 0"
        trigger="hover"
        :options="dropdownOptions"
        @select="handleDropdownSelect"
      >
        <NButton :size="size" class="px-1!" quaternary>
          <template #icon>
            <EllipsisVerticalIcon class="w-4 h-4" />
          </template>
        </NButton>
      </NDropdown>
    </div>

    <template v-if="currentAction && actionTarget">
      <TaskRolloutActionPanel
        :show="showActionPanel"
        :action="currentAction"
        :target="actionTarget"
        @close="handleActionPanelClose"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { EllipsisVerticalIcon } from "lucide-vue-next";
import { NButton, NDropdown } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { usePlanContextWithRollout } from "@/components/Plan/logic";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import TaskRolloutActionPanel from "./TaskRolloutActionPanel.vue";
import { useTaskActions } from "./useTaskActions";

const props = withDefaults(
  defineProps<{
    task: Task;
    stage: Stage;
    size?: "tiny" | "small" | "medium" | "large";
  }>(),
  {
    size: "small",
  }
);

const emit = defineEmits<{
  "action-confirmed": [];
}>();

const { t } = useI18n();
const { readonly } = usePlanContextWithRollout();

const {
  canRun,
  canSkip,
  canCancel,
  hasActions,
  showActionPanel,
  currentAction,
  actionTarget,
  runTask,
  skipTask,
  cancelTask,
  closeActionPanel,
} = useTaskActions(
  () => props.task,
  () => props.stage
);

const primaryAction = computed(() => {
  if (canRun.value) {
    if (props.task.status === Task_Status.FAILED) return "RETRY";
    return "RUN";
  }
  return null;
});

const primaryActionLabel = computed(() => {
  if (primaryAction.value === "RETRY") return t("common.retry");
  if (primaryAction.value === "RUN") return t("common.run");
  return "";
});

const dropdownOptions = computed(() => {
  const options: { key: string; label: string }[] = [];
  if (canSkip.value) {
    options.push({ key: "SKIP", label: t("common.skip") });
  }
  if (canCancel.value) {
    options.push({ key: "CANCEL", label: t("common.cancel") });
  }
  return options;
});

const handlePrimaryAction = () => {
  runTask();
};

const handleDropdownSelect = (key: string) => {
  if (key === "SKIP") skipTask();
  else if (key === "CANCEL") cancelTask();
};

const handleActionPanelClose = () => {
  closeActionPanel();
  emit("action-confirmed");
};
</script>
