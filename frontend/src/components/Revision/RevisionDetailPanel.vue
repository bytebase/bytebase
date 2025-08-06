<template>
  <div v-if="revision" class="w-full">
    <div class="flex flex-row items-center gap-2">
      <p class="text-lg space-x-1">
        <span class="text-control">{{ $t("common.version") }}:</span>
        <span class="font-bold text-main">{{ revision.version }}</span>
      </p>
    </div>
    <div
      class="mt-3 text-control text-base space-x-4 flex flex-row items-center flex-wrap"
    >
      <span>
        {{ $t("database.revision.applied-at") }}:
        <HumanizeDate
          :date="getDateForPbTimestampProtoEs(revision.createTime)"
        />
      </span>
      <span v-if="relatedIssueUID">
        {{ $t("common.issue") }}:
        <RouterLink class="normal-link" :to="`/${revision.issue}`"
          >#{{ relatedIssueUID }}</RouterLink
        >
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

  <div v-if="taskRun" class="my-4">
    <p class="w-auto flex items-center text-base text-main mb-2">
      {{ $t("issue.task-run.logs") }}
    </p>
    <TaskRunLogTable :task-run="taskRun" :sheet="sheet" />
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { computed, reactive, ref, watch } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { CopyButton } from "@/components/v2";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { useRevisionStore, useSheetV1Store } from "@/store";
import { getDateForPbTimestampProtoEs, type ComposedDatabase } from "@/types";
import { GetTaskRunRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { extractIssueUID, getSheetStatement } from "@/utils";
import TaskRunLogTable from "../IssueV1/components/TaskRunSection/TaskRunLogTable/TaskRunLogTable.vue";
import HumanizeDate from "../misc/HumanizeDate.vue";

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
        await sheetStore.getOrFetchSheetByName(revision.sheet, "FULL");
      }
    }
    state.loading = false;
  },
  { immediate: true }
);

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

const relatedIssueUID = computed(() => {
  const uid = extractIssueUID(revision.value?.issue || "");
  if (!uid) return null;
  return uid;
});
</script>
