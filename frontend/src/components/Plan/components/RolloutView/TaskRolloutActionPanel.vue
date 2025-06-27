<template>
  <CommonDrawer
    :show="action !== undefined"
    :title="title"
    :loading="state.loading"
    @show="resetState"
    @close="$emit('close')"
  >
    <template #default>
      <div
        v-if="action"
        class="flex flex-col gap-y-4 h-full overflow-y-hidden px-1"
      >
        <div
          class="flex flex-row gap-x-2 shrink-0 overflow-y-hidden justify-start items-center"
        >
          <label class="font-medium text-control">
            {{ $t("common.stage") }}
          </label>
          <span class="break-all">
            {{ environmentStore.getEnvironmentByName(stage.environment).title }}
          </span>
        </div>

        <div
          class="flex flex-col gap-y-1 shrink overflow-y-hidden justify-start"
        >
          <label class="font-medium text-control">
            <template v-if="runnableTasks.length === 1">
              {{ $t("common.task") }}
            </template>
            <template v-else>{{ $t("common.tasks") }}</template>
            <span class="opacity-80" v-if="runnableTasks.length > 1"
              >({{ runnableTasks.length }})</span
            >
          </label>
          <div class="flex-1 overflow-y-auto">
            <template v-if="useVirtualScroll">
              <NVirtualList
                :items="runnableTasks"
                :item-size="itemHeight"
                class="max-h-64"
                item-resizable
              >
                <template #default="{ item: task }">
                  <div
                    :key="task.name"
                    class="flex items-center text-sm"
                    :style="{ height: `${itemHeight}px` }"
                  >
                    <NTag
                      v-if="semanticTaskType(task.type)"
                      class="mr-2"
                      size="small"
                    >
                      <span class="inline-block text-center">
                        {{ semanticTaskType(task.type) }}
                      </span>
                    </NTag>
                    <TaskDatabaseName :task="task" />
                  </div>
                </template>
              </NVirtualList>
            </template>
            <template v-else>
              <NScrollbar class="max-h-64">
                <ul class="text-sm space-y-2">
                  <li
                    v-for="task in runnableTasks"
                    :key="task.name"
                    class="flex items-center"
                  >
                    <NTag
                      v-if="semanticTaskType(task.type)"
                      class="mr-2"
                      size="small"
                    >
                      <span class="inline-block text-center">
                        {{ semanticTaskType(task.type) }}
                      </span>
                    </NTag>
                    <TaskDatabaseName :task="task" />
                  </li>
                </ul>
              </NScrollbar>
            </template>
          </div>
        </div>

        <div
          v-if="showScheduledTimePicker"
          class="flex flex-col gap-y-3 shrink-0"
        >
          <div class="flex items-center">
            <NCheckbox
              :checked="runTimeInMS === undefined"
              @update:checked="
                (checked) =>
                  (runTimeInMS = checked
                    ? undefined
                    : Date.now() + DEFAULT_RUN_DELAY_MS)
              "
            >
              {{ $t("task.run-immediately") }}
            </NCheckbox>
          </div>
          <div v-if="runTimeInMS !== undefined" class="flex flex-col gap-y-1">
            <p class="font-medium text-control">
              {{ $t("task.scheduled-time", runnableTasks.length) }}
            </p>
            <NDatePicker
              v-model:value="runTimeInMS"
              type="datetime"
              :placeholder="$t('task.select-scheduled-time')"
              :is-date-disabled="
                (date: number) => date < dayjs().startOf('day').valueOf()
              "
              format="yyyy-MM-dd HH:mm:ss"
              clearable
            />
          </div>
        </div>

        <div class="flex flex-col gap-y-1 shrink-0">
          <p class="font-medium text-control">
            {{ $t("common.comment") }}
          </p>
          <NInput
            v-model:value="comment"
            type="textarea"
            :placeholder="$t('issue.leave-a-comment')"
            :autosize="{
              minRows: 3,
              maxRows: 10,
            }"
          />
        </div>
      </div>
    </template>
    <template #footer>
      <div
        v-if="action"
        class="w-full flex flex-row justify-between items-center gap-x-2"
      >
        <div class="flex justify-end gap-x-3 w-full">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>

          <NTooltip :disabled="confirmErrors.length === 0" placement="top">
            <template #trigger>
              <NButton
                :disabled="confirmErrors.length > 0"
                type="primary"
                @click="handleConfirm"
              >
                {{ $t("common.rollout") }}
              </NButton>
            </template>
            <template #default>
              <ErrorList :errors="confirmErrors" />
            </template>
          </NTooltip>
        </div>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import Long from "long";
import {
  NButton,
  NCheckbox,
  NDatePicker,
  NInput,
  NScrollbar,
  NTag,
  NTooltip,
  NVirtualList,
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { semanticTaskType } from "@/components/IssueV1";
import CommonDrawer from "@/components/IssueV1/components/Panel/CommonDrawer.vue";
import { ErrorList } from "@/components/IssueV1/components/common";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { pushNotification, useEnvironmentV1Store } from "@/store";
import { BatchRunTasksRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { Stage } from "@/types/proto/v1/rollout_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import TaskDatabaseName from "./TaskDatabaseName.vue";

// Default delay for running tasks if not scheduled immediately.
const DEFAULT_RUN_DELAY_MS = 60000;

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  action?: "RUN_TASKS";
  stage: Stage;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  loading: false,
});
const environmentStore = useEnvironmentV1Store();
const comment = ref("");
const runTimeInMS = ref<number | undefined>(undefined);

const title = computed(() => {
  if (!props.action) return "";
  return t("common.rollout");
});

const runnableTasks = computed(() => {
  return props.stage.tasks.filter(
    (task) =>
      task.status === Task_Status.NOT_STARTED ||
      task.status === Task_Status.PENDING ||
      task.status === Task_Status.FAILED
  );
});

// Virtual scroll configuration
const useVirtualScroll = computed(() => runnableTasks.value.length > 50);
const itemHeight = 32; // Height of each task item in pixels

const showScheduledTimePicker = computed(() => {
  return props.action === "RUN_TASKS";
});

const confirmErrors = computed(() => {
  const errors: string[] = [];

  if (runnableTasks.value.length === 0) {
    errors.push(t("common.no-data"));
  }

  // Validate scheduled time if not running immediately
  if (runTimeInMS.value !== undefined) {
    if (runTimeInMS.value <= Date.now()) {
      errors.push(t("task.error.scheduled-time-must-be-in-the-future"));
    }
  }

  return errors;
});

const handleConfirm = async () => {
  state.loading = true;
  try {
    // Prepare the request parameters
    const requestParams: any = {
      parent: props.stage.name,
      tasks: runnableTasks.value.map((task) => task.name),
      reason: comment.value,
    };

    if (runTimeInMS.value !== undefined) {
      // Convert timestamp to protobuf Timestamp format
      const runTimeSeconds = Math.floor(runTimeInMS.value / 1000);
      const runTimeNanos = (runTimeInMS.value % 1000) * 1000000;
      requestParams.runTime = {
        seconds: Long.fromNumber(runTimeSeconds),
        nanos: runTimeNanos,
      };
    }

    const request = create(BatchRunTasksRequestSchema, requestParams);
    await rolloutServiceClientConnect.batchRunTasks(request);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.success"),
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: String(error),
    });
  } finally {
    state.loading = false;
    emit("close");
  }
};

const resetState = () => {
  comment.value = "";
  runTimeInMS.value = undefined;
};
</script>
