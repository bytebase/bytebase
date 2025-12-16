<template>
  <div v-if="revision" class="w-full">
    <div class="flex flex-row items-center gap-2">
      <p class="text-lg flex gap-x-1">
        <span class="text-control">{{ $t("common.version") }}:</span>
        <span class="font-bold text-main">{{ revision.version }}</span>
      </p>
    </div>
    <div
      class="mt-3 text-control text-base flex flex-row items-center flex-wrap gap-x-4"
    >
      <span>
        {{ $t("database.revision.applied-at") }}:
        <HumanizeDate
          :date="getDateForPbTimestampProtoEs(revision.createTime)"
        />
      </span>
    </div>
  </div>

  <div class="flex flex-col my-4">
    <p class="w-auto flex items-center text-base text-main mb-2 gap-x-2">
      <span>{{ $t("common.statement") }}</span>
      <CopyButton :content="statement" />
    </p>
    <MonacoEditor
      class="h-auto max-h-[480px] min-h-[120px] border rounded-[3px] text-sm overflow-clip relative"
      :content="statement"
      :readonly="true"
      :auto-height="{ min: 120, max: 480 }"
    />
  </div>

  <div v-if="logEntries.length > 0" class="my-4">
    <p class="w-auto flex items-center text-base text-main mb-2">
      {{ $t("issue.task-run.logs") }}
    </p>
    <TaskRunLogViewer :entries="logEntries" :sheet="sheet" />
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { computedAsync } from "@vueuse/core";
import { computed, reactive, ref, watch } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { CopyButton } from "@/components/v2";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { useRevisionStore, useSheetV1Store } from "@/store";
import { type ComposedDatabase, getDateForPbTimestampProtoEs } from "@/types";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import {
  GetTaskRunLogRequestSchema,
  GetTaskRunRequestSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { getSheetStatement } from "@/utils";
import HumanizeDate from "../misc/HumanizeDate.vue";
import { TaskRunLogViewer } from "../Plan/components/RolloutView/v2/TaskRunLogViewer";

interface LocalState {
  loading: boolean;
}

const props = defineProps<{
  database: ComposedDatabase;
  revisionName: string;
}>();

const state = reactive<LocalState>({
  loading: false,
});

const revisionStore = useRevisionStore();
const sheetStore = useSheetV1Store();
const taskRun = ref<TaskRun | undefined>(undefined);

watch(
  () => props.revisionName,
  async (revisionName) => {
    if (!revisionName) {
      return;
    }

    state.loading = true;
    const revision = await revisionStore.getOrFetchRevisionByName(revisionName);
    if (revision) {
      if (revision.taskRun) {
        // Fetch the task run details.
        const request = create(GetTaskRunRequestSchema, {
          name: revision.taskRun,
        });
        const response = await rolloutServiceClientConnect.getTaskRun(request);
        taskRun.value = response;
      }
      // Prepare the sheet data from task run.
      if (revision.sheet) {
        await sheetStore.getOrFetchSheetByName(revision.sheet);
      }
    }
    state.loading = false;
  },
  { immediate: true }
);

// Fetch task run log
const taskRunLog = computedAsync(async () => {
  if (!taskRun.value?.name) return undefined;
  const request = create(GetTaskRunLogRequestSchema, {
    parent: taskRun.value.name,
  });
  return await rolloutServiceClientConnect.getTaskRunLog(request);
}, undefined);

const logEntries = computed(() => taskRunLog.value?.entries ?? []);

const revision = computed(() =>
  revisionStore.getRevisionByName(props.revisionName)
);

const sheet = computed(() =>
  revision.value?.sheet
    ? sheetStore.getSheetByName(revision.value.sheet)
    : undefined
);

const statement = computed(() =>
  sheet.value ? getSheetStatement(sheet.value) : ""
);
</script>
