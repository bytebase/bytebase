<template>
  <div class="w-full flex flex-row gap-4">
    <div class="flex items-center justify-between">
      <h3 class="textlabel">
        {{ $t("common.task", 2) }}
        <span>({{ taskList.length }})</span>
      </h3>
    </div>
    <div class="flex flex-row gap-2 items-center">
      <div
        class="bg-gray-50 pl-2 p-1 flex flex-row items-center rounded-full gap-1"
      >
        <span class="text-sm mr-1 text-gray-600">{{
          isCreating ? $t("issue.sql-check.sql-checks") : $t("task.task-checks")
        }}</span>
        <template v-for="status in ADVICE_STATUS_FILTERS" :key="status">
          <NTag
            v-if="getTaskCount(undefined, status) > 0"
            :disabled="disabled"
            :size="'small'"
            round
            checkable
            :checked="adviceStatusList.includes(status)"
            @update:checked="
              (checked) => {
                emit(
                  'update:adviceStatusList',
                  checked
                    ? [...adviceStatusList, status]
                    : adviceStatusList.filter((s) => s !== status)
                );
              }
            "
          >
            <template #avatar>
              <AdviceStatusIcon :status="status" />
            </template>
            <span class="select-none">{{
              getTaskCount(undefined, status)
            }}</span>
          </NTag>
        </template>
      </div>
      <div
        v-if="!isCreating"
        class="bg-gray-50 pl-2 p-1 flex flex-row items-center rounded-full gap-1"
      >
        <span class="text-sm mr-1 text-gray-600">{{
          $t("common.status")
        }}</span>
        <template v-for="status in TASK_STATUS_FILTERS" :key="status">
          <NTag
            v-if="getTaskCount(status) > 0"
            :disabled="disabled"
            :size="'small'"
            round
            checkable
            :checked="taskStatusList.includes(status)"
            @update:checked="
              (checked) => {
                emit(
                  'update:taskStatusList',
                  checked
                    ? [...taskStatusList, status]
                    : taskStatusList.filter((s) => s !== status)
                );
              }
            "
          >
            <template #avatar>
              <TaskStatusIconV1 :status="status" :size="'small'" />
            </template>
            <span class="select-none">{{ getTaskCount(status) }}</span>
          </NTag>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui";
import { computed } from "vue";
import AdviceStatusIcon from "@/components/Plan/components/SQLCheckSection/AdviceStatusIcon.vue";
import { usePlanSQLCheckContext } from "@/components/Plan/components/SQLCheckSection/context";
import { TASK_STATUS_FILTERS } from "@/components/Plan/constants/task";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { Advice_Status } from "@/types/proto-es/v1/sql_service_pb";
import { useIssueContext } from "../../logic";
import TaskStatusIconV1 from "../TaskStatusIconV1.vue";
import { filterTask } from "./filter";

defineProps<{
  disabled: boolean;
  taskStatusList: Task_Status[];
  adviceStatusList: Advice_Status[];
}>();

const emit = defineEmits<{
  (event: "update:taskStatusList", taskStatusList: Task_Status[]): void;
  (event: "update:adviceStatusList", adviceStatusList: Advice_Status[]): void;
}>();

// Using unified task status filters from constants
const ADVICE_STATUS_FILTERS: Advice_Status[] = [
  Advice_Status.STATUS_UNSPECIFIED,
  Advice_Status.SUCCESS,
  Advice_Status.WARNING,
  Advice_Status.ERROR,
];

const issueContext = useIssueContext();
const { resultMap } = usePlanSQLCheckContext();

const { isCreating, selectedStage } = issueContext;

const taskList = computed(() => selectedStage.value.tasks);

const getTaskCount = (status?: Task_Status, adviceStatus?: Advice_Status) => {
  return taskList.value.filter((task) =>
    filterTask(issueContext, resultMap.value, task, { status, adviceStatus })
  ).length;
};
</script>
