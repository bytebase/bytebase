<template>
  <div class="focus:outline-hidden" tabindex="0" v-bind="$attrs">
    <div
      v-if="state.loading"
      class="flex items-center justify-center py-2 text-gray-400 text-sm"
    >
      <BBSpin />
    </div>
    <main v-else-if="revision" class="flex flex-col relative gap-y-6">
      <!-- Highlight Panel -->
      <div class="flex flex-col gap-y-4">
        <!-- Version Title -->
        <h2 class="text-2xl font-semibold text-main">
          {{ revision.version }}
        </h2>

        <!-- Metadata Row -->
        <div class="flex items-center gap-x-3 text-sm text-control-light">
          <span>{{ getRevisionType(revision.type) }}</span>
          <span v-if="formattedCreateTime">•</span>
          <span v-if="formattedCreateTime">
            {{ formattedCreateTime }}
          </span>
        </div>
      </div>

      <div class="flex flex-col gap-y-6">
        <!-- Task Run Logs Section -->
        <div v-if="revision.taskRun" class="flex flex-col gap-y-2">
          <div class="flex items-center justify-between">
            <p class="text-lg text-main">
              {{ $t("issue.task-run.logs") }}
            </p>
            <router-link
              v-if="taskFullLink"
              :to="taskFullLink"
              class="flex items-center gap-x-1 text-sm text-control-light hover:text-accent transition-colors"
            >
              {{ $t("common.show-more") }}
              <ArrowUpRightIcon class="w-4 h-4" />
            </router-link>
          </div>
          <TaskRunLogViewer :task-run-name="revision.taskRun" />
        </div>

        <!-- Statement Section -->
        <div class="flex flex-col gap-y-2">
          <p class="flex items-center text-lg text-main gap-x-2">
            {{ $t("common.statement") }}
            <span
              v-if="formattedStatementSize"
              class="text-sm font-normal text-control-light"
            >
              ({{ formattedStatementSize }})
            </span>
            <CopyButton size="small" :content="state.statement" />
          </p>
          <MonacoEditor
            class="h-auto max-h-[600px] min-h-[120px] border rounded-md text-sm overflow-clip relative"
            :content="state.statement"
            :readonly="true"
            :auto-height="{ min: 120, max: 600 }"
          />
        </div>
      </div>
    </main>
  </div>
</template>

<script lang="ts" setup>
import { ArrowUpRightIcon } from "lucide-vue-next";
import { computed, reactive, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { MonacoEditor } from "@/components/MonacoEditor";
import { TaskRunLogViewer } from "@/components/RolloutV1/components/TaskRun/TaskRunLogViewer";
import { CopyButton } from "@/components/v2";
import { sheetServiceClientConnect } from "@/connect";
import { useRevisionStore } from "@/store";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import { bytesToString } from "@/utils";
import { extractTaskLink, getRevisionType } from "@/utils/v1/revision";

interface LocalState {
  loading: boolean;
  statement: string;
}

const props = defineProps<{
  database: Database;
  revisionName: string;
}>();

const revisionStore = useRevisionStore();
const state = reactive<LocalState>({
  loading: false,
  statement: "",
});

const revision = computed((): Revision | undefined =>
  revisionStore.getRevisionByName(props.revisionName)
);

const taskFullLink = computed(() => {
  if (!revision.value?.taskRun) {
    return "";
  }
  return extractTaskLink(revision.value.taskRun);
});

const formattedCreateTime = computed(() => {
  if (!revision.value) {
    return "";
  }
  return getDateForPbTimestampProtoEs(
    revision.value.createTime
  )?.toLocaleString();
});

const formattedStatementSize = computed(() => {
  if (!state.statement) {
    return "";
  }
  return bytesToString(new TextEncoder().encode(state.statement).length);
});

watch(
  () => props.revisionName,
  async (revisionName) => {
    if (!revisionName) {
      return;
    }

    state.loading = true;
    state.statement = "";

    try {
      const rev = await revisionStore.getOrFetchRevisionByName(revisionName);
      if (rev?.sheet) {
        try {
          const sheet = await sheetServiceClientConnect.getSheet({
            name: rev.sheet,
            raw: true,
          });
          if (sheet.content) {
            state.statement = new TextDecoder().decode(sheet.content);
          }
        } catch (error) {
          console.error("Failed to fetch sheet content", error);
        }
      }
    } catch (error) {
      console.error("Failed to fetch revision details", error);
    } finally {
      state.loading = false;
    }
  },
  { immediate: true }
);
</script>
