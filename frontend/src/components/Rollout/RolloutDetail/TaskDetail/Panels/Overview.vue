<template>
  <NAlert
    class="mb-3"
    v-if="taskRun?.schedulerInfo?.waitingCause?.task"
    type="warning"
  >
    <span>{{ $t("task-run.status.waiting-task") }}</span>
    <router-link
      class="inline-flex items-center normal-link shrink-0 ml-1"
      :to="`/${taskRun.schedulerInfo.waitingCause.task?.task}`"
      target="_blank"
    >
      {{ $t("common.blocking-task") }}
      <ExternalLinkIcon class="ml-1" :size="16" />
    </router-link>
  </NAlert>
  <p class="w-auto flex items-center mb-2">
    <span class="text-base text-main">{{ $t("common.statement") }}</span>
    <NButton class="ml-1" text @click.prevent="copyStatement">
      <template #icon>
        <ClipboardIcon class="w-4 h-4" />
      </template>
    </NButton>
  </p>
  <MonacoEditor
    class="h-auto max-h-[480px] min-h-[120px] border rounded-[3px] text-sm overflow-clip relative"
    :content="statement"
    :readonly="true"
    :auto-height="{ min: 256 }"
  />
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { ClipboardIcon, ExternalLinkIcon } from "lucide-vue-next";
import { NAlert, NButton } from "naive-ui";
import { computed, watchEffect } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { pushNotification, useSheetV1Store } from "@/store";
import { getSheetStatement, sheetNameOfTaskV1, toClipboard } from "@/utils";
import { useTaskDetailContext } from "../context";

const { task, taskRuns } = useTaskDetailContext();
const sheetStore = useSheetV1Store();

const statement = computed(() => {
  const sheet = sheetStore.getSheetByName(sheetNameOfTaskV1(task.value));
  if (sheet) {
    return getSheetStatement(sheet);
  }
  return "";
});

// The latest task run of the task.
const taskRun = computed(() => head(taskRuns.value));

const copyStatement = async () => {
  toClipboard(statement.value).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: `Statement copied to clipboard.`,
    });
  });
};

watchEffect(async () => {
  // Prepare the sheet for the task.
  const sheet = sheetNameOfTaskV1(task.value);
  if (sheet) {
    await sheetStore.getOrFetchSheetByName(sheet);
  }
});
</script>
