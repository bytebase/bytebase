<template>
  <NPopover v-if="showProgress" placement="bottom" :disabled="!showPopover">
    <template #trigger>
      <BBProgressPie
        class="w-10 h-10"
        :class="task.status === 'DONE' ? 'text-success' : 'text-info'"
        :thickness="3"
        :percent="progress.percent"
      >
        <template #default="{ percent }">
          <span v-if="task.status !== 'DONE'" class="text-xs">
            {{ percent }}%
          </span>
          <span v-else>
            <heroicons-outline:check class="w-8 h-8" />
          </span>
        </template>
      </BBProgressPie>
    </template>

    <div class="flex flex-col gap-y-2">
      <div class="flex flex-col items-start">
        <label class="textlabel">
          {{
            $t("task.progress.completed-units", {
              units: unitStr,
            })
          }}
        </label>
        <span>
          <slot name="unit" :unit="progress.completedUnit">
            {{ progress.completedUnit }}
          </slot>
        </span>
      </div>
      <div class="flex flex-col items-start">
        <label class="textlabel">
          {{
            $t("task.progress.total-units", {
              units: unitStr,
            })
          }}
        </label>
        <span v-if="progress.totalUnit > 0">
          <slot name="unit" :unit="progress.totalUnit">
            {{ progress.totalUnit }}
          </slot>
        </span>
        <span v-else class="text-gray-400">
          {{ $t("task.progress.counting") }}
        </span>
      </div>
      <div
        v-if="task.status === 'RUNNING' && progress.eta > 0"
        class="flex flex-col items-start whitespace-nowrap"
      >
        <label class="textlabel whitespace-nowrap">
          <span>{{ $t("task.progress.eta") }}</span>
          <span class="text-gray-400 text-xs">
            UTC{{ dayjs().format("ZZ") }}
          </span>
        </label>
        <span class="whitespace-nowrap">
          {{ dayjs(progress.eta * 1000).format("YYYY-MM-DD HH:mm:ss") }}
        </span>
      </div>
    </div>
  </NPopover>
</template>

<script lang="ts" setup>
import { NPopover } from "naive-ui";
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { BBProgressPie } from "@/bbkit";
import type { Task, TaskProgress } from "@/types";
import { empty } from "@/types";

type ProgressSummary = TaskProgress & {
  percent: number;
  eta: number;
};

const props = defineProps({
  task: {
    type: Object as PropType<Task>,
    required: true,
  },
  unitKey: {
    type: String,
    default: undefined,
  },
});

const { t } = useI18n();

const showProgress = computed((): boolean => {
  const { status } = props.task;

  return status === "PENDING" || status === "RUNNING";
});

const progress = computed((): ProgressSummary => {
  const ZERO: ProgressSummary = {
    ...empty("TASK_PROGRESS"),
    percent: 0,
    eta: 0,
  };

  const { task } = props;

  if (task.status === "DONE") {
    return { ...task.progress, percent: 100, eta: task.updatedTs };
  }

  const { progress } = task;
  if (progress.totalUnit > 0) {
    const completedUnit = progress.completedUnit;
    const totalUnit = Math.max(progress.completedUnit, progress.totalUnit);

    const p = completedUnit / totalUnit;
    const percent = Math.floor(p * 100);

    const result: ProgressSummary = {
      ...task.progress,
      totalUnit,
      percent,
      eta: 0,
    };

    const { updatedTs, createdTs } = progress;
    if (updatedTs > 0 && createdTs > 0 && p > 0) {
      const elapsedSeconds = progress.updatedTs - progress.createdTs;
      const estimatedTotalSeconds = Math.floor(elapsedSeconds / p);
      const eta = progress.createdTs + estimatedTotalSeconds;
      result.eta = eta;
    }

    return result;
  }

  return ZERO;
});

const showPopover = computed((): boolean => {
  return props.task.status !== "DONE";
});

const unitStr = computed(() => {
  const { unitKey } = props;
  if (!unitKey) {
    return "";
  }

  return t(`task.progress.units.${unitKey}`);
});
</script>
