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

  <div v-if="revision?.taskRun" class="my-4">
    <p class="w-auto flex items-center text-base text-main mb-2">
      {{ $t("issue.task-run.logs") }}
    </p>
    <TaskRunLogViewer :task-run-name="revision.taskRun" />
  </div>
</template>

<script lang="ts" setup>
import { NSpin } from "naive-ui";
import { computed, ref, watch } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { TaskRunLogViewer } from "@/components/RolloutV1/components/TaskRunLogViewer";
import { CopyButton } from "@/components/v2";
import { sheetServiceClientConnect } from "@/connect";
import { useRevisionStore } from "@/store";
import { type ComposedDatabase, getDateForPbTimestampProtoEs } from "@/types";
import HumanizeDate from "../misc/HumanizeDate.vue";

const props = defineProps<{
  database: ComposedDatabase;
  revisionName: string;
}>();

const loading = ref(false);
const fetchedStatement = ref("");

const revisionStore = useRevisionStore();

watch(
  () => props.revisionName,
  async (revisionName) => {
    if (!revisionName) {
      return;
    }

    loading.value = true;
    fetchedStatement.value = "";

    try {
      const revision =
        await revisionStore.getOrFetchRevisionByName(revisionName);
      if (revision) {
        // Prepare the sheet data for statement display
        if (revision.sheet) {
          try {
            const sheet = await sheetServiceClientConnect.getSheet({
              name: revision.sheet,
              raw: true,
            });
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

const revision = computed(() =>
  revisionStore.getRevisionByName(props.revisionName)
);
</script>
