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
      <CopyButton :content="fetchedStatement" />
    </p>
    <div class="relative">
      <NSpin v-if="loading" :show="loading" class="absolute inset-0 z-10" />
      <MonacoEditor
        class="h-auto max-h-[480px] min-h-[120px] border rounded-[3px] text-sm overflow-clip relative"
        :content="fetchedStatement"
        :readonly="true"
        :auto-height="{ min: 120, max: 480 }"
      />
    </div>
  </div>

  <div v-if="logEntries.length > 0" class="my-4">
    <p class="w-auto flex items-center text-base text-main mb-2">
      {{ $t("issue.task-run.logs") }}
    </p>
    <TaskRunLogViewer :entries="logEntries" :sheet="fetchedSheet" />
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { computedAsync } from "@vueuse/core";
import { NSpin } from "naive-ui";
import { computed, ref, watch } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { TaskRunLogViewer } from "@/components/RolloutV1/components/TaskRunLogViewer";
import { CopyButton } from "@/components/v2";
import {
  rolloutServiceClientConnect,
  sheetServiceClientConnect,
} from "@/connect";
import { useRevisionStore } from "@/store";
import { type ComposedDatabase, getDateForPbTimestampProtoEs } from "@/types";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import {
  GetTaskRunLogRequestSchema,
  GetTaskRunRequestSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import HumanizeDate from "../misc/HumanizeDate.vue";

const props = defineProps<{
  database: ComposedDatabase;
  revisionName: string;
}>();

import { type Sheet } from "@/types/proto-es/v1/sheet_service_pb";

const loading = ref(false);
const fetchedStatement = ref("");
const fetchedSheet = ref<Sheet | undefined>(undefined);

const revisionStore = useRevisionStore();
const taskRun = ref<TaskRun | undefined>(undefined);

watch(
  () => props.revisionName,
  async (revisionName) => {
    if (!revisionName) {
      return;
    }

    loading.value = true;
    fetchedStatement.value = "";
    fetchedSheet.value = undefined;

    try {
      const revision =
        await revisionStore.getOrFetchRevisionByName(revisionName);
      if (revision) {
        if (revision.taskRun) {
          // Fetch the task run details.
          const request = create(GetTaskRunRequestSchema, {
            name: revision.taskRun,
          });
          const response =
            await rolloutServiceClientConnect.getTaskRun(request);
          taskRun.value = response;
        }
        // Prepare the sheet data from task run.
        // We fetch raw content directly.
        if (revision.sheet) {
          try {
            const sheet = await sheetServiceClientConnect.getSheet({
              name: revision.sheet,
              raw: true,
            });
            fetchedSheet.value = sheet;
            if (sheet.content) {
              fetchedStatement.value = new TextDecoder().decode(sheet.content);
            }
          } catch (error) {
            console.error("Failed to fetch sheet content", error);
          }
        }
      }
    } catch (error) {
      console.error("Failed to fetch revision details", error);
    } finally {
      loading.value = false;
    }
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
</script>
