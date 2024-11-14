<template>
  <p class="w-auto flex items-center text-base text-main mb-2">
    {{ $t("common.statement") }}
    <button tabindex="-1" class="btn-icon ml-1" @click.prevent="copyStatement">
      <ClipboardIcon class="w-4 h-4" />
    </button>
  </p>
  <MonacoEditor
    class="h-auto max-h-[480px] min-h-[120px] border rounded-[3px] text-sm overflow-clip relative"
    :content="statement"
    :readonly="true"
    :auto-height="{ min: 256 }"
  />
</template>

<script lang="ts" setup>
import { ClipboardIcon } from "lucide-vue-next";
import { computed, watchEffect } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { pushNotification, useSheetV1Store } from "@/store";
import type { Task, TaskRun } from "@/types/proto/v1/rollout_service";
import { getSheetStatement, sheetNameOfTaskV1, toClipboard } from "@/utils";

const props = defineProps<{
  task: Task;
  latestTaskRun?: TaskRun;
}>();

const sheetStore = useSheetV1Store();

const statement = computed(() => {
  const sheet = sheetStore.getSheetByName(sheetNameOfTaskV1(props.task));
  if (sheet) {
    return getSheetStatement(sheet);
  }
  return "";
});

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
  const sheet = sheetNameOfTaskV1(props.task);
  if (sheet) {
    await sheetStore.getOrFetchSheetByName(sheet);
  }
});
</script>
